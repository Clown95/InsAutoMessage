package config

import (
	"fmt"
	"gopkg.in/ini.v1"
	"gorm.io/gorm"
	"os"
)

var AppCfg *AppConfig

var GormDb *gorm.DB

var Dsn string

var ConfigFileName = "instagram.ini"

// 默认配置内容
var defaultConfig = `
	app_mode = production

	[addr]
	Proxyaddr = http://127.0.0.1:7897  #代理IP地址
	MysqlDSN = "caijue:Zhijian123@tcp(rm-bp1z2996838g3yp5jko.mysql.rds.aliyuncs.com:3306)/ins?charset=utf8mb4&parseTime=True" # 数据库链接字符串

	[setTableName]
	IndiaBloggersTableName ="india_bloggers" # 印度博主表名
	LocationTableName ="location"    # 地点ID表名
	LoginAccountTableName ="login_accounts" # 登录账号表名


	[setCrawlers]
	#采集数据
	CrawlersBloggerNum = 1  # 开启采集博主窗口数量
    CrawlersTimeInterval =5 # 采集时间间隔【最大值】  （单位分钟）

	AccountTxtFile ="account.txt"
	#登录账号
	AccountLoginNum = 1 #批量登录账号数量
	LoginNum =1 #登录账号窗口数量
	

	#是否需要补号
	NeedAddAccount = true # 是否需要补号 true 是 false 否
	AddAccountNum = 1 # 补号数量	
	CheckAddNewAccountTimeInterval =5 #监控添加新账号时间间隔  （单位分钟）

	# 发送消息
	SendBloggerNum = 50 #获取博主数量
	SendMessageNum = 1 # 发送消息浏览器窗口数量
	SendMessageTimeInterval = 10  # 发送消息间隔 （单位秒）

	# 消息列表 如需增加在下面追加Txt[数字] ，如需删除则删除对应Txt[数字]	
	[SendMessageTxt]
	Txt1 = "Paid promotion cooperation, sincerely invite excellent agents to cooperate, generous commissions, low threshold, everyone can do it! Welcome to contact us. भुगतान पदोन्नति सहयोग, ईमानदारी से उत्कृष्ट एजेंटों को सहयोग करने के लिए आमंत्रित करते हैं, उदार कमीशन, कम सीमा, हर कोई यह कर सकता है! हमसे संपर्क करने के लिए आपका स्वागत है। 📱 WhatsApp: +44 7481 537226 📲 telegraph: @DailyMoney007"


	#消息回复设置
	NeedReplyMessage = true # 是否需要监控回复消息 true 是 false 否
	CheckReplyTimeInterval =60 # 监控回复消息时间间隔  （单位分钟）

	# 回复消息列表 如需增加在下面追加ReplyTxt[数字] ，如需删除则删除对应ReplyTxt[数字]	
	[ReplyMessageTxt]
	ReplyTxt1 = "Paid promotion cooperation, sincerely invite excellent agents to cooperate, generous commissions, low threshold, everyone can do it! Welcome to contact us. भुगतान पदोन्नति सहयोग, ईमानदारी से उत्कृष्ट एजेंटों को सहयोग करने के लिए आमंत्रित करते हैं, उदार कमीशन, कम सीमा, हर कोई यह कर सकता है! हमसे संपर्क करने के लिए आपका स्वागत है। 📱 WhatsApp: +44 7481 537226 📲 telegraph: @DailyMoney007"
	


`

type AppConfig struct {
	MysqlDSN  string
	Proxyaddr string

	IndiaBloggersTableName string // 印度博主表名
	LocationTableName      string // 地点ID表名
	LoginAccountTableName  string // 登录账号表名

	CrawlersBloggerNum   int // 开启采集博主窗口数量
	CrawlersTimeInterval int //采集时间间隔【最大值】  （单位分钟）

	AccountTxtFile  string //账号文件路径
	AccountLoginNum int    //批量登录账号数量
	LoginNum        int    //登录账号窗口数量

	NeedAddAccount                 string // 是否需要补号 true 是 false 否
	AddAccountNum                  int    // 补号数量
	CheckAddNewAccountTimeInterval int    //监控添加新账号时间间隔  （单位分钟）

	SendBloggerNum          int      //	获取博主数量
	SendMessageNum          int      // 发送消息浏览器窗口数量
	SendMessageTimeInterval int      // 发送消息间隔 （单位秒）
	SendMessageTexts        []string // 发送消息内容

	NeedReplyMessage       string   // 是否需要监控回复消息 true 是 false 否
	CheckReplyTimeInterval int      // 监控回复消息时间间隔  （单位分钟）
	ReplyMessageTexts      []string // 回复消息内容
}

func LoadConfig() (*AppConfig, error) {

	cfg, err := ini.Load(ConfigFileName)
	if err != nil {
		return nil, fmt.Errorf("unable to read config file: %v", err)
	}

	proxyaddr := cfg.Section("addr").Key("Proxyaddr").String()
	mysqlDSN := cfg.Section("addr").Key("MysqlDSN").String()

	indiaBloggersTableName := cfg.Section("setTableName").Key("IndiaBloggersTableName").String()
	locationTableName := cfg.Section("setTableName").Key("LocationTableName").String()
	loginAccountTableName := cfg.Section("setTableName").Key("LoginAccountTableName").String()

	crawlersBloggerNum, _ := cfg.Section("setCrawlers").Key("CrawlersBloggerNum").Int()
	crawlersTimeInterval, _ := cfg.Section("setCrawlers").Key("CrawlersTimeInterval").Int()

	accountTxtFile := cfg.Section("setCrawlers").Key("AccountTxtFile").String()
	accountLoginNum, _ := cfg.Section("setCrawlers").Key("AccountLoginNum").Int()
	loginNum, _ := cfg.Section("setCrawlers").Key("LoginNum").Int()

	needAddAccount := cfg.Section("setCrawlers").Key("NeedAddAccount").String()
	addAccountNum, _ := cfg.Section("setCrawlers").Key("AddAccountNum").Int()
	checkAddNewAccountTimeInterval, _ := cfg.Section("setCrawlers").Key("CheckAddNewAccountTimeInterval").Int()

	sendBloggerNum, _ := cfg.Section("setCrawlers").Key("SendBloggerNum").Int()
	sendMessageNum, _ := cfg.Section("setCrawlers").Key("SendMessageNum").Int()
	sendMessageTimeInterval, _ := cfg.Section("setCrawlers").Key("SendMessageTimeInterval").Int()

	messageTxtSection := cfg.Section("SendMessageTxt")
	// 收集所有以 "server" 开头的键
	var sendTexts []string
	for _, value := range messageTxtSection.Keys() {
		//fmt.Println(key)
		if value.Name()[:3] == "Txt" { // 检查前缀
			sendTexts = append(sendTexts, value.String())
		}
	}

	needReplyMessage := cfg.Section("setCrawlers").Key("NeedReplyMessage").String()
	checkReplyTimeInterval, _ := cfg.Section("setCrawlers").Key("CheckReplyTimeInterval").Int()
	ReplyTxtSection := cfg.Section("ReplyMessageTxt")

	// 收集所有以 "server" 开头的键
	var replyTexts []string
	for _, value := range ReplyTxtSection.Keys() {
		//fmt.Println(key)
		if value.Name()[:8] == "ReplyTxt" { // 检查前缀
			replyTexts = append(replyTexts, value.String())
		}
	}

	return &AppConfig{

		MysqlDSN:  mysqlDSN,
		Proxyaddr: proxyaddr,

		IndiaBloggersTableName: indiaBloggersTableName,
		LocationTableName:      locationTableName,
		LoginAccountTableName:  loginAccountTableName,

		CheckReplyTimeInterval:         checkReplyTimeInterval,
		SendMessageTimeInterval:        sendMessageTimeInterval,
		CheckAddNewAccountTimeInterval: checkAddNewAccountTimeInterval,
		CrawlersTimeInterval:           crawlersTimeInterval,
		CrawlersBloggerNum:             crawlersBloggerNum,
		SendBloggerNum:                 sendBloggerNum,
		SendMessageNum:                 sendMessageNum,
		AccountLoginNum:                accountLoginNum,
		LoginNum:                       loginNum,

		AddAccountNum:    addAccountNum,
		NeedAddAccount:   needAddAccount,
		NeedReplyMessage: needReplyMessage,

		AccountTxtFile:    accountTxtFile,
		SendMessageTexts:  sendTexts,
		ReplyMessageTexts: replyTexts,
	}, nil
}

// SaveConfig 将配置保存到文件
func SaveConfig(config *AppConfig) error {
	// 创建或打开配置文件
	cfg := ini.Empty()

	cfg.Section("addr").Key("Proxyaddr").SetValue(config.Proxyaddr)
	cfg.Section("addr").Key("MysqlDSN").SetValue(config.MysqlDSN)

	cfg.Section("setTableName").Key("IndiaBloggersTableName").SetValue(config.IndiaBloggersTableName)
	cfg.Section("setTableName").Key("LocationTableName").SetValue(config.LocationTableName)
	cfg.Section("setTableName").Key("LoginAccountTableName").SetValue(config.LoginAccountTableName)

	cfg.Section("setCrawlers").Key("CrawlersBloggerNum").SetValue(fmt.Sprintf("%d", config.CrawlersBloggerNum))
	cfg.Section("setCrawlers").Key("CrawlersTimeInterval").SetValue(fmt.Sprintf("%d", config.CrawlersTimeInterval))
	//
	cfg.Section("setCrawlers").Key("AccountTxtFile").SetValue(config.AccountTxtFile)
	cfg.Section("setCrawlers").Key("AccountLoginNum").SetValue(fmt.Sprintf("%d", config.AccountLoginNum))
	cfg.Section("setCrawlers").Key("LoginNum").SetValue(fmt.Sprintf("%d", config.LoginNum))
	//
	cfg.Section("setCrawlers").Key("NeedAddAccount").SetValue(config.NeedAddAccount)
	cfg.Section("setCrawlers").Key("AddAccountNum").SetValue(fmt.Sprintf("%d", config.AddAccountNum))
	cfg.Section("setCrawlers").Key("CheckAddNewAccountTimeInterval").SetValue(fmt.Sprintf("%d", config.CheckAddNewAccountTimeInterval))
	//
	cfg.Section("setCrawlers").Key("SendBloggerNum").SetValue(fmt.Sprintf("%d", config.SendBloggerNum))
	cfg.Section("setCrawlers").Key("SendMessageNum").SetValue(fmt.Sprintf("%d", config.SendMessageNum))
	cfg.Section("setCrawlers").Key("SendMessageTimeInterval").SetValue(fmt.Sprintf("%d", config.SendMessageTimeInterval))

	for key, value := range config.SendMessageTexts {

		newKey := fmt.Sprintf("Txt%d", key+1)
		cfg.Section("MessageTxt").Key(newKey).SetValue(value)
	}
	//

	cfg.Section("setCrawlers").Key("NeedReplyMessage").SetValue(config.NeedReplyMessage)
	cfg.Section("setCrawlers").Key("CheckReplyTimeInterval").SetValue(fmt.Sprintf("%d", config.CheckReplyTimeInterval))
	for key, value := range config.ReplyMessageTexts {

		newKey := fmt.Sprintf("ReplyTxt%d", key+1)
		cfg.Section("MessageTxt").Key(newKey).SetValue(value)
	}

	// 保存配置到文件
	err := cfg.SaveTo(ConfigFileName)
	if err != nil {
		return fmt.Errorf("unable to write config file: %v", err)
	}

	return nil
}

// createDefaultConfig 创建默认配置文件
func CreateDefaultConfig() error {
	// 创建配置文件并写入默认内容
	file, err := os.Create(ConfigFileName)
	if err != nil {
		return err
	}
	defer file.Close()

	// 写入默认配置内容
	_, err = file.WriteString(defaultConfig)
	if err != nil {
		return err
	}

	return nil
}
