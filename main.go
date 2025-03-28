package main

import (
	"InsAutoMessage/config"
	DB "InsAutoMessage/database"
	"InsAutoMessage/transaction"
	"InsAutoMessage/utils"
	"bufio"
	"fmt"
	"github.com/fatih/color"
	"github.com/mattn/go-runewidth"
	"io"
	"log"
	"math/rand"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"
)

func init() {
	var configMessage string
	var err error
	if _, err := os.Stat(config.ConfigFileName); os.IsNotExist(err) {
		// 文件不存在，创建并写入默认配置
		configMessage = "配置文件不存在，创建并写入默认配置..."

		if err := config.CreateDefaultConfig(); err != nil {
			//log.Fatalf("创建配置文件失败: %v", err)
			configMessage += fmt.Sprintf("\n创建配置文件失败: %v", err)
		}
	} else {
		// 文件存在，读取配置文件
		configMessage = "配置文件已存在，加载配置..."
	}
	config.AppCfg, err = config.LoadConfig()
	config.Dsn = config.AppCfg.MysqlDSN

	config.GormDb, err = DB.InitDB(config.Dsn)
	if err != nil {
		return
	}

	if err != nil {
		configMessage += fmt.Sprintf("\n读取配置文件错误: %v", err)
	}

	err = utils.CheckAndCreateDir("logs")
	if err != nil {
		return
	}
	err = utils.CheckAndCreateDir("cookies")
	if err != nil {
		return
	}

	go func() {
		utils.IsAllow()
	}()
}

var (
	cyan    = color.New(color.FgCyan).SprintFunc()
	green   = color.New(color.FgGreen).SprintFunc()
	red     = color.New(color.FgRed).SprintFunc()
	yellow  = color.New(color.FgYellow).SprintFunc()
	magenta = color.New(color.FgMagenta).SprintFunc()
)

func main() {
	// 新增：初始化日志文件
	startTime := time.Now().Format("2006-01-02_15-04-05")
	logFileName := "logs/" + startTime + ".log"

	logFile, err := os.Create(logFileName)
	if err != nil {
		log.Fatal("无法创建日志文件:", err)
	}
	defer logFile.Close()

	multiWriter := io.MultiWriter(os.Stdout, logFile)
	log.SetOutput(multiWriter)

	/*
		go func() {
			http.ListenAndServe("localhost:6060", nil)
		}()


	*/

	go func() {
		ticker := time.NewTicker(35 * time.Minute)
		defer ticker.Stop()

		for {
			log.Println("恢复限制消息发送的账号")
			select {
			case <-ticker.C:
				err := utils.AutoReName("cookies")
				if err != nil {
					return
				}
				utils.UsedAccounts = make(map[string]bool)

			}
		}
	}()

	if config.AppCfg.NeedReplyMessage == "true" {
		go func() {

			timeInterval := config.AppCfg.CheckReplyTimeInterval

			if timeInterval == 0 {
				timeInterval = 30
			}
			ticker := time.NewTicker(time.Duration(timeInterval) * time.Minute)
			defer ticker.Stop()

			for {
				log.Println("开始监控是否有回复消息")
				select {
				case <-ticker.C:
					handleReplyMessages()

				}
			}
		}()
	}

	if config.AppCfg.NeedAddAccount == "true" {
		go func() {
			timeInterval := config.AppCfg.CheckAddNewAccountTimeInterval

			//fmt.Println(timeInterval)
			if timeInterval == 0 {
				timeInterval = 5
			}
			ticker := time.NewTicker(time.Duration(timeInterval) * time.Minute)
			defer ticker.Stop()

			for {
				log.Println("开始监控是否有需要补号")
				select {
				case <-ticker.C:
					handleIsNeedAddNewAccount()
				}
			}
		}()
	}

	config.AppCfg, _ = config.LoadConfig()
	// 初始化随机数种子（仅需一次）
	rand.Seed(time.Now().UnixNano())
	handleIsNeedAddNewAccount()

	showMenu()

}

func clearScreen() {
	switch runtime.GOOS {
	case "linux", "darwin":
		cmd := exec.Command("clear")
		cmd.Stdout = os.Stdout
		cmd.Run()
	case "windows":
		cmd := exec.Command("cmd", "/c", "cls")
		cmd.Stdout = os.Stdout
		cmd.Run()
	default:
		fmt.Println("\n\n\n")
	}
}

func showMenu() {
	menuItems := []struct {
		number  int
		label   string
		handler func()
	}{
		{1, "批量账号登录", handleLogin},
		{2, "扫描网页获取博主", handleFetchBlogger},
		{3, "通过API获取博主(需代理)", handleFetchBloggerAPI},
		{4, "批量发送消息(旧)", handleSendMessages},
		{5, "批量发送消息(新)", handleSendMessagesNew},
		{6, "批量回复消息【启动自动监控】", handleReplyMessages},
		{7, "批量检测账号", handleCheckAccount},
		{8, "配置管理", showConfigMenu},
		{9, "从文本导入账号", handleImportAccount},
		{0, "导入地点ID(需代理)【备用】", handleImportLocation},
	}

	for {
		clearScreen()
		printMenuHeader()
		printCurrentConfigStatus()

		// 打印菜单项
		for _, item := range menuItems {

			fmt.Println(cyan("|"+magenta("▶")+center(fmt.Sprintf("%d. %s", item.number, item.label), 56)) + cyan("|"))
		}

		printMenuFooter()

		choice := getUserChoice(0, len(menuItems)-1)
		if choice == -1 {
			continue
		}

		// 执行对应操作
		for _, item := range menuItems {
			if item.number == choice {
				clearScreen()
				printActionHeader(item.label)
				item.handler()
				pause()
				break
			}
		}
	}
}
func center(s string, width int) string {
	sWidth := runewidth.StringWidth(s)
	padding := width - sWidth
	if padding <= 0 {
		return s
	}
	leftPadding := padding / 2
	rightPadding := padding - leftPadding
	return strings.Repeat(" ", leftPadding) + s + strings.Repeat(" ", rightPadding)
}

func printMenuHeader() {
	fmt.Println()
	fmt.Println(cyan("┏" + strings.Repeat("━", 57) + "┓"))

	fmt.Println(cyan("|" + center(" 自动化演示工具,仅供学习交流使用,请勿用于非法用途 ", 57) + "|"))
	fmt.Println(cyan("┣" + strings.Repeat("━", 57) + "┫"))
}

func printMenuFooter() {
	fmt.Println(cyan("┣" + strings.Repeat("━", 57) + "┫"))
	fmt.Println(cyan("┃" + center("提示: 输入数字选择操作，q 返回上级菜单", 57) + "┃"))
	fmt.Println(cyan("┗" + strings.Repeat("━", 57) + "┛"))
}

func printActionHeader(actionName string) {
	fmt.Println(cyan("┏" + strings.Repeat("━", 57) + "┓"))
	fmt.Printf(cyan("┃ 正在执行: %-30s ┃\n"), yellow(actionName))
	fmt.Println(cyan("┣" + strings.Repeat("━", 57) + "┫"))
}

func printCurrentConfigStatus() {
	conf := config.AppCfg

	account, _ := utils.GetJSONFiles("./cookies")

	fmt.Println(cyan("|") + green(center(fmt.Sprintf("已登录账号数: %d ", len(account)), 57)) + cyan("|"))
	fmt.Println(cyan("|") + green(center(fmt.Sprintf("代理地址:%s", conf.Proxyaddr), 57)) + cyan("|"))

	//fmt.Println(green(status))
	fmt.Println(cyan("┣" + strings.Repeat("━", 57) + "┫"))
}

func getUserChoice(min, max int) int {
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Print(yellow("请输入选项 (", min, "-", max, "): "))
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		if strings.ToLower(input) == "q" {
			return -1
		}

		choice, err := strconv.Atoi(input)
		if err != nil || choice < min || choice > max {
			fmt.Println(red("错误：请输入", min, "-", max, "之间的有效数字"))
			continue
		}

		return choice
	}
}

// 示例处理函数
func handleLogin() {
	conf := config.AppCfg
	if conf.AccountLoginNum > 0 && conf.LoginNum > 0 {
		loginNum := conf.LoginNum
		if loginNum > conf.AccountLoginNum {
			loginNum = conf.AccountLoginNum
		}
		transaction.LoginWork(loginNum, conf.AccountLoginNum)
	}
	fmt.Println(green("\n✓ 登录操作完成"))
}

func handleCheckAccount() {

	files, err := utils.GetJSONFiles("./cookies")
	if err != nil {
		return
	}
	if len(files) == 0 {
		log.Println("没有已登陆账号")

	}
	for i, file := range files {

		log.Println("正在检查", i+1, "个账号:", file)
		err := transaction.CheckAccountStatus(file)
		if err != nil {
			return
		}

	}
	log.Println("检查完成")

}

func handleReplyMessages() {

	txts := config.AppCfg.ReplyMessageTexts

	if len(txts) == 0 {
		log.Println("没有设置回复文本")
		return
	}

	files, err := utils.GetJSONFiles("./cookies")
	if err != nil {
		return
	}

	if len(files) == 0 {
		log.Println("没有已登陆账号")

	}

	for i, file := range files {

		log.Println("正在检查", i+1, "个账号:", file)
		transaction.ReplyWork(file, txts)

	}
	log.Println("检查完成")

}

func handleIsNeedAddNewAccount() {
	files, err := utils.GetJSONFiles("./cookies")
	if err != nil {
		return
	}

	num := config.AppCfg.AddAccountNum

	if len(files) > num {
		log.Println("当前账号数量大于指定补号数量,不需要补号")
		return
	}

	count, err := DB.GetAvailableAccountCount(config.GormDb)
	if err != nil {
		return
	}

	if count == 0 {
		log.Println("数据没有可用的未登录账号")
		return
	}

	if num > int(count) {
		log.Printf("指定补号数量:%d,实际可用账号数量:%d\n", num, count)
		log.Printf("自动设置补号数量为:%d\n", count)
		num = int(count)
	}

	if len(files) < num {
		log.Printf("需要补 %d 个账号", num-len(files))
		for i := 0; i < num-len(files); i++ {
			log.Printf("开始补第%d个号", i+1)
			err := transaction.AddNewAccount(0)
			if err != nil {
				return
			}
		}
	}
}

func handleImportLocation() {

	proxyaddr := config.AppCfg.Proxyaddr

	if proxyaddr == "" {
		log.Println("没有在配置设置代理IP")

	}
	account, _ := utils.GetLoginAccount("./cookies")

	if len(account) > 0 {

		cookie, err := utils.ToCookieStr(account)
		if err != nil {
			return
		}
		transaction.GetlocationID(proxyaddr, cookie)

	} else {
		fmt.Println(red("没有可用账号"))
	}
}

func handleFetchBlogger() {
	conf := config.AppCfg
	// 调用实际的博主获取逻辑
	crawlersBloggerNum := conf.CrawlersBloggerNum
	if crawlersBloggerNum > 0 {
		transaction.SearchBloggerWork(crawlersBloggerNum)
	}
}

func handleFetchBloggerAPI() {
	conf := config.AppCfg
	// 调用实际的博主获取逻辑
	crawlersBloggerNum := conf.CrawlersBloggerNum
	if crawlersBloggerNum > 0 {
		transaction.SearchBloggerWorkAPi(crawlersBloggerNum)
	}
}

func handleSendMessages() {
	conf := config.AppCfg
	sendBloggerNum := conf.SendBloggerNum
	sendMessageNum := conf.SendMessageNum
	sendMessageTimeInterval := conf.SendMessageTimeInterval

	//messageTxt := ""
	messageTxts := conf.SendMessageTexts
	if len(messageTxts) > 0 {
		//messageTxt = messageTxts[rand.Intn(len(messageTxts))]
		if sendMessageNum > 0 && sendBloggerNum > 0 && sendMessageTimeInterval > 0 {
			if sendMessageNum > sendBloggerNum {
				sendMessageNum = sendBloggerNum
			}

			//transaction.SendMessageWork(sendMessageNum, sendBloggerNum, sendMessageTimeInterval, messageTxts)
			transaction.SendMessageWorker(sendMessageNum, messageTxts, sendMessageTimeInterval)

		}
	}

}

func handleSendMessagesNew() {
	conf := config.AppCfg
	sendBloggerNum := conf.SendBloggerNum
	sendMessageNum := conf.SendMessageNum
	sendMessageTimeInterval := conf.SendMessageTimeInterval

	//messageTxt := ""
	messageTxts := conf.SendMessageTexts
	if len(messageTxts) > 0 {
		//messageTxt = messageTxts[rand.Intn(len(messageTxts))]
		if sendMessageNum > 0 && sendBloggerNum > 0 && sendMessageTimeInterval > 0 {
			if sendMessageNum > sendBloggerNum {
				sendMessageNum = sendBloggerNum
			}

			transaction.SendMessageWorkOne(sendMessageNum, messageTxts, sendMessageTimeInterval)

		}
	}

}

func handleImportAccount() {

	fmt.Print(yellow("请输入要导入的账号文件路径: "))
	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	filename := ""

	if input == "" {
		fmt.Println(red("未输入文件路径,使用默认文件\"account.txt\""))
		filename = "account.txt"
		config.AppCfg.AccountTxtFile = "account.txt"
	} else {
		config.AppCfg.AccountTxtFile = input
		filename = input
	}

	config.SaveConfig(config.AppCfg)

	transaction.TxtToDb(filename)

}

func showConfigMenu() {
	subItems := []struct {
		number  int
		label   string
		handler func()
	}{
		{1, "编辑配置文件", openConfigFile},
		{2, "重新加载配置", reloadConfig},
		{3, "设置代理地址", setProxy},
		{9, "返回主菜单", func() {}},
	}

	for {
		clearScreen()
		printActionHeader("配置管理")

		for _, item := range subItems {
			//	fmt.Printf(cyan("┃ %s %d. %-20s ┃\n"), magenta("▶"), item.number, item.label)

			fmt.Println(cyan("|" + magenta("▶") + center(fmt.Sprintf("%d. %s", item.number, item.label), 56) + "|"))

		}

		printMenuFooter()

		choice := getUserChoice(1, 9)
		if choice == -1 {
			return
		}

		for _, item := range subItems {
			if item.number == choice {
				clearScreen()
				printActionHeader(item.label)
				item.handler()
				if item.number != 9 { // 非返回选项需要暂停
					pause()
				}
				break
			}
		}

		if choice == 9 {
			return
		}
	}
}

func reloadConfig() {
	config.AppCfg, _ = config.LoadConfig()
	fmt.Println(green("✓ 配置已重新加载"))
}

func setProxy() {
	fmt.Print(yellow("请输入新的代理地址 (格式: ip:port): "))
	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	config.AppCfg.Proxyaddr = input
	config.SaveConfig(config.AppCfg)
	fmt.Println(green("✓ 代理地址已更新"))
}

func pause() {
	fmt.Print(yellow("\n按 Enter 继续..."))
	bufio.NewReader(os.Stdin).ReadBytes('\n')
}

// 打开配置文件
func openConfigFile() {
	//clearScreen()

	filename := config.ConfigFileName
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("notepad", filename)
	case "darwin":
		cmd = exec.Command("open", "-t", filename)
	case "linux":
		cmd = exec.Command("xdg-open", filename)
	default:
		fmt.Println(red("不支持的操作系统"))
		return
	}

	if err := cmd.Start(); err != nil {
		fmt.Println(red("打开配置文件失败:"), err)
	} else {
		fmt.Println(green("配置文件已打开"))
	}
	pause()
}
