package utils

import (
	"encoding/json"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/chromedp/cdproto/network"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
)

// getCode 通过HTTP请求获取2FA验证码
func GetCode(rawUrl string) (string, error) {
	resp, err := http.Get(rawUrl)
	if err != nil {
		return "", fmt.Errorf("failed to get 2FA code: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(body)))
	if err != nil {
		return "", fmt.Errorf("failed to parse HTML: %w", err)
	}

	code := strings.TrimSpace(doc.Find("#code").Text())
	if code == "" {
		return "", fmt.Errorf("没有发现2FA code")
	}
	return code, nil
}

// ToCookieStr 读取 Cookies 文件并转换为 HTTP 请求所用的 Cookie 字符串
func ToCookieStr(cookiefile string) (string, error) {
	data, err := os.ReadFile(cookiefile)
	if err != nil {
		log.Printf("读取Cookies文件失败: %s 错误: %v\n", cookiefile, err)
		return "", err
	}

	var cookies []*network.CookieParam
	if err := json.Unmarshal(data, &cookies); err != nil {
		return "", err
	}
	var myCookie string
	for _, cookie := range cookies {
		myCookie += fmt.Sprintf("%s=%s;", cookie.Name, cookie.Value)
	}
	return myCookie, nil
}
