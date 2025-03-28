package transaction

import (
	"InsAutoMessage/config"
	DB "InsAutoMessage/database"
	"InsAutoMessage/utils"
	"context"
	"fmt"
	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/chromedp"
	"github.com/chromedp/chromedp/kb"
	"log"
	"math/rand"
	"strings"
	"time"
)

// 带上下文的发送函数
func ReplyMessageContext(ctx context.Context, replyMessages []string) error {
	// 加载Cookies

	messageTxt := replyMessages[rand.Intn(len(replyMessages))]
	time.Sleep(100)

	log.Println("消息内容：", messageTxt)

	err := chromedp.Run(ctx,
		chromedp.WaitVisible(`[aria-label="发消息"]`, chromedp.ByQuery),
		chromedp.Click(`[aria-label="发消息"]`, chromedp.ByQuery),
		chromedp.Sleep(500*time.Millisecond),
		chromedp.SendKeys(`[aria-label="发消息"]`, messageTxt+kb.Enter, chromedp.ByQuery),
		chromedp.Sleep(2*time.Second),
	)
	if err != nil {
		//return fmt.Errorf("发送消息出错: %w", err)
	}
	bloggerName := ""
	chromedp.Run(ctx,
		chromedp.WaitVisible(`span[class="x1lliihq x1plvlek xryxfnj x1n2onr6 x1ji0vk5 x18bv5gf x193iq5w xeuugli x1fj9vlw x13faqbe x1vvkbs x1s928wv xhkezso x1gmr53x x1cpjm7i x1fgarty x1943h6x x1i0vuye xvs91rp xo1l8bm x1roi4f4 x2b8uid x1tu3fi x3x7a5m x10wh9bi x1wdrske x8viiok x18hxmgj"]`, chromedp.ByQuery),
		chromedp.Text(`span[class="x1lliihq x1plvlek xryxfnj x1n2onr6 x1ji0vk5 x18bv5gf x193iq5w xeuugli x1fj9vlw x13faqbe x1vvkbs x1s928wv xhkezso x1gmr53x x1cpjm7i x1fgarty x1943h6x x1i0vuye xvs91rp xo1l8bm x1roi4f4 x2b8uid x1tu3fi x3x7a5m x10wh9bi x1wdrske x8viiok x18hxmgj"]`, &bloggerName, chromedp.ByQuery),
	)

	bloggerName = strings.ReplaceAll(bloggerName, " · Instagram", "")
	fmt.Println("博主名称：", bloggerName)
	if bloggerName != "" {

		err := DB.UpdateIsReplyByNickeName(config.GormDb, bloggerName, 1)
		if err != nil {
			//return err

			return fmt.Errorf("博主状态更新失败: %w", err)
		}
	}
	return nil
}

func ReplyWork(accountFile string, replyTexts []string) {
	opts := utils.SetChromeOptions()
	allocCtx, cancelAlloc := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancelAlloc()

	ctx, cancelCtx := chromedp.NewContext(allocCtx)
	defer cancelCtx()

	// 设置整体超时，防止任务长时间阻塞
	ctx, cancelTimeout := context.WithTimeout(ctx, 3*time.Minute)
	defer cancelTimeout()

	// 加载cookies
	if err := utils.LoadCookies(ctx, accountFile); err != nil {
		//return fmt.Errorf("加载Cookies失败: %w", err)
	}

	// 获取所有消息条目
	err := chromedp.Run(ctx,
		chromedp.Navigate("https://www.instagram.com/direct/inbox/"),
	)

	/*
		scale := float64(rand.Intn(6)+8) / 10
		chromedp.Run(ctx,
			chromedp.EvaluateAsDevTools(
				// 设置页面缩放比例的 JavaScript 代码
				fmt.Sprintf("document.body.style.zoom = %f;", scale),
				nil,
			),
		)

	*/
	if err != nil {
		log.Printf("初始化失败:", err)
	}

	go func() {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()
		timeout := time.After(30 * time.Second)
		for {
			select {
			case <-ticker.C:
				err := CheckAlreadyLoggedStatus(ctx, accountFile)
				if err != nil {
					chromedp.Cancel(ctx)
					return
				}
			case <-timeout:
				return
			}
		}
	}()

	chromedp.Run(ctx,
		chromedp.Evaluate(`
            var element = document.querySelector('[aria-label="对话列表"]');
            if (element) {
                element.scrollTop = element.scrollHeight;
            }
        `, nil),
	)

	var hasListitem bool

	err = chromedp.Run(ctx,
		chromedp.WaitReady(`[aria-label="对话列表"]`, chromedp.ByQuery),
		chromedp.EvaluateAsDevTools(
			`document.querySelector('div[role="listitem"]') !== null`,
			&hasListitem,
		),
	)

	if !hasListitem {
		log.Println("没有消息")
		//chromedp.Cancel(ctx)
		return
	}

	retryCount := 0
	maxRetries := 5
	var lastScrollHeight int
	err = chromedp.Run(ctx,

		// 循环滚动到底部
		chromedp.ActionFunc(func(ctx context.Context) error {
			for retryCount < maxRetries {
				// 获取当前滚动高度
				var newHeight int
				if err := chromedp.Evaluate(`
                    (() => {
                        const list = document.querySelector('div[class="x78zum5 xdt5ytf x1iyjqo2 xs83m0k x1xzczws x6ikm8r x1odjw0f x1n2onr6 xh8yej3 xish69e x16o0dkt"]');
                        list.scrollTop = list.scrollHeight;
                        return list.scrollHeight;
                    })()
                `, &newHeight).Do(ctx); err != nil {
					return err
				}

				var hasMessagesToReply bool
				err = chromedp.Run(ctx,
					//chromedp.WaitReady(`div[role="listitem"]`, chromedp.ByQuery),
					chromedp.EvaluateAsDevTools(
						`document.querySelector('div[class="x9f619 x1ja2u2z xzpqnlu x1hyvwdk x14bfe9o xjm9jq1 x6ikm8r x10wlt62 x10l6tqk x1i1rx1s"]') !== null`,
						&hasMessagesToReply,
					),
				)

				if newHeight == lastScrollHeight {
					retryCount++
					log.Printf("未检测到新内容，重试 %d/%d", retryCount, maxRetries)
					time.Sleep(1 * time.Second)
				} else {
					//retryCount = 0
					lastScrollHeight = newHeight
					log.Printf("已滚动到新位置，当前高度: %d", newHeight)
				}

				if hasMessagesToReply {
					break
				}

				/*
					// 检查是否滚动到底部



				*/
			}
			return nil
		}),
	)

	var hasMessagesToReply bool
	err = chromedp.Run(ctx,
		//chromedp.WaitReady(`div[role="listitem"]`, chromedp.ByQuery),
		chromedp.EvaluateAsDevTools(
			`document.querySelector('div[class="x9f619 x1ja2u2z xzpqnlu x1hyvwdk x14bfe9o xjm9jq1 x6ikm8r x10wlt62 x10l6tqk x1i1rx1s"]') !== null`,
			&hasMessagesToReply,
		),
	)

	var messageNodes []*cdp.Node
	if hasMessagesToReply {
		chromedp.Run(ctx,
			chromedp.Nodes(
				`div[class="x9f619 x1ja2u2z xzpqnlu x1hyvwdk x14bfe9o xjm9jq1 x6ikm8r x10wlt62 x10l6tqk x1i1rx1s"]`,
				&messageNodes,
				chromedp.ByQueryAll,
			),
		)
	}

	fmt.Printf("有 %d 个博主回复了消息\n", len(messageNodes))

	if len(messageNodes) > 0 {
		for i := 0; i < len(messageNodes); i++ {
			err = chromedp.Run(ctx,
				chromedp.Click(`div[class="x9f619 x1ja2u2z xzpqnlu x1hyvwdk x14bfe9o xjm9jq1 x6ikm8r x10wlt62 x10l6tqk x1i1rx1s"]`, chromedp.ByQuery),
			)
			if err != nil {
				//log.Printf("点击失败: %v (内容: %s)", err, content)
				continue
			}
			time.Sleep(1 * time.Second)

			// Reply to the message
			err = ReplyMessageContext(ctx, replyTexts)
			if err != nil {
				//log.Printf("回复失败: %v (内容: %s)", err, content)
			} else {
				fmt.Printf("回复成功\n")
			}
			time.Sleep(2 * time.Second)
		}

	} else {
		fmt.Println("没有需要回复的消息")
		return
	}

}
