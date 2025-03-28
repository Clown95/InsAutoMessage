package transaction

import (
	"InsAutoMessage/config"
	DB "InsAutoMessage/database"
	"InsAutoMessage/utils"
	"context"
	"errors"
	"fmt"
	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/chromedp"
	"github.com/chromedp/chromedp/kb"
	"log"
	"math/rand"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// 添加超时控制的SendtoBlogger函数
func SendtoBlogger(ctx context.Context, blogger DB.IndiaBlogger, messageTxts []string) error {
	// 创建一个带超时的上下文，防止函数无限阻塞
	timeoutCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	// 使用errChan来捕获函数执行结果
	errChan := make(chan error, 1)

	// 在goroutine中执行SendtoBlogger的主要逻辑
	go func() {
		randomDelay := time.Duration(rand.Intn(4)) * time.Second
		time.Sleep(randomDelay)

		log.Println("开始查找用户")

		err := chromedp.Run(timeoutCtx,
			chromedp.WaitReady(`[aria-label="首页"]`, chromedp.ByQuery),
			chromedp.Click(`[aria-label="新消息"]`, chromedp.ByQuery),
			chromedp.WaitReady(`input[name="queryBox"]`, chromedp.ByQuery),
			chromedp.SendKeys(`input[name="queryBox"]`, blogger.Nickname, chromedp.ByQuery),
		)
		if err != nil {
			errChan <- err
			return
		}

		isNotFind := false
		err = chromedp.Run(timeoutCtx,
			chromedp.WaitReady(`[aria-label="关闭"]`, chromedp.ByQuery),
			chromedp.Sleep(5*time.Second),
			chromedp.Evaluate(`document.evaluate("//span[contains(text(), '找不到帐户。')]", document, null, XPathResult.FIRST_ORDERED_NODE_TYPE, null).singleNodeValue!== null;`, &isNotFind),
		)
		if err != nil {
			errChan <- err
			return
		}

		if isNotFind {
			updateBloggerStatus(&blogger, 1)

			chromedp.Run(timeoutCtx,
				chromedp.Click(`[aria-label="关闭"]`, chromedp.ByQuery),
			)
			errChan <- fmt.Errorf("找不到用户:%s", blogger.Nickname)
			return
		}

		var nodes []*cdp.Node
		err = chromedp.Run(timeoutCtx,
			chromedp.WaitReady(`[aria-label="Radio selection"]`, chromedp.ByQuery),
			chromedp.Nodes(`[aria-label="Radio selection"]`, &nodes, chromedp.AtLeast(0)),
		)
		if err != nil {
			errChan <- err
			return
		}

		fmt.Println(len(nodes))
		if len(nodes) > 0 {
			log.Println("找到用户，并点击")
			err = chromedp.Run(timeoutCtx,
				chromedp.MouseClickNode(nodes[0]),
				chromedp.Sleep(1000),
				chromedp.WaitReady(`div[class="x1i10hfl xjqpnuy xa49m3k xqeqjp1 x2hbi6w x972fbf xcfux6l x1qhh985 xm0m39n xdl72j9 x2lah0s xe8uvvx xdj266r x11i5rnm xat24cr x1mh8g0r x2lwn1j xeuugli xexx8yu x18d9i69 x1hl2dhg xggy1nq x1ja2u2z x1t137rt x1q0g3np x1lku1pv x1a2a7pz x6s0dn4 xjyslct x1lq5wgf xgqcy7u x30kzoy x9jhf4c x1ejq31n xd10rxx x1sy0etr x17r0tee x9f619 x9bdzbf x1ypdohk x78zum5 x1f6kntn xwhw2v2 xl56j7k x17ydfre x1n2onr6 x2b8uid xlyipyv x87ps6o x14atkfc xcdnw81 x1i0vuye xn3w4p2 x5ib6vp xc73u3c x1tu34mt xzloghq"]`, chromedp.ByQuery),
				chromedp.Click(`div[class="x1i10hfl xjqpnuy xa49m3k xqeqjp1 x2hbi6w x972fbf xcfux6l x1qhh985 xm0m39n xdl72j9 x2lah0s xe8uvvx xdj266r x11i5rnm xat24cr x1mh8g0r x2lwn1j xeuugli xexx8yu x18d9i69 x1hl2dhg xggy1nq x1ja2u2z x1t137rt x1q0g3np x1lku1pv x1a2a7pz x6s0dn4 xjyslct x1lq5wgf xgqcy7u x30kzoy x9jhf4c x1ejq31n xd10rxx x1sy0etr x17r0tee x9f619 x9bdzbf x1ypdohk x78zum5 x1f6kntn xwhw2v2 xl56j7k x17ydfre x1n2onr6 x2b8uid xlyipyv x87ps6o x14atkfc xcdnw81 x1i0vuye xn3w4p2 x5ib6vp xc73u3c x1tu34mt xzloghq"]`, chromedp.ByQuery),
			)
			if err != nil {
				errChan <- err
				return
			}
		}

		time.Sleep(300 * time.Millisecond)

		messageTxt := messageTxts[rand.Intn(len(messageTxts))]

		hasInvite := false
		err = chromedp.Run(timeoutCtx,
			chromedp.Sleep(1*time.Second),
			chromedp.WaitVisible(`span[class="x1lliihq x193iq5w x6ikm8r x10wlt62 xlyipyv xuxw1ft"]`, chromedp.ByQuery),
			chromedp.Evaluate(`document.evaluate("//span[text()='Invite sent']", document, null, XPathResult.FIRST_ORDERED_NODE_TYPE, null).singleNodeValue!== null;`, &hasInvite),
		)
		if err != nil {
			errChan <- err
			return
		}

		if hasInvite {
			log.Println("已发过信息")
			updateBloggerStatus(&blogger, 1)
			errChan <- fmt.Errorf("%s 已经发送过消息", blogger.Nickname)
			return
		}

		log.Println("开始发送消息")
		err = chromedp.Run(timeoutCtx,
			chromedp.WaitVisible(`[aria-label="发消息"]`, chromedp.ByQuery),
			chromedp.Click(`[aria-label="发消息"]`, chromedp.ByQuery),
			chromedp.Sleep(time.Duration(rand.Intn(2)+1)*300*time.Millisecond),
			chromedp.SendKeys(`[aria-label="发消息"]`, messageTxt+kb.Enter, chromedp.ByQuery),
		)
		if err != nil {
			updateBloggerStatus(&blogger, 0)
			errChan <- err
			return
		}
		updateBloggerStatus(&blogger, 1)

		log.Println("消息发送完成")

		err = chromedp.Run(timeoutCtx,
			chromedp.Sleep(3*time.Second),
		)
		if err != nil {
			errChan <- err
			return
		}

		err = CheckClickMessage(timeoutCtx)
		if err != nil {
			errChan <- err
			return
		}

		// 成功完成
		errChan <- nil
	}()

	// 等待结果或超时
	select {
	case err := <-errChan:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

// 周期性刷新页面函数
func RefreshPage(ctx context.Context) error {
	return chromedp.Run(ctx,
		chromedp.Reload(),
		chromedp.WaitReady(`[aria-label="首页"]`, chromedp.ByQuery),
	)
}

func SendMessageWorkOne(workerNum int, messageTxts []string, timeInterval int) {
	files, err := utils.GetJSONFiles("./cookies")
	if err != nil {
		log.Println("获取账号失败:", err)
		return
	}

	if len(files) == 0 {
		log.Println("没有已登录账号")
		return
	}

	var wg sync.WaitGroup
	sem := make(chan struct{}, workerNum)

	// 定义页面刷新的时间间隔
	refreshInterval := 5 * time.Minute

	// 账号循环
	for {
		accountFile, err := utils.GetUniqueLoginAccount("./cookies")
		if err != nil {
			log.Println("获取账号失败:", err)
			return
		}

		// 初始化浏览器实例
		opts := utils.SetChromeOptionsImgless()
		allocCtx, cancelAlloc := chromedp.NewExecAllocator(context.Background(), opts...)
		defer cancelAlloc()

		ctx, cancelCtx := chromedp.NewContext(allocCtx)
		defer cancelCtx()

		if err := utils.LoadCookies(ctx, accountFile); err != nil {
			log.Println("加载cookies失败:", err)
			continue
		}

		if err := chromedp.Run(ctx,
			chromedp.Navigate("https://www.instagram.com/direct/inbox/"),
		); err != nil {
			log.Println("导航失败:", err)
			continue
		}

		// 状态监控
		statusCheckCtx, stopStatusCheck := context.WithCancel(ctx)
		defer stopStatusCheck()
		go func() {
			ticker := time.NewTicker(900)
			defer ticker.Stop()
			for {
				select {
				case <-ticker.C:
					if stateErr := CheckAlreadyLoggedStatus(statusCheckCtx, accountFile); stateErr != nil {
						log.Printf("[状态异常] 准备切换账号...")
						cancelCtx()
						return
					}
				case <-statusCheckCtx.Done():
					return
				}
			}
		}()

		// 周期性页面刷新计时器
		refreshTimer := time.NewTimer(refreshInterval)
		defer refreshTimer.Stop()

		// 博主处理循环
	bloggerLoop:
		for {
			select {
			case <-ctx.Done():
				log.Println("上下文已取消，切换账号")
				break bloggerLoop
			case <-refreshTimer.C:
				// 页面刷新逻辑
				log.Println("执行定期页面刷新...")

				// 创建一个新的上下文，设置超时来防止刷新阻塞
				refreshCtx, cancelRefresh := context.WithTimeout(ctx, 30*time.Second)
				err := RefreshPage(refreshCtx)
				cancelRefresh()

				if err != nil {
					log.Printf("页面刷新失败: %v, 尝试切换账号", err)
					cancelCtx() // 页面刷新失败，切换账号
					break bloggerLoop
				}

				// 重置刷新计时器
				refreshTimer.Reset(refreshInterval)

			default:
				// 获取下一个待处理的博主
				bloggers, err := DB.GetRandomBlogger(config.GormDb, 1)
				if err != nil || len(bloggers) == 0 {
					log.Println("没有可用博主，等待5秒后重试")
					time.Sleep(5 * time.Second)
					continue
				}
				blogger := bloggers[0]

				// 使用信号量控制并发
				sem <- struct{}{}
				wg.Add(1)

				// 为每个博主创建一个处理goroutine
				go func(blogger DB.IndiaBlogger) {
					defer func() {
						<-sem // 释放信号量
						wg.Done()

						// 捕获panic避免整个程序崩溃
						if r := recover(); r != nil {
							log.Printf("处理博主 %s 时发生panic: %v", blogger.Nickname, r)
						}
					}()

					accountName := strings.TrimSuffix(filepath.Base(accountFile), filepath.Ext(accountFile))
					log.Printf("使用账号 %s 处理博主 %s", accountName, blogger.Nickname)

					start := time.Now()

					// 创建一个子上下文用于这个特定任务，有自己的超时控制
					taskCtx, taskCancel := context.WithTimeout(ctx, 3*time.Minute)
					defer taskCancel()

					updateBloggerStatus(&blogger, 99)
					// 发送消息给博主
					err := SendtoBlogger(taskCtx, blogger, messageTxts)

					if err != nil {
						if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
							log.Printf("处理博主 %s 超时或被取消", blogger.Nickname)
						} else {
							log.Printf("发送失败: %v", err)
						}
						return
					}

					log.Printf("成功处理 %s (耗时: %s)", blogger.Nickname, time.Since(start))

					// 添加随机延迟，避免操作过于规律
					delay := time.Duration(timeInterval+rand.Intn(5)) * time.Second
					time.Sleep(delay)
				}(blogger)

				// 等待当前博主处理完成后再处理下一个
				wg.Wait()
			}
		}

		// 清理当前账号资源
		cancelCtx()
		log.Println("切换新账号...")
		time.Sleep(5 * time.Second) // 切换账号前稍作延迟
	}
}

// 浏览器池管理
type BrowserInstance struct {
	Context context.Context
	Cancel  context.CancelFunc
	Account string
	Valid   bool
}

func updateBloggerStatus(blogger *DB.IndiaBlogger, status int) {
	tx := config.GormDb.Begin()
	defer tx.Rollback()

	blogger.Issend = uint(status)
	if err := tx.Save(blogger).Error; err == nil {
		tx.Commit()
	}
}
