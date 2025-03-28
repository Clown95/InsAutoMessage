package utils

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
	"log"
	"math/rand"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

func IsChromeRunning() bool {
	cmd := exec.Command("tasklist")
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(strings.ToLower(line), "chrome.exe") {
			return true
		}
	}
	return false
}

func StartChreome() error {
	if runtime.GOOS == "windows" {
		cmd := exec.Command("cmd", "/C", "start", "chrome", "--remote-debugging-port=9222")
		err := cmd.Start()
		if err != nil {
			return errors.New(fmt.Sprintf("启动命令失败: %v", err))
		}
	}
	return nil
}

func IsRemote() bool {
	ctx, cancel := chromedp.NewRemoteAllocator(context.Background(), "http://127.0.0.1:9222")
	defer cancel()
	ctx, cancel = chromedp.NewContext(ctx)
	defer cancel()
	var title string
	err := chromedp.Run(ctx, chromedp.Title(&title))
	return err == nil
}

func generateRandomUserAgent() string {
	// 定义一些常见的桌面浏览器 User-Agent 模式
	userAgents := []string{
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/100.0.4896.75 Safari/537.36",
		"Mozilla/5.0 (Windows NT 6.1; WOW64; rv:64.0) Gecko/20100101 Firefox/64.0",

		// 新增Windows UA ↓
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/121.0.6167.160 Safari/537.36",
		"Mozilla/5.0 (Windows NT 10.0; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/118.0.5993.118 Safari/537.36",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:126.0) Gecko/20100101 Firefox/126.0",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/125.0.6422.142 Safari/537.36 Edg/125.0.2535.92",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.6367.118 Safari/537.36",
		"Mozilla/5.0 (Windows NT 6.3; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/123.0.6312.88 Safari/537.36",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.6266.95 Safari/537.36",
		"Mozilla/5.0 (Windows NT 11.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.6478.112 Safari/537.36",
		"Mozilla/5.0 (Windows NT 10.0; rv:124.0) Gecko/20100101 Firefox/124.0",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/123.0.6312.105 Safari/537.36 Edg/123.0.2420.81",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/127.0.6533.82 Safari/537.36",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.6367.91 Safari/537.36 Vivaldi/6.5.3206.57",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:125.0) Gecko/20100101 Firefox/125.0",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.6478.127 Safari/537.36 Edg/126.0.2572.98",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/125.0.6422.112 Safari/537.36 OPR/102.0.0.0",
		// 新增其他浏览器 UA ↓
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.6367.79 Safari/537.36 Edg/124.0.2478.51",
		"Mozilla/5.0 (Windows NT 10.0; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/123.0.6312.58 Safari/537.36 OPR/109.0.0.0",
		"Mozilla/5.0 (Windows NT 11.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/127.0.6533.82 Safari/537.36 Vivaldi/6.5.3206.57",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Brave Chrome/125.0.6422.142 Safari/537.36",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:127.0) Gecko/20100101 Firefox/127.0",
		"Mozilla/5.0 (Windows NT 6.3; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/128.0.6555.178 Safari/537.36 Edg/128.0.2735.88",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/129.0.6667.89 Safari/537.36 OPR/115.0.0.0",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/130.0.6723.98 Safari/537.36 Edg/130.0.1587.46",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.6478.112 YaBrowser/23.9.5.1102 Yowser/2.5 Safari/537.36",
		"Mozilla/5.0 (Windows NT 11.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/127.0.6533.82 Safari/537.36 Edg/127.0.2651.105",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/125.0.6422.142 Safari/537.36 Whale/3.21.192.18",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:128.0) Gecko/20100101 Firefox/128.0",
		"Mozilla/5.0 (Windows NT 6.1; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.6367.118 Safari/537.36 OPR/110.0.0.0",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/129.0.6723.112 Safari/537.36 Vivaldi/6.7.3320.47",
		// ... 其他现有代码保持不变 ...
	}

	// 随机选择一个 User-Agent
	rand.Seed(time.Now().UnixNano())
	return userAgents[rand.Intn(len(userAgents))]
}

// SetChromeOptions 返回chromedp配置选项
func SetChromeOptions() []chromedp.ExecAllocatorOption {
	return append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.NoDefaultBrowserCheck,
		chromedp.Flag("headless", false),
		chromedp.Flag("disable-hang-monitor", false),
		chromedp.Flag("ignore-certificate-errors", false),
		chromedp.Flag("disable-web-security", false),
		chromedp.Flag("start-maximized", false),
		chromedp.Flag("window-size", "1000,800"),
		chromedp.Flag("disable-extensions", true), // 禁用扩展程序，扩展程序可能会占用额外的资源
		chromedp.Flag("allow-scripting-gallery", true),

		chromedp.Flag("disable-notifications", true), // 禁用通知
		chromedp.Flag("disable-gpu", true),           // 禁用 GPU 加速，节省资源
		chromedp.Flag("enable-automation", false),
		chromedp.Flag("no-sandbox", true), // 禁用沙箱，提升性能（但降低安全性）
		//chromedp.Flag("blink-settings", "imagesEnabled=false"), //启动无图加载
		chromedp.Flag("useAutomationExtension", false),
		//chromedp.Flag("single-process", true), // 启用单进程模式，减少后台进程数量
		chromedp.Flag("disable-blink-features", "AutomationControlled"),
		//chromedp.Flag("mute-audio", false),
		chromedp.Flag("disable-background-networking", true), // 禁用后台网络活动
		chromedp.Flag("mute-audio", true),                    // 禁用音频，避免音频处理
		chromedp.Flag("hide-scrollbars", false),
		chromedp.UserAgent(generateRandomUserAgent()),
	)
}

// SetChromeOptions 返回chromedp配置选项
func SetChromeOptionsImgless() []chromedp.ExecAllocatorOption {
	return append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.NoDefaultBrowserCheck,
		chromedp.Flag("headless", false),
		chromedp.Flag("disable-hang-monitor", false),
		chromedp.Flag("ignore-certificate-errors", false),
		chromedp.Flag("disable-web-security", false),
		chromedp.Flag("start-maximized", false),
		chromedp.Flag("window-size", "1000,800"),
		chromedp.Flag("disable-extensions", true), // 禁用扩展程序，扩展程序可能会占用额外的资源
		chromedp.Flag("allow-scripting-gallery", true),
		chromedp.Flag("disable-gpu", true), // 禁用 GPU 加速，节省资源
		chromedp.Flag("enable-automation", false),
		chromedp.Flag("disable-auto-update", true),
		chromedp.Flag("disable-notifications", true),           // 禁用通知// 禁用自动更新
		chromedp.Flag("no-sandbox", true),                      // 禁用沙箱，提升性能（但降低安全性）
		chromedp.Flag("blink-settings", "imagesEnabled=false"), //启动无图加载
		chromedp.Flag("useAutomationExtension", false),
		//chromedp.Flag("single-process", true), // 启用单进程模式，减少后台进程数量
		chromedp.Flag("disable-blink-features", "AutomationControlled"),

		chromedp.Flag("disable-background-networking", true), // 禁用后台网络活动
		//chromedp.Flag("mute-audio", false),
		chromedp.Flag("mute-audio", true), // 禁用音频，避免音频处理
		chromedp.Flag("hide-scrollbars", false),
		chromedp.UserAgent(generateRandomUserAgent()),
	)
}

// SaveCookies 保存当前页面的Cookies到指定文件
func SaveCookies(ctx context.Context, filename string) error {
	var cookies []*network.Cookie
	if err := chromedp.Run(ctx,
		chromedp.ActionFunc(func(ctx context.Context) error {
			var err error
			cookies, err = network.GetCookies().Do(ctx)
			return err
		}),
	); err != nil {
		return fmt.Errorf("failed to get cookies: %w", err)
	}

	data, err := json.Marshal(cookies)
	if err != nil {
		return fmt.Errorf("failed to marshal cookies: %w", err)
	}
	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("写入cookie文件失败: %w", err)
	}
	log.Printf("cookies 保存到文件 %s", filename)
	return nil
}

// LoadCookies 加载Cookies
func LoadCookies(ctx context.Context, filename string) error {
	// 读取 Cookies 文件
	data, err := os.ReadFile(filename)
	if err != nil {
		return err
	}

	// 反序列化 Cookies
	var cookies []*network.CookieParam
	if err := json.Unmarshal(data, &cookies); err != nil {
		return err
	}

	// 设置 Cookies 到浏览器上下文
	return chromedp.Run(ctx,
		chromedp.ActionFunc(func(ctx context.Context) error {
			for _, cookie := range cookies {
				if err := network.SetCookie(cookie.Name, cookie.Value).
					WithDomain(cookie.Domain).
					WithPath(cookie.Path).
					WithHTTPOnly(cookie.HTTPOnly).
					WithSecure(cookie.Secure).
					WithExpires(cookie.Expires).
					Do(ctx); err != nil {
					return err
				}
			}
			return nil
		}),
	)
}
