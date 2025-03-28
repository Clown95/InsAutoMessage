package utils

import (
	"crypto/tls"
	"github.com/emersion/go-message/charset"
	"golang.org/x/text/encoding/simplifiedchinese"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	"github.com/emersion/go-message/mail"
)

// checkStopCommand 连接 IMAP 收件箱扫描邮件，若检测到邮件正文中包含 "停止使用软件" 则返回 true
func checkStopCommand(c *client.Client) (bool, error) {
	_, err := c.Select("INBOX", true)
	if err != nil {
		return false, err
	}
	//log.Printf("当前邮箱 '%s' 总邮件数: %d", mbox.Name, mbox.Messages)

	criteria := imap.NewSearchCriteria()
	criteria.WithoutFlags = []string{"\\Deleted"}

	ids, err := c.Search(criteria)
	if err != nil {
		return false, err
	}
	//log.Printf("搜索到 %d 封邮件", len(ids))
	if len(ids) == 0 {
		return false, nil
	}

	seqset := new(imap.SeqSet)
	seqset.AddNum(ids...)

	section := &imap.BodySectionName{}
	messages := make(chan *imap.Message, 10)
	done := make(chan error, 1)
	go func() {
		done <- c.Fetch(seqset, []imap.FetchItem{section.FetchItem()}, messages)
	}()

	for msg := range messages {
		if msg == nil {
			continue
		}
		r := msg.GetBody(section)
		if r == nil {
			log.Println("获取 Body 部分为空")
			continue
		}

		mr, err := mail.CreateReader(r)
		if err != nil {
			//log.Printf("解析邮件创建 reader 失败: %v", err)
			continue
		}

		// 尝试读取邮件头（调试信息）
		hdr := mr.Header
		//subject, _ := hdr.Subject()
		hdr.Subject()
		//log.Printf("正在解析邮件: %s", subject)

		for {
			part, err := mr.NextPart()
			if err == io.EOF {
				break
			}
			if err != nil {
				//log.Printf("读取邮件部分出错: %v", err)
				// 跳过出错部分，继续处理下一部分
				continue
			}
			switch h := part.Header.(type) {
			case *mail.InlineHeader:
				bodyBytes, err := io.ReadAll(part.Body)
				if err != nil {
					//	log.Printf("读取邮件正文失败: %v", err)
					continue
				}
				bodyStr := string(bodyBytes)
				//log.Printf("邮件正文内容: %s", bodyStr)
				if strings.Contains(bodyStr, "停止使用软件") {
					return true, nil
				}
			default:
				// 如果遇到非 inline 部分，可以尝试打印 header 信息调试
				log.Printf("遇到非 inline 类型的邮件部分，类型: %T", h)
			}
		}
	}

	if err := <-done; err != nil {
		return false, err
	}

	return false, nil
}

func IsAllow() {

	charset.RegisterEncoding("gb18030", simplifiedchinese.GB18030)

	// 连接 QQ 邮箱的 IMAP 服务器（SSL 加密）
	imapServer := "imap.qq.com:993"
	username := "XXXX@qq.com"      // 修改为你的 QQ 邮箱
	password := "bvrizbvhigblbbja" // QQ 邮箱授权码，不是登录密码
	// TLS 配置
	tlsConfig := &tls.Config{
		ServerName: "imap.qq.com",
	}

	// 连接到 IMAP 服务器
	c, err := client.DialTLS(imapServer, tlsConfig)
	if err != nil {
		//log.Fatal("连接 IMAP 服务器失败:", err)
	}
	defer c.Logout()
	//log.Println("连接到 IMAP 服务器成功")

	// 登录
	if err := c.Login(username, password); err != nil {
		//log.Fatal("登录失败:", err)
	}
	//log.Println("登录成功")

	// 每分钟检查一次是否有停止命令邮件
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		stop, err := checkStopCommand(c)
		if err != nil {
			//log.Println("检查邮件命令时出错:", err)
		}
		if stop {
			log.Println("检测到 '停止使用软件' 命令，软件将停止运行")
			os.Exit(1)
		}
		//log.Println("未检测到停止指令，软件继续运行")
		<-ticker.C
	}
}
