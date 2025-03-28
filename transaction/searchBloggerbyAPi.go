package transaction

import (
	"InsAutoMessage/config"
	DB "InsAutoMessage/database"
	"InsAutoMessage/utils"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
	"io"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

var (
	SearchLocationsMap sync.Map
)

// Response 定义结构体解析 GraphQL 返回的 JSON 数据
type Response struct {
	Data struct {
		XDTLocationGetWebInfoTab struct {
			Edges []struct {
				Node struct {
					User struct {
						Username string `json:"username"`
					} `json:"user"`
					Owner struct {
						Username string `json:"username"`
					} `json:"owner"`
					CoauthorProducers []struct {
						Username string `json:"username"`
					} `json:"coauthor_producers"`
					Usertags struct {
						In []struct {
							User struct {
								Username string `json:"username"`
							} `json:"user"`
						} `json:"in"`
					} `json:"usertags"`
				} `json:"node"`
			} `json:"edges"`
			PageInfo struct {
				EndCursor   string `json:"end_cursor"`
				HasNextPage bool   `json:"has_next_page"`
			} `json:"page_info"`
		} `json:"xdt_location_get_web_info_tab"`
	} `json:"data"`
}

func EnBase64(base64String string) string {
	decodedBytes, err := base64.StdEncoding.DecodeString(base64String)
	if err != nil {
		log.Println("解码错误:", err)
		return ""
	}
	return string(decodedBytes)
}

// extractUsernames 从返回数据中提取所有不重复的 username，并返回是否还有下一页
func extractUsernames(response Response) ([]string, bool) {
	usernameSet := make(map[string]bool)
	var usernames []string

	// 从 user 字段提取
	for _, edge := range response.Data.XDTLocationGetWebInfoTab.Edges {
		usernameSet[edge.Node.User.Username] = true
	}
	// 从 owner 字段提取
	for _, edge := range response.Data.XDTLocationGetWebInfoTab.Edges {
		usernameSet[edge.Node.Owner.Username] = true
	}
	// 从 coauthor_producers 字段提取
	for _, edge := range response.Data.XDTLocationGetWebInfoTab.Edges {
		for _, producer := range edge.Node.CoauthorProducers {
			usernameSet[producer.Username] = true
		}
	}
	// 从 usertags 字段提取
	for _, edge := range response.Data.XDTLocationGetWebInfoTab.Edges {
		for _, tag := range edge.Node.Usertags.In {
			usernameSet[tag.User.Username] = true
		}
	}
	// 将 map 中的键转换为切片
	for username := range usernameSet {
		usernames = append(usernames, username)
	}

	hasNextPage := response.Data.XDTLocationGetWebInfoTab.PageInfo.HasNextPage

	return usernames, hasNextPage
}

// sendHTTPRequest 通过代理发送 POST 请求获取接口响应内容
func sendHTTPRequest(proxyArr, payloads, cookieStr string) (string, error) {
	url1 := "https://www.instagram.com/graphql/query"
	method := "POST"

	payload := strings.NewReader(payloads)
	proxy, err := url.Parse(proxyArr)
	if err != nil {
		log.Println("解析代理地址错误:", err)
		return "", err
	}

	// 配置网络传输
	netTransport := &http.Transport{
		Proxy:                 http.ProxyURL(proxy),
		MaxIdleConnsPerHost:   10,
		ResponseHeaderTimeout: 5 * time.Second,
	}
	// 创建 HTTP 客户端
	client := &http.Client{
		Timeout:   10 * time.Second,
		Transport: netTransport,
	}
	req, err := http.NewRequest(method, url1, payload)
	if err != nil {
		log.Println(err)
		return "", err
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) "+
		"AppleWebKit/537.36 (KHTML, like Gecko) Chrome/133.0.0.0 Safari/537.36")
	req.Header.Add("X-IG-App-ID", "936619743392459")
	req.Header.Add("COOKIE", cookieStr)

	res, err := client.Do(req)
	if err != nil {
		log.Println("HTTP请求失败:", err)
		return "", err
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		log.Println("读取响应体失败:", err)
		return "", err
	}
	return string(body), nil
}

// extractAndProcessPayload 提取并处理 PostDataEntries
func extractAndProcessPayload(entries []*network.PostDataEntry) string {
	var payload string
	for _, entry := range entries {
		payload = EnBase64(entry.Bytes)
	}
	return payload
}

// SearchJobApi 通过浏览器实例和 API 请求采集指定地点的博主数据
func SearchJobApi(workerID int, accountFile, locationId string) error {
	log.Printf("启动浏览器实例处理地点: %s", locationId)

	// 创建带超时控制的 ExecAllocator 和上下文（例如3分钟超时）
	opts := utils.SetChromeOptionsImgless()
	allocCtx, cancelAlloc := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancelAlloc()

	ctx, cancelCtx := chromedp.NewContext(allocCtx)
	// 此处设置超时，防止页面操作无限等待
	ctx, cancelTimeout := context.WithTimeout(ctx, 2*time.Minute)

	// 统一 defer 保证取消顺序
	defer func() {
		cancelTimeout()
		cancelCtx()
		cancelAlloc()
	}()

	// 加载 Cookies 文件
	if err := utils.LoadCookies(ctx, accountFile); err != nil {
		log.Printf("加载Cookies失败: %v", err)

		return fmt.Errorf("加载Cookies失败: %w", err)

	}

	var payloadStr string
	// 监听网络请求，捕获目标接口的请求载荷
	chromedp.ListenTarget(ctx, func(ev interface{}) {
		if ev, ok := ev.(*network.EventRequestWillBeSent); ok {
			if ev.Request.URL == "https://www.instagram.com/graphql/query" {
				if ev.Request.HasPostData {
					postData := extractAndProcessPayload(ev.Request.PostDataEntries)

					if strings.Contains(postData, "variables=%7B%22after") && !strings.Contains(postData, "after%22%3Anull") {
						//postData = strings.ReplaceAll(postData, "ranked", "recent")

						payloadStr = postData

						//payloadStr = strings.ReplaceAll(payloadStr, "ranked", "recent")

						log.Printf("🔹 请求载荷:%s \n", payloadStr)

					}
				}
			}
		}
	})

	// 导航至指定地点页面，并触发滚动操作
	if err := chromedp.Run(ctx,
		chromedp.Navigate(fmt.Sprintf("https://www.instagram.com/explore/locations/%s", locationId)),
		chromedp.Sleep(2*time.Second),
		chromedp.Evaluate(`window.scrollTo(0, document.body.scrollHeight)`, nil),
		chromedp.Sleep(3*time.Second),
	); err != nil {
		log.Printf("浏览器操作失败: %v", err)
		return fmt.Errorf("浏览器操作失败: %w", err)
	}

	body := ""
	Url := ""
	err := chromedp.Run(ctx,
		chromedp.WaitReady(`img`, chromedp.ByQuery),
		chromedp.Text(`body`, &body, chromedp.ByQuery),
		chromedp.Location(&Url),
	)
	if err != nil {
		return err
	}

	go func() {
		ticker := time.NewTicker(800)
		defer ticker.Stop()

		timeout := time.After(60 * time.Second)
		log.Println("正在检测")

		for {
			//
			select {
			case <-ticker.C:
				err = CheckAlreadyLoggedStatus(ctx, accountFile)
				if err != nil {
					cancelCtx()
					return
				}
			case <-timeout:
				cancelCtx()
				return
			}
		}

	}()

	if strings.Contains(body, "出现问题，页面无法加载。") {
		log.Println("当前地址错误")

		err := DB.UpdateLocationState(config.GormDb, locationId, 1)
		if err != nil {
			return err
		}
		return fmt.Errorf("当前地址错误")
	}

	// 获取帖子数量
	log.Println("获取帖子数量")
	// 获取帖子数量
	count, err := GetPostStatistics(ctx)
	if err != nil || count == 0 {

		chromedp.Cancel(ctx)
		return fmt.Errorf("获取帖子数量失败: %w", err)
	}

	/*
		chromedp.Run(ctx,
			chromedp.WaitReady(`//span[text()='最新']`, chromedp.BySearch),
			chromedp.Click(`//span[text()='最新']`, chromedp.BySearch),
		)

	*/

	// 更新数据库中的帖子数量
	loc, err := DB.GetLocationByID(config.GormDb, locationId)
	if err != nil {
		log.Println("查询失败:", err)
	} else {
		//log.Printf("查询结果: %+v\n", loc)
	}
	loc.Count = count
	if err := DB.UpdateLocation(config.GormDb, loc); err != nil {
		log.Println("更新失败:", err)
	}

	// 如果 payloadStr 为空，则尝试 Reload 并等待一段时间捕获
	if payloadStr == "" {
		if err := chromedp.Run(ctx,
			//network.Enable(),
			chromedp.Reload(),
			chromedp.Sleep(2*time.Second),
		); err != nil {
			log.Printf("Reload失败: %v", err)
		}
		// 等待最多10秒看是否捕获到 payload
		deadline := time.Now().Add(10 * time.Second)
		for payloadStr == "" && time.Now().Before(deadline) {
			time.Sleep(500 * time.Millisecond)
		}
		if payloadStr == "" {
			log.Printf("[Worker %d] 重试后仍未捕获到 payload (location=%s)", workerID, locationId)

			return fmt.Errorf("未捕获到有效的 payload (location=%s)", locationId)
		}
	}

	cookiesStr, err := utils.ToCookieStr(accountFile)
	if err != nil && cookiesStr == "" {
		return fmt.Errorf("cookies转字符串失败: %w", err)
	}

	currentCount := 0
	for {

		body, err = sendHTTPRequest(config.AppCfg.Proxyaddr, payloadStr, cookiesStr)
		if err != nil {

			return fmt.Errorf("获取数据失败: %w", err)
		}
		var response Response
		if err := json.Unmarshal([]byte(body), &response); err != nil {
			//return fmt.Errorf("解析JSON失败: %w", err)

			return fmt.Errorf("解析JSON失败: %w", err)

		}

		if len(response.Data.XDTLocationGetWebInfoTab.Edges) == 0 {

			return fmt.Errorf("未找到数据")
		}

		// 提取用户名和是否还有下一页数据
		usernames, hasNextPage := extractUsernames(response)
		var bloggersTmp []*DB.IndiaBlogger

		for _, nickName := range usernames {
			blogHome := fmt.Sprintf("https://www.instagram.com/%s/", url.PathEscape(nickName))
			blogger := &DB.IndiaBlogger{
				Nickname: nickName,
				Bloghome: blogHome,
				Issend:   0,
				Isreply:  0,
			}
			/*
				if err := DB.CreateBlogger(config.GormDb, blogger); err != nil {
					if !DB.IsDuplicateError(err) {
						log.Printf("插入失败: %s - %v", nickName, err)
					}
					continue
				}
			*/
			log.Printf("[Worker %d]  采集到: %s ，已采集：%d条", workerID, nickName, currentCount)
			bloggersTmp = append(bloggersTmp, blogger)
			currentCount++
		}

		// 如果有待插入的记录，则批量插入
		if len(bloggersTmp) > 0 {
			// 批量插入，假设 DB.BatchCreateBlogger 实现了批量插入并自动忽略重复错误
			if err := DB.BatchCreateBlogger(config.GormDb, bloggersTmp); err != nil {
				log.Printf("[Worker %d] 批量插入失败: %v", workerID, err)
			} else {
				currentCount += len(bloggersTmp)
				log.Printf("[Worker %d] 批量采集到 %d 个用户，累计 %d 条", workerID, len(bloggersTmp), currentCount)
			}
		}

		// 如果已经没有下一页，则更新状态并启动新任务后退出
		if !hasNextPage {

			if err := DB.UpdateLocationState(config.GormDb, locationId, 1); err != nil {
				fmt.Println("状态更新失败:", err)
			}
			log.Printf("[Worker %d] 采集完成", workerID)

			return nil

		}
		// 每轮采集后稍作等待
		//rand.Seed(time.Now().UnixNano())
		waitTime := time.Duration(rand.Intn(8)+3) * time.Second
		log.Printf("[Worker %d] 等待 %s 后继续采集", workerID, waitTime)
		time.Sleep(waitTime)

	}

}

// SearchBloggerWorkAPi 启动多个并发任务采集不同地点的博主数据，保持浏览器实例数始终为指定的 num
func SearchBloggerWorkAPi(num int) {
	var wg sync.WaitGroup

	for i := 0; i < num; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			maxRetries := 5
			retryCount := 0
			for retryCount < maxRetries {
				// 从数据库获取一个未采集的地点ID
				locations, err := DB.GetRandomUncollectedLocations(config.GormDb, 1)

				if err != nil || len(locations) == 0 {
					retryCount++
					time.Sleep(5 * time.Second)
					continue
				}

				if err != nil {
					log.Printf("[Worker %d] 查询地点失败: %v", workerID, err)
					return
				}
				if len(locations) == 0 {
					log.Printf("[Worker %d] 没有可用地点ID", workerID)
					return
				}
				locationId := locations[0].LocationID // 安全访问

				if _, loaded := SearchLocationsMap.LoadOrStore(locationId, struct{}{}); loaded {
					log.Printf("[Worker %d] 地点 %s 已被其他协程处理，跳过", workerID, locationId)
					time.Sleep(1 * time.Second)
					continue
				}

				SearchLocationsMap.Store(locationId, struct{}{})

				// 获取账号（失败后释放地点锁）
				account, err := utils.GetLoginAccount("./cookies")
				if err != nil || account == "" {
					log.Printf("[Worker %d] 无可用账号，释放地点锁", workerID)
					SearchLocationsMap.Delete(locationId) // 关键修改：立即释放锁
					if err := AddNewAccount(workerID); err != nil {
						log.Printf("[Worker %d] 补号失败: %v", workerID, err)
					}
					continue
				}
				// 调用 SearchJobApi 进行采集（内部会创建新的浏览器实例，并通过 defer 关闭）
				log.Printf("[Worker %d] 启动新浏览器实例，处理地点: %s", workerID, locationId)
				err = SearchJobApi(workerID, account, locationId)
				if err != nil {
					log.Printf("[Worker %d] 采集任务失败（%s），关闭当前浏览器，重新获取新地点", workerID, err)
					// 强制释放地点锁（允许其他协程重试）
					SearchLocationsMap.Delete(locationId) // 关键修改：失败时删除锁

				} else {
					log.Printf("[Worker %d] 采集成功，准备下一任务", workerID)
				}
				// 稍作延时，防止过快轮询
				// 任务间隔控制
				interval := config.AppCfg.CrawlersTimeInterval
				if interval == 0 {
					interval = 5
				}
				wait := time.Duration(rand.Intn(interval)) * time.Minute
				log.Printf("[Worker %d] 等待 %v 后继续", workerID, wait)
				time.Sleep(wait)

				retryCount = 0 // 重置重试计数器
			}

			if retryCount >= maxRetries {
				log.Printf("[Worker %d] 达到最大重试次数，退出任务", workerID)
				return
			}
		}(i)
	}
	wg.Wait()
	log.Println("所有浏览器实例执行完毕")
}
