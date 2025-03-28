package transaction

import (
	"InsAutoMessage/config"
	DB "InsAutoMessage/database"
	"InsAutoMessage/utils"
	"context"
	"fmt"
	"github.com/chromedp/chromedp"
	"log"
	"net/url"
	"runtime/debug"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

var (
	activeSearchWorkers int32 // 当前活跃的worker数量
	workerWaitGroup     sync.WaitGroup
	uniqueIDs           sync.Map
	scrollLock          sync.Mutex

	locationsMap sync.Map // 用于标记正在处理的地点
)

// 智能解析动态内容并返回采集的数量
func ExtractDynamicContent(ctx context.Context, maxCount int) (int, error) {
	scrollLock.Lock()
	defer scrollLock.Unlock()

	if err := chromedp.Run(ctx,
		chromedp.WaitReady(`//div[contains(@class, '_ac7v')]`, chromedp.BySearch),
		chromedp.Sleep(1*time.Second),
	); err != nil {
		return 0, err
	}

	var links []string
	if err := chromedp.Run(ctx,
		chromedp.EvaluateAsDevTools(`
            Array.from(document.querySelectorAll('div._ac7v a[href^="/"]'))
                .map(a => a.getAttribute('href'))
                .filter(href => href.split('/').filter(Boolean).length >= 2)
                .map(href => href.split('/').filter(Boolean)[0])
        `, &links),
	); err != nil {
		return 0, fmt.Errorf("执行JS查询失败: %w", err)
	}

	currentCount := 0
	for _, nickName := range links {
		if currentCount >= maxCount {
			break
		}

		// 防止重复采集
		if _, loaded := uniqueIDs.LoadOrStore(nickName, struct{}{}); loaded {
			continue
		}

		blogHome := fmt.Sprintf("https://www.instagram.com/%s/", url.PathEscape(nickName))
		blogger := &DB.IndiaBlogger{
			Nickname: nickName,
			Bloghome: blogHome,
			Issend:   0,
			Isreply:  0,
		}

		if err := DB.CreateBlogger(config.GormDb, blogger); err != nil {
			if !DB.IsDuplicateError(err) {
				log.Printf("插入失败: %s - %v", nickName, err)
			}
			continue
		}
		currentCount++
		log.Printf("新增采集: %s (进度: %d/%d)", nickName, currentCount, maxCount)
	}
	return currentCount, nil
}

// 工作goroutine
func searchWorker(workerID int) {
	defer workerWaitGroup.Done()

	defer func() {
		if r := recover(); r != nil {
			log.Printf("[Worker %d] 发生 panic: %v\n堆栈: %s", workerID, r, debug.Stack())
		}
	}()

	atomic.AddInt32(&activeSearchWorkers, 1)
	defer atomic.AddInt32(&activeSearchWorkers, -1)
	for {
		// 直接从数据库获取地点（每次获取一个）
		location, err := DB.GetAndLockLocation(config.GormDb)
		if err != nil || location == nil {
			log.Printf("[Worker %d] 获取地点失败或没有可用地点: %v", workerID, err)
			time.Sleep(5 * time.Second) // 避免高频空查询
			continue
		}

		// 使用原子操作标记地点已被处理
		if _, loaded := locationsMap.LoadOrStore(location, struct{}{}); loaded {
			log.Printf("[Worker %d] 地点 %s 正在被其他协程处理", workerID, location.LocationID)
			time.Sleep(1 * time.Second)
			continue
		}

		log.Printf("[Worker %d] 开始处理地点: %s", workerID, location.LocationID)

		// 获取账号
		account, err := utils.GetLoginAccount("./cookies")
		if err != nil || account == "" {
			log.Printf("[Worker %d] 获取账号失败，尝试补充账号", workerID)
			if err := AddNewAccount(workerID); err != nil {
				log.Printf("[Worker %d] 补充账号失败: %v", workerID, err)
			}
			locationsMap.Delete(location.LocationID) // 释放地点锁
			continue
		}

		// 执行任务
		if err := RetrySearchJob(workerID, account, location.LocationID); err != nil {
			log.Printf("[Worker %d] 处理失败: %v", workerID, err)
			if IsFatalError(err) {
				locationsMap.Delete(location.LocationID)
				return
			}
		}

		// 标记为已完成
		if err := DB.UpdateLocationState(config.GormDb, location.LocationID, 1); err != nil {
			log.Printf("[Worker %d] 标记地点完成状态失败: %v", workerID, err)
		}
		locationsMap.Delete(location.LocationID)
	}

}

// searchBlogger.go 的 RetrySearchJob 函数
func RetrySearchJob(workerID int, account, locationID string) error {
	const maxRetries = 3
	var lastErr error // 记录最后一次错误

	for attempt := 1; attempt <= maxRetries; attempt++ {
		err := SearchJob(account, locationID)
		if err == nil {
			return nil // 成功时返回 nil
		}
		lastErr = err // 记录错误

		if IsRecoverableError(err) {
			log.Printf("[Worker %d] 尝试 %d/%d 遇到可恢复错误: %v", workerID, attempt, maxRetries, err)
			time.Sleep(time.Duration(attempt) * time.Second)
		} else {
			return err // 非可恢复错误直接返回
		}
	}
	return fmt.Errorf("经过%d次重试仍失败: %w", maxRetries, lastErr) // 明确携带最后一次错误
}

// 错误类型判断
func IsRecoverableError(err error) bool {
	return strings.Contains(err.Error(), "账号已被暂停使用") ||
		strings.Contains(err.Error(), "地点页面错误") ||
		strings.Contains(err.Error(), "没有新数据") ||
		strings.Contains(err.Error(), "账号登录已过期")
}

func IsFatalError(err error) bool {
	if err == nil {
		return false // 如果 err 为 nil，直接返回 false
	}
	return strings.Contains(err.Error(), "没有可用地点ID")
}

// 修改后的SearchJob
func SearchJob(accountFile, locationId string) (retErr error) {
	// 初始化浏览器
	opts := utils.SetChromeOptionsImgless()
	allocCtx, cancelAlloc := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancelAlloc()

	ctx, cancelCtx := chromedp.NewContext(allocCtx)
	defer cancelCtx()

	// 错误处理
	defer func() {
		if retErr != nil {
			cancelCtx()
			cancelAlloc()
		}
	}()

	// 加载cookies
	if err := utils.LoadCookies(ctx, accountFile); err != nil {
		return fmt.Errorf("加载Cookies失败: %w", err)
	}

	// 执行主要逻辑
	if err := processLocation(ctx, accountFile, locationId); err != nil {
		return err
	}
	return nil
}

// 处理单个地点
func processLocation(ctx context.Context, accountFile, locationID string) error {
	receiverUrl := fmt.Sprintf("https://www.instagram.com/explore/locations/%s", locationID)

	// 页面初始化
	if err := chromedp.Run(ctx,
		chromedp.Navigate(receiverUrl),
	); err != nil {
		return fmt.Errorf("页面初始化失败: %w", err)
	}

	if err := chromedp.Run(ctx,

		chromedp.Sleep(2*time.Second),
		chromedp.WaitReady(`img`, chromedp.ByQuery),
		chromedp.WaitReady(`body`, chromedp.ByQuery),
	); err != nil {
		//return fmt.Errorf("页面初始化失败: %w", err)
	}

	var err1 error
	go func() {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()

		timeout := time.After(60 * time.Second)
		is := false
		var err error
		for {

			select {
			case <-ticker.C:
				err = CheckAlreadyLoggedStatus(ctx, accountFile)
				if err != nil {
					chromedp.Cancel(ctx)
					err1 = err
					return
				}
			case <-timeout:
				return
			}

			if is {

				break
			}
		}
	}()
	// 错误检测
	var body string
	if err := chromedp.Run(ctx,
		chromedp.Text(`body`, &body, chromedp.ByQuery),
	); err != nil {
		return err
	}

	switch {
	case strings.Contains(body, "你的账户或账户动态违反了我们社群守则"):
		log.Printf("%s 账号已被暂停使用", accountFile)
		err := HandleAbnormalAccount(ctx, accountFile)
		if err != nil {
			//	chromedp.Cancel(ctx)
			//return nil, err
			return err
		}

		return fmt.Errorf("账号已被暂停使用")

	case strings.Contains(body, "很抱歉，无法访问此页面"):
		DB.UpdateLocationState(config.GormDb, receiverUrl, 1)
		return fmt.Errorf("地点页面错误")

	case strings.Contains(body, "页面无法加载"):
		DB.UpdateLocationState(config.GormDb, receiverUrl, 1)
		return fmt.Errorf("地点页面错误")
	}

	log.Println("获取帖子数量")
	// 获取帖子数量
	count, err := GetPostStatistics(ctx)
	if err != nil || count == 0 {

		chromedp.Cancel(ctx)
		return fmt.Errorf("获取帖子数量失败: %w", err)
	}

	var links []string
	if err := chromedp.Run(ctx,
		chromedp.EvaluateAsDevTools(`
            Array.from(document.querySelectorAll('div._ac7v a[href^="/"]'))
                .map(a => a.getAttribute('href'))
                .filter(href => href.split('/').filter(Boolean).length >= 2)
                .map(href => href.split('/').filter(Boolean)[0])
        `, &links),
	); err != nil {
		//return 0, fmt.Errorf("执行JS查询失败: %w", err)
	}

	if len(links) == 0 {
		return fmt.Errorf("没有数据")
	}

	if err1 != nil {
		return err1
	}

	// 执行滚动采集
	if err := chromedp.Run(ctx, scrollBrowser(count, locationID)); err != nil {
		return fmt.Errorf("滚动采集失败: %w", err)
	}

	return nil
}

// 新增向上滚动函数
func scrollUp(ctx context.Context) error {
	upScript := `
    (() => {
        window.scrollBy({
            top: -200,
            behavior: 'smooth'
        });
        return true;
    })()
    `
	var success bool
	return chromedp.Evaluate(upScript, &success).Do(ctx)
}
func enhancedScroll(ctx context.Context) error {
	// 先向上滚动200px触发可能的懒加载
	if err := scrollUp(ctx); err != nil {
		log.Printf("向上滚动失败: %v", err)
	}

	// 原有向下滚动逻辑
	var oldHeight int
	if err := chromedp.Evaluate(`document.documentElement.scrollHeight`, &oldHeight).Do(ctx); err != nil {
		return fmt.Errorf("获取高度失败: %w", err)
	}

	scrollScript := `
    (() => {
        const delta = Math.floor(window.innerHeight * 0.8);
        window.scrollBy({ 
            top: delta,
            behavior: 'smooth' 
        });
        return delta;
    })()
    `

	var delta int
	if err := chromedp.Evaluate(scrollScript, &delta).Do(ctx); err != nil {
		return fmt.Errorf("滚动执行失败: %w", err)
	}

	// 增加基础等待时间 + 动态计算
	baseWait := 2 * time.Second
	dynamicWait := time.Duration(delta/300) * time.Millisecond
	totalWait := baseWait + dynamicWait

	return chromedp.Run(ctx,
		chromedp.Sleep(totalWait),
		chromedp.WaitReady(`div._ac7v:last-child`, chromedp.ByQuery),
	)
}

// 修改后的滚动逻辑
func scrollBrowser(maxCount int, locationId string) chromedp.Action {
	return chromedp.ActionFunc(func(ctx context.Context) error {
		retryCount := 0
		lastCount := 0

		for {
			currentCount, _ := ExtractDynamicContent(ctx, maxCount)

			switch {
			case currentCount >= maxCount:
				return nil
			case currentCount == lastCount:
				if retryCount++; retryCount >= 5 {
					if err := DB.UpdateLocationState(config.GormDb, locationId, 1); err != nil {
						fmt.Println("状态更新失败:", err)
					}
					return fmt.Errorf("连续5次无新数据")
				}
				time.Sleep(time.Duration(retryCount) * time.Second)
			default:
				retryCount = 0
				lastCount = currentCount
			}

			if err := enhancedScroll(ctx); err != nil {
				return err
			}
		}
	})
}

// 增强版搜索工作
func SearchBloggerWork(num int) {

	workerWaitGroup.Add(num)
	for i := 0; i < num; i++ {
		go searchWorker(i + 1)
	}
	workerWaitGroup.Wait()
	log.Println("所有采集任务完成")
}
