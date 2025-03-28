package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"
)

type CookieParam struct {
	Name     string  `json:"name"`
	Value    string  `json:"value"`
	Domain   string  `json:"domain"`
	Path     string  `json:"path"`
	Expires  float64 `json:"expires"` // 使用Unix时间戳格式
	HTTPOnly bool    `json:"httpOnly"`
	Secure   bool    `json:"secure"`
	Session  bool    `json:"session"`  // 新增session字段
	SameSite string  `json:"sameSite"` // 新增sameSite
	Priority string  `json:"priority"` // 新增priority
}

// 提取账号名（格式：xxx----）
func ExtractAccountName(raw string) string {
	firstLine := strings.Split(raw, "\n")[0]
	parts := strings.Split(firstLine, "----")
	if len(parts) > 0 {
		return strings.TrimSpace(parts[0])
	}
	return ""
}

// 解析Cookie核心逻辑
func ParseCookies(raw string, account string) []CookieParam {
	var cookies []CookieParam

	// 模拟真实Cookie过期时间（示例使用30天有效期）
	expiration := time.Now().Add(30 * 24 * time.Hour).Unix()

	lines := strings.Split(raw, "\n")
	for _, line := range lines {
		for _, part := range strings.Split(line, ";") {
			part = strings.TrimSpace(part)
			if part == "" || strings.Contains(part, "----") {
				continue
			}

			kv := strings.SplitN(part, "=", 2)
			if len(kv) != 2 {
				continue
			}

			name := strings.TrimSpace(kv[0])
			value := strings.TrimSpace(kv[1])

			cookie := CookieParam{
				Name:     name,
				Value:    value,
				Domain:   ".instagram.com",
				Path:     "/",
				Expires:  float64(expiration),
				HTTPOnly: isHttpOnly(name), // 根据Cookie名称判断
				Secure:   true,
				Session:  false,
				SameSite: "Lax",
				Priority: "Medium",
			}

			// 特殊处理sessionid等敏感Cookie
			if name == "sessionid" || name == "csrftoken" {
				cookie.HTTPOnly = true
				cookie.SameSite = "Strict"
			}

			cookies = append(cookies, cookie)
		}
	}
	return cookies
}

func isHttpOnly(name string) bool {
	httpOnlyCookies := map[string]bool{
		"sessionid": true,
		"ig_did":    true,
		"datr":      true,
		"rur":       true,
	}
	return httpOnlyCookies[strings.ToLower(name)]
}

func CheckAndCreateDir(file string) error {
	// 检查 logs 目录是否存在
	_, err := os.Stat(file)
	if err == nil {
		// 目录已存在
		return nil
	}
	if os.IsNotExist(err) {
		// 目录不存在，创建它
		err := os.MkdirAll(file, 0755)
		if err != nil {
			return fmt.Errorf("创建目录失败: %w", err)
		}
	}

	return nil
}

func main() {
	//transaction.SendMessageNew()
	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Println("请粘贴Cookie:【输入完成后空行回车结束】")
		var lines []string
		for scanner.Scan() {
			line := scanner.Text()
			if line == "" {
				break
			}
			lines = append(lines, line)
		}

		raw := strings.Join(lines, "\n")

		// 提取账号名
		account := ExtractAccountName(raw)
		if account == "" {
			fmt.Println("无法提取账号名")
			continue
		}
		cookies := ParseCookies(raw, account)
		if len(cookies) == 0 {
			fmt.Println("未找到有效Cookie")
			continue
		}

		// 生成文件名
		filename := fmt.Sprintf("%s.json", account)

		// 保存文件
		data, _ := json.MarshalIndent(cookies, "", "  ")
		os.WriteFile(filename, data, 0644)
		fmt.Printf("Cookies已保存到 %s\n", filename)
	}
}
