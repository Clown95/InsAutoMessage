package transaction

import (
	"InsAutoMessage/config"
	DB "InsAutoMessage/database"
	"InsAutoMessage/utils"
	"context"
	"fmt"
	"github.com/chromedp/chromedp"
	"github.com/chromedp/chromedp/kb"
	"log"
	"math/rand"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// clickSendMessageButton 点击第一种发送消息按钮
func clickSendMessageButton(ctx context.Context) error {
	return chromedp.Run(ctx,
		chromedp.WaitVisible(`//div[text()='发消息']`, chromedp.BySearch),
		chromedp.Click(`//div[text()='发消息']`, chromedp.BySearch),
	)
}

// clickSendMessageButtonAlt 点击备用方案的发送消息按钮
func clickSendMessageButtonAlt(ctx context.Context) error {
	return chromedp.Run(ctx,
		chromedp.WaitVisible(`[aria-label="选项"]`, chromedp.ByQuery),
		chromedp.Sleep(1*time.Second),
		chromedp.Click(`[aria-label="选项"]`, chromedp.ByQuery),
		chromedp.Sleep(1*time.Second),
		chromedp.Click(`//button[text()='发消息']`, chromedp.BySearch),
	)
}

// 发送函数
func SendMessageJob(accountFile, homePage string, messageTxts []string) error {

	opts := utils.SetChromeOptionsImgless()
	allocCtx, cancelAlloc := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancelAlloc()

	ctx, cancelCtx := chromedp.NewContext(allocCtx)
	defer cancelCtx()

	// 设置整体超时，防止任务长时间阻塞
	ctx, cancelTimeout := context.WithTimeout(ctx, 60*time.Second)
	defer cancelTimeout()

	// 加载Cookies
	if err := utils.LoadCookies(ctx, accountFile); err != nil {
		return fmt.Errorf("加载Cookies失败: %w", err)
	}
	scale := float64(rand.Intn(9)+6) / float64(10)
	// 执行导航操作
	//fmt.Println("scale", scale)
	//isSuccess := true

	if err := chromedp.Run(ctx,
		chromedp.Navigate(homePage),
		chromedp.EvaluateAsDevTools(
			// 设置页面缩放比例的 JavaScript 代码
			fmt.Sprintf("document.body.style.zoom = %f;", scale),
			nil,
		),
		chromedp.Sleep(3*time.Second),
	); err != nil {
		return fmt.Errorf("跳转页面失败: %w", err)
	}

	body := ""
	checkUrl := ""
	err := chromedp.Run(ctx,

		chromedp.WaitReady(`img`, chromedp.ByQuery),
		chromedp.Text(`body`, &body, chromedp.ByQuery),
		chromedp.EvaluateAsDevTools(fmt.Sprintf("window.scrollBy(0, %d);", rand.Intn(100)+10), nil),
		chromedp.Location(&checkUrl),
	)
	if err != nil {
		return fmt.Errorf("操作浏览器失败: %w", err)
	}

	if strings.Contains(body, "很抱歉，无法访问此页面") {
		//isSuccess = false
		err := DB.UpdateIsReplyByBloghome(config.GormDb, homePage, 1)
		if err != nil {
			return err
		}
		//isSuccess = f
		return fmt.Errorf("主页地址错误")
	}

	go func() {
		ticker := time.NewTicker(900)
		defer ticker.Stop()
		timeout := time.After(60 * time.Second)
		log.Println("正在检测账号状态")
		for {
			select {
			case <-ticker.C:
				stateErr := CheckAlreadyLoggedStatus(ctx, accountFile)
				if stateErr != nil {
					//isSuccess = false
					cancelCtx()
					break
				}
			case <-timeout:
				return
			}
		}
	}()

	for i := 0; i < 3; i++ {
		var sendButtonExists bool
		chromedp.Run(ctx,
			chromedp.WaitReady(`[aria-label="选项"]`, chromedp.ByQuery),
			chromedp.EvaluateAsDevTools(`document.querySelector('[aria-label="对话信息"]') !== null`, &sendButtonExists),
		)

		if !sendButtonExists {

			// 检查发消息按钮
			var buttonExists bool
			err := chromedp.Run(ctx,
				chromedp.Sleep(time.Duration(rand.Intn(3)+1)*time.Second),
				chromedp.WaitVisible(`[aria-label="选项"]`, chromedp.ByQuery),
				//chromedp.WaitReady(".x5n08af .x1s688f", chromedp.ByQuery),
				chromedp.EvaluateAsDevTools(`document.evaluate("//div[text()='发消息']", document, null, XPathResult.FIRST_ORDERED_NODE_TYPE, null).singleNodeValue !== null`, &buttonExists),
			)
			if err != nil {
				log.Printf("检查发消息按钮出错: %v\n", err)
				return err
			}

			// 根据按钮存在情况点击发送消息按钮
			if buttonExists {
				if err := clickSendMessageButton(ctx); err != nil {
					log.Printf("点击发消息按钮出错: %v\n", err)
				}
				break
			} else {
				if err := clickSendMessageButtonAlt(ctx); err != nil {
					log.Printf("点击备用发消息按钮出错: %v\n", err)
				}
				break
			}
		}
	}

	messageTxt := messageTxts[rand.Intn(len(messageTxts))]
	time.Sleep(100)

	log.Println("消息内容：", messageTxt)

	err = chromedp.Run(ctx,
		chromedp.WaitVisible(`[aria-label="发消息"]`, chromedp.ByQuery),
		chromedp.Click(`[aria-label="发消息"]`, chromedp.ByQuery),
		chromedp.Sleep(time.Duration(rand.Intn(2)+1)*300),

		chromedp.SendKeys(`[aria-label="发消息"]`, messageTxt+kb.Enter, chromedp.ByQuery),
		chromedp.Sleep(time.Duration(rand.Intn(5)+4)*time.Second),
	)
	if err != nil {
		return fmt.Errorf("发送消息出错: %w", err)
	}

	err = DB.UpdateIssendByBloghome(config.GormDb, homePage, 1)
	if err != nil {
		return fmt.Errorf("发送状态更新失败: %w", err)
	}

	return nil
}

func SendMessageWorker(workerNum int, messageTxts []string, timeInterval int) {

	files, err := utils.GetJSONFiles("./cookies")
	if err != nil {
		log.Println("获取账号失败:", err)
		return
	}

	if len(files) == 0 {
		log.Println("没有已登录账号")
		return
	}

	bloggerNum := config.AppCfg.SendBloggerNum

	if bloggerNum < 0 {
		log.Println("没有设置SendBloggerNum的值,自动启动发送全部博主")
		count, err := DB.GetAvailableBloggerCount(config.GormDb)
		if err != nil {
			return
		}
		bloggerNum = int(count)
	}

	jobChan := make(chan DB.IndiaBlogger, bloggerNum)

	bloggers, err := DB.GetRandomBlogger(config.GormDb, bloggerNum)
	if err != nil {
		return
	}

	var wg sync.WaitGroup
	sem := make(chan struct{}, workerNum) // 并发控制信号量

	// 启动worker池
	for i := 0; i < workerNum; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			log.Printf("Worker %d 启动", workerID)
			for blogger := range jobChan {

				sem <- struct{}{} // 获取信号量
				go func(blogger DB.IndiaBlogger) {
					defer func() { <-sem }() // 释放信号量

					//获取已登录账号
					accountFile, err := utils.GetLoginAccount("./cookies")
					if err != nil {
						return
					}

					accountName := strings.TrimSuffix(filepath.Base(accountFile), filepath.Ext(accountFile))
					if accountName == "" {
						return
					}

					log.Printf("[Worker %d] 开始使用账号 %s 发送消息到 %s", workerID, accountName, blogger.Nickname)
					start := time.Now()
					err = SendMessageJob(accountFile, blogger.Bloghome, messageTxts)
					if err != nil {

						return
					}
					log.Printf("[Worker %d] 成功处理账号 %s 发送消息到 %s (耗时: %s)",
						workerID, accountName, blogger.Nickname, time.Since(start).Round(time.Second))
				}(blogger)

				time.Sleep(time.Duration(timeInterval) * time.Second)
			}
			log.Printf("Worker %d 退出", workerID)
		}(i + 1)
	}

	// 将所有账号加入任务队列

	for _, blogger := range bloggers {
		jobChan <- blogger
	}
	close(jobChan)
	wg.Wait()
	log.Printf("全部 %d 个博主处理完成", len(bloggers))

}
