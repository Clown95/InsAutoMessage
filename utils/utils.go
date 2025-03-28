package utils

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

var UsedAccounts = make(map[string]bool)
var mu sync.Mutex // 用于保护并发访问的锁
// 获取未使用的登录账号
func GetUniqueLoginAccount(path string) (string, error) {
	mu.Lock() // 加锁，防止并发时修改 UsedAccounts
	defer mu.Unlock()

	files, err := GetJSONFiles(path)
	if err != nil {
		return "", err
	}

	// 筛选出没有被使用的账号
	var unusedFiles []string
	for _, file := range files {
		if !UsedAccounts[file] {
			unusedFiles = append(unusedFiles, file)
		}
	}

	// 如果没有剩余未使用的账号，返回错误
	if len(unusedFiles) == 0 {
		return "", errors.New("所有账号都已被用")
	}

	// 随机选择一个未使用的账号
	account := unusedFiles[rand.Intn(len(unusedFiles))]

	// 标记该账号已被使用
	UsedAccounts[account] = true

	return account, nil
}

func GetJSONFiles(path string) ([]string, error) {
	var jsonFiles []string

	// 读取当前目录
	files, err := os.ReadDir(path)
	if err != nil {
		return nil, fmt.Errorf("读取目录失败: %w", err)
	}

	// 遍历目录项
	for _, file := range files {
		// 跳过目录
		if file.IsDir() {
			continue
		}

		// 获取文件信息以检查符号链接
		info, err := file.Info()
		if err != nil {
			continue // 跳过无法获取信息的文件
		}

		// 跳过非常规文件（如设备文件、套接字等）
		if !info.Mode().IsRegular() {
			continue
		}

		// 检查文件扩展名（不区分大小写）
		if strings.EqualFold(filepath.Ext(info.Name()), ".json") {
			// 获取绝对路径
			//fullPath, err := filepath.Abs(info.Name())
			fullPath, err := filepath.Abs(filepath.Join(path, info.Name()))
			if err != nil {
				continue // 跳过路径解析失败的文件
			}
			jsonFiles = append(jsonFiles, fullPath)
		}
	}

	return jsonFiles, nil
}

func GetLoginAccount(path string) (string, error) {

	files, err := GetJSONFiles(path)
	if err != nil {
		return "", err
	}
	//fmt.Println(files)

	if len(files) == 0 {
		return "", errors.New("没有登录的账号")

	}
	account := files[rand.Intn(len(files))]
	//time.Sleep(111)
	return account, nil

}

/*
// readAccountsFromExcel 读取Excel中的账号信息
func readAccountsFromExcel(filename string) ([]Db.LoginAccount, error) {
	f, err := excelize.OpenFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open excel file: %w", err)
	}
	defer func() {
		if err := f.Close(); err != nil {
			log.Printf("failed to close excel file: %v", err)
		}
	}()

	sheetName := f.GetSheetName(0)
	if sheetName == "" {
		return nil, fmt.Errorf("no sheet found in excel file")
	}

	rows, err := f.GetRows(sheetName)
	if err != nil {
		return nil, fmt.Errorf("failed to get rows: %w", err)
	}

	var accounts []Db.LoginAccount
	for i, row := range rows {
		if len(row) < 3 {
			log.Printf("skipping row %d: insufficient columns", i)
			continue
		}
		account := Db.LoginAccount{
			Account:  strings.TrimSpace(row[0]),
			Password: strings.TrimSpace(row[1]),
			Hash:     strings.TrimSpace(row[2]),
		}
		accounts = append(accounts, account)
	}
	return accounts, nil
}


*/

// AutoDel 自动删除指定目录下所有文件名中包含 .限制消息发送 的后缀
func AutoReName(currentDir string) error {
	return filepath.Walk(currentDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if strings.HasSuffix(info.Name(), ".限制消息发送") {
			return renameFile(path)
		}
		return nil
	})
}

// renameFile 重命名文件，将文件名中的 .限制消息发送 后缀删除
func renameFile(path string) error {
	newName := strings.TrimSuffix(filepath.Base(path), ".限制消息发送")
	newPath := filepath.Join(filepath.Dir(path), newName)
	if err := os.Rename(path, newPath); err != nil {
		log.Printf("重命名文件 %s 时出错: %v", path, err)
		return err
	}
	log.Printf("文件 %s 已重命名为 %s", path, newPath)
	return nil
}

// 安全重命名函数
func SafeRename(oldName, newName string) error {
	// 获取文件路径对应的专用锁
	fileLocks.Lock()
	lock, exists := fileLocks.locks[oldName]
	if !exists {
		lock = &sync.Mutex{}
		fileLocks.locks[oldName] = lock
	}
	fileLocks.Unlock()

	// 加锁操作
	lock.Lock()
	defer lock.Unlock()

	// 检查文件是否已被处理
	//newName := accountFile + ".账号异常"
	if _, err := os.Stat(newName); err == nil {
		return fmt.Errorf("文件 %s 已存在", newName)
	}

	// 执行重命名
	if err := os.Rename(oldName, newName); err != nil {
		return fmt.Errorf("重命名失败: %w", err)
	}

	// 清理锁映射（可选）
	fileLocks.Lock()
	delete(fileLocks.locks, oldName)
	fileLocks.Unlock()

	return nil
}

// 全局文件操作锁（按文件路径粒度控制）
var fileLocks = struct {
	sync.Mutex
	locks map[string]*sync.Mutex
}{
	locks: make(map[string]*sync.Mutex),
}

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

func main11() {
	//transaction.SendMessageNew()

	raw := `kty.helen.k687dc----j4o9l3qCmaria9----android-54e0ed07c398a167;41a07624-fc92-43a8-fb5e-cbe862926046;86e7ddfa-0663-4ab1-2f06-b5c178bef886;4388b3e0-0bfe-428b-ba5b-2ba3e814bdc3|rur=LDC
71697404922
1773361469----01f7879d1e821fd09956810131b2b5fcd69c4d59b810d8699582bf78ee8d775548159f33; mid=Z4iCIAAAAAHh_y930PShpoUL7LsB; ds_user_id=71697404922; csrftoken=ud4BoA9CooPQBU54XRTBUuLNgW20ls9i; sessionid=71697404922%3AABXaXRmTOOEfP5%3A0%3AAYf-vdSNj6fP1vzIEryj3viLwkxYkfzUQ502hP7H9Q; Authorization=Bearer IGT:2:eyJkc191c2VyX2lkIjoiNzE2OTc0MDQ5MjIiLCJzZXNzaW9uaWQiOiI3MTY5NzQwNDkyMiUzQUFCWGFYUm1UT09FZlA1JTNBMCUzQUFZZi12ZFNOajZmUDF2eklFcnlqM3ZpTHdreFlrZnpVUTUwMmhQN0g5USJ9; IG-U-DS-USER-ID=71697404922; IG-INTENDED-USER-ID=71697404922; IG-U-RUR=LDC
71697404922
1773361469:01f7879d1e821fd09956810131b2b5fcd69c4d59b810d8699582bf78ee8d775548159f33; X-MID=Z4iCIAAAAAHh_y930PShpoUL7LsB; X-IG-WWW-Claim=hmac.AR2lh1cLci5zNIp4tutnoIgFVszCzq1aJZIaoFbrNjw-iTzW;` // 替换为你的原始字符串

	// 提取账号名
	account := ExtractAccountName(raw)
	if account == "" {
		fmt.Println("无法提取账号名")
		return
	}
	cookies := ParseCookies(raw, account)
	if len(cookies) == 0 {
		fmt.Println("未找到有效Cookie")
		return
	}

	// 生成文件名
	filename := fmt.Sprintf("%s.json", account)

	// 保存文件
	data, _ := json.MarshalIndent(cookies, "", "  ")
	os.WriteFile(filename, data, 0644)
	fmt.Printf("Cookies已保存到 %s\n", filename)

}
