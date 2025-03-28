package transaction

import (
	"InsAutoMessage/config"
	DB "InsAutoMessage/database"
	"InsAutoMessage/utils"
	"context"
	"errors"
	"fmt"
	"github.com/chromedp/chromedp"
	"log"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// 获取动态内容数量
func GetPostStatistics(ctx context.Context) (int, error) {

	hasCount := false
	chromedp.Run(ctx,

		chromedp.WaitReady(`[aria-label="首页"]`, chromedp.ByQuery),

		chromedp.Evaluate(`document.evaluate("//span[contains(., '篇帖子')]/span", document, null, XPathResult.FIRST_ORDERED_NODE_TYPE, null).singleNodeValue!== null`, &hasCount),
	)

	if hasCount == false {
		return 0, fmt.Errorf("获取动态内容数量失败")
	}

	var postCountText string
	if err := chromedp.Run(ctx,
		chromedp.WaitVisible(`//span[contains(., '篇帖子')]/span`, chromedp.BySearch),
		chromedp.Text(`//span[contains(., '篇帖子')]/span`, &postCountText, chromedp.BySearch),
		chromedp.Sleep(500),
	); err != nil {
		log.Printf("浏览器操作失败: %v", err)
	}

	if postCountText == "" {
		return 0, fmt.Errorf("获取动态内容数量失败")
	}

	countInt := 0
	if strings.Contains(postCountText, "万") {
		postCountText = strings.ReplaceAll(postCountText, "万", "")
		count, err := strconv.ParseFloat(postCountText, 64)
		if err == nil {
			countInt = int(count * 10000) // 1.5万 → 15000
		}
	} else {
		count, err := strconv.ParseFloat(postCountText, 64)
		if err != nil {
			return 0, err
		}
		countInt = int(count)
	}
	return countInt, nil

}

// 获取停用信息并记录日志
func getAccountDisableInfo(ctx context.Context, accountName string) (string, error) {
	log.Printf("[账号异常检测] 开始检查账号 %s 的停用信息", accountName)

	var disableInfo string
	err := chromedp.Run(ctx,
		chromedp.WaitReady(`span[aria-label*="停用日期："]`, chromedp.ByQuery),
		chromedp.Text(`span[aria-label*="停用日期："]`, &disableInfo, chromedp.ByQuery),
	)
	if err == nil && disableInfo != "" {
		log.Printf("[账号异常检测] 账号 %s 停用详情: %s", accountName, disableInfo)
		return disableInfo, nil
	} else {
		return "", fmt.Errorf("获取停用信息失败: %v", err)
	}
}

// 处理异常账号
func HandleAbnormalAccount(ctx context.Context, accountName string) error {

	// 处理文件重命名，仅在获取到停用信息时才进行操作
	disableInfo, err := getAccountDisableInfo(ctx, accountName)

	if disableInfo == "" {
		return err
	}
	//fmt.Println("停用信息1:", disableInfo)

	if strings.Contains(accountName, ".json") {
		if renameErr := utils.SafeRename(accountName, accountName+".账号被封禁"); renameErr != nil {
			log.Printf("重命名失败: %v", renameErr)
			return fmt.Errorf("重命名失败: %v", renameErr)
		}
	}

	updateErr := UpdateAccountState(accountName, 1, 1, disableInfo)
	if updateErr != nil {
		return fmt.Errorf("账号状态更新失败: %v", updateErr)
	}

	return nil
}

// 标记账号
func UpdateAccountState(accountName string, state, islogin int, result string) error {

	// 判断账号是登录文件 还是数据库中获取的账号
	if strings.Contains(accountName, ".json") {

		accountName = strings.TrimSuffix(filepath.Base(accountName), filepath.Ext(accountName))
		if accountName == "" {
			return errors.New("无法从 accountFile 中提取有效的文件名")
		}
		// 处理文件重命名，仅在获取到停用信息时才进行操作

	}

	account, err := DB.GetAccountByAccount(config.GormDb, accountName)
	if err != nil {
		return fmt.Errorf("查询失败:%v", err)
	}
	account.State = int64(state)
	account.Result = result
	account.Islogin = int64(islogin)
	if err := DB.UpdateAccount(config.GormDb, account); err != nil {

		return fmt.Errorf("更新失败:%v", err)
	}

	return nil
}

// 处理消息限制发送
func HandleCanNotSend(ctx context.Context, accountFile string) error {

	if renameErr := utils.SafeRename(accountFile, accountFile+".限制消息发送"); renameErr != nil {
		log.Printf("重命名失败: %v", renameErr)
		return fmt.Errorf("重命名失败: %v", renameErr)
	}

	result := ""
	err := chromedp.Run(ctx,
		chromedp.Sleep(500),
		chromedp.Text(`span:contains("You can't send messages for")`, &result, chromedp.BySearch),
	)
	if err != nil {
		return fmt.Errorf("获取消息限制内容失败: %v", err)
	}

	if result == "" {
		result = "限制消息发送"
	}

	err = UpdateAccountState(accountFile, 1, 1, result)
	if err != nil {
		return fmt.Errorf("账号状态更新失败: %v", err)
	}

	return nil
}

// 处理弹窗
func HandleClickDialog(ctx context.Context) error {
	if clickErr := chromedp.Run(ctx,

		chromedp.WaitVisible(`//*[text()='以后再说']`, chromedp.BySearch),
		chromedp.Click(`//*[text()='以后再说']`, chromedp.BySearch),
	); clickErr != nil {
		return clickErr
	}
	return nil
}

// 处理重试弹窗
func HandleRetryAlert(accountFile string) error {

	// 处理异常账号
	if renameErr := utils.SafeRename(accountFile, accountFile+".账号异常"); renameErr != nil {
		//log.Printf("重命名失败: %v", renameErr)
		return fmt.Errorf("重命名失败: %v", renameErr)
	}

	/*
		accountName := strings.TrimSuffix(filepath.Base(accountFile), filepath.Ext(accountFile))
		if accountName == "" {
			return errors.New("无法从 accountFile 中提取有效的文件名")
		}

	*/

	err := UpdateAccountState(accountFile, 1, 1, "账号异常，发消息有弹窗")
	if err != nil {
		return fmt.Errorf("账号状态更新失败: %v", err)
	}

	return nil
	//return fmt.Errorf("账号异常，发消息有弹窗，停止后续操作")
}

func HandleDisconnect(accountFile string) error {

	err := utils.SafeRename(accountFile, accountFile+".账号登录已过期")
	if err != nil {
		return fmt.Errorf("文件重命名失败: %v", err)
	}

	err = UpdateAccountState(accountFile, 0, 0, "账号登录过期")
	if err != nil {
		return fmt.Errorf("账号状态更新失败: %v", err)
	}

	return nil
}

func CheckClickMessage(ctx context.Context) error {
	var result struct {
		HasFirst  bool `json:"hasFirst"`
		HasSecond bool `json:"hasSecond"`
	}

	err := chromedp.Run(ctx,
		chromedp.EvaluateAsDevTools(`
            (() => {

				const hasFirst =document.querySelector('[aria-label="Messenger"]') !== null;
				const hasSecond =document.querySelector('[aria-label="Direct"]') !== null;
                return {
                    hasFirst: !!hasFirst,
					hasSecond : !!hasSecond,
                };
            })()
        `, &result),
	)

	if err != nil {
		//return fmt.Errorf("JS注入,状态检测失败: %w", err)
	}

	switch {
	case result.HasFirst:

		err := chromedp.Run(ctx,
			chromedp.Click(`[aria-label="Messenger"]`, chromedp.ByQuery),
		)
		if err != nil {
			return err
		}

	case result.HasSecond:
		err := chromedp.Run(ctx,
			chromedp.Click(`[aria-label="Direct"]`, chromedp.ByQuery),
		)
		if err != nil {
			return err
		}

	}
	return nil
}

// CheckLoggingState 检查账号登录时的账号状态
func CheckLoggingState(ctx context.Context, accountName string) error {

	// 设置总超时
	//ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	//defer cancel()

	var result struct {
		HasDialog    bool `json:"hasDialog"`
		HasEmail     bool `json:"hasEmail"`
		HasEmailCode bool `json:"hasEmailCode"`
		HasBan       bool `json:"hasBan"`
	}

	err := chromedp.Run(ctx,
		chromedp.EvaluateAsDevTools(`
            (() => {
                // 同时检测4种场景
				const hasDialog =document.evaluate("//*[contains(text(), '以后再说')]", document, null, XPathResult.FIRST_ORDERED_NODE_TYPE, null).singleNodeValue !== null ;
				const hasEmail =document.querySelector('[aria-label="我们会将验证码发送到这个邮箱。"]');
				const hasEmailCode = document.evaluate("//*[contains(text(), '请帮助我们验证')]", document, null, XPathResult.FIRST_ORDERED_NODE_TYPE, null).singleNodeValue!== null;
				const hasBan =document.querySelector('[aria-label="申诉"]') !== null;

                return {
                    hasBan: !!hasBan,
					hasDialog : !!hasDialog,
					HasEmail :!!hasEmail,
					HasEmailCode : !!hasEmailCode,
                };
            })()
        `, &result),
	)

	if err != nil {
		//return fmt.Errorf("JS注入,状态检测失败: %w", err)
	}

	switch {
	case result.HasDialog:
		err := HandleClickDialog(ctx)
		if err != nil {
			return err
		}
		return CheckLoggingState(ctx, accountName)

	case result.HasEmail:
		log.Printf("账号: %s 出现邮箱验证", accountName)
		err := UpdateAccountState(accountName, 1, 1, "账号出现邮箱验证")
		if err != nil {
			return err
		}
		return fmt.Errorf("账号: %s 出现邮箱验证", accountName)
	case result.HasEmailCode:
		log.Printf("账号: %s 出现邮箱验证", accountName)
		err := UpdateAccountState(accountName, 1, 1, "账号出现邮箱验证")
		if err != nil {
			return err
		}
		return fmt.Errorf("账号: %s 出现邮箱验证", accountName)

	case result.HasBan:
		log.Printf("账号: %s 已被停用", accountName)
		err := HandleAbnormalAccount(ctx, accountName)
		if err != nil {
			return err
		}
		log.Println("test")
		return fmt.Errorf("账号: %s 已被停用", accountName)
	default:
		return nil
	}

}

// CheckMessageSendStatus 检查已经登录的账号状态
func CheckAlreadyLoggedStatus(ctx context.Context, accountFile string) error {
	// 设置总超时
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	accountName := strings.TrimSuffix(filepath.Base(accountFile), filepath.Ext(accountFile))
	if strings.Contains(accountFile, ".json") {

		accountName = strings.TrimSuffix(filepath.Base(accountName), filepath.Ext(accountName))
		if accountName == "" {
			return errors.New("无法从 accountFile 中提取有效的文件名")
		}
		// 处理文件重命名，仅在获取到停用信息时才进行操作
	}

	// 定义一个结构体变量用于接收 JavaScript 返回结果
	var result struct {
		HasCantSend   bool `json:"hasCantSend"`
		HasError      bool `json:"hasError"`
		HasBan        bool `json:"hasBan"`
		HasDialog     bool `json:"hasDialog"`
		HasEmail      bool `json:"hasEmail"`
		HasEmailCode  bool `json:"hasEmailCode"`
		IsDisconnect  bool `json:"isDisconnect"`
		IsSelf        bool `json:"isSelf"`
		HasSendFailed bool `json:"hasSendFailed"`
		HasAuto       bool `json:"hasAuto"`
		IsCantFind    bool `json:"isCantFind"`
	}

	err := chromedp.Run(ctx,
		chromedp.EvaluateAsDevTools(`
            (() => {
                // 同时检测10种场景
                const hasCantSend = Array.from(document.querySelectorAll('span')).find(span => 
                    span.textContent.includes("你在某个聊天中发送的某些内容违反了我们的社群守则") ||
                    span.textContent.includes("You can't send messages for")
                );

				const hasDialog =document.evaluate("//*[contains(text(), '以后再说')]", document, null, XPathResult.FIRST_ORDERED_NODE_TYPE, null).singleNodeValue !== null ;
				const isDisconnect = document.evaluate("//span[contains(text(), '忘记密码了？')]", document, null, XPathResult.FIRST_ORDERED_NODE_TYPE, null).singleNodeValue!== null;
                const errorDiv = document.querySelector('div[role="alert"][aria-live="polite"]');
				const hasEmail =document.querySelector('[aria-label="我们会将验证码发送到这个邮箱。"]');
				const hasEmailCode = document.evaluate("//span[contains(text(), '帮助我们验证')]", document, null, XPathResult.FIRST_ORDERED_NODE_TYPE, null).singleNodeValue!== null;
			
				const isCantFind = document.evaluate("//span[contains(text(), '找不到帐户。')]", document, null, XPathResult.FIRST_ORDERED_NODE_TYPE, null).singleNodeValue!== null;
				
				const hasAuto = document.evaluate("//*[contains(text(), '自动化行为')]", document, null, XPathResult.FIRST_ORDERED_NODE_TYPE, null).singleNodeValue!== null;

				const hasSendFailed =document.querySelector('[aria-label="Failed to send"]') !== null;
				const isSelf = document.evaluate("//*[contains(text(), '检测到了可疑登录')]", document, null, XPathResult.FIRST_ORDERED_NODE_TYPE, null).singleNodeValue!== null;

				const hasBan =document.querySelector('[aria-label="申诉"]') !== null;

                return {
                    hasCantSend: !!hasCantSend,
					hasDialog : !!hasDialog,
					jasEmail :!!hasEmail,
					IsDisconnect : !!isDisconnect,
					hasEmailCode : !!hasEmailCode,
					hasBan : !!hasBan,
					isSelf :!!isSelf,
					hasSendFailed :!!hasSendFailed,
					IsCantFind :!!isCantFind,
					hasAuto :!!hasAuto,
                    hasError: errorDiv ? errorDiv.textContent.includes('啊哦，出错了，请重试！') : false
                };
            })()
        `, &result),
	)

	if err != nil {
		//return fmt.Errorf("JS注入，状态检测失败: %w", err)
	}

	// 互斥处理逻辑，根据返回结果执行不同的处理逻辑
	switch {

	case result.HasDialog:
		err := HandleClickDialog(ctx)
		if err != nil {
			return err
		}
		//return nil
		return CheckAlreadyLoggedStatus(ctx, accountFile)
	case result.IsSelf:
		log.Printf("检测到账号: %s 出现可疑登录", accountName)
		chromedp.Run(ctx,
			chromedp.Click(`//*[contains(text(), '是我本人')]`, chromedp.BySearch),
		)
		return CheckAlreadyLoggedStatus(ctx, accountFile)

	case result.HasEmail:
		log.Printf("账号: %s 出现邮箱验证", accountName)

		err = utils.SafeRename(accountFile, accountFile+".出现邮箱验证")
		if err != nil {
			return err
		}

		err = UpdateAccountState(accountName, 1, 1, "账号出现邮箱验证")
		if err != nil {
			return err
		}

		return fmt.Errorf("有邮箱验证")

	case result.HasEmailCode:
		log.Printf("账号: %s 登录已过期", accountName)

		err := utils.SafeRename(accountFile, accountFile+".出现邮箱验证")
		if err != nil {
			return err
		}

		err = UpdateAccountState(accountName, 1, 1, "账号出现邮箱验证")
		if err != nil {
			return err
		}

		return fmt.Errorf("账号登录已过期")
	case result.IsDisconnect:
		log.Printf("账号: %s 登录过期", accountName)

		err := HandleDisconnect(accountFile)
		if err != nil {
			return err
		}

		return fmt.Errorf("账号登录已过期")
	case result.HasAuto:
		log.Printf("账号: %s 出现自动化行为", accountName)

		chromedp.Run(ctx,
			chromedp.Click(`//*[contains(text(), '关闭')]`, chromedp.BySearch),
		)
	case result.HasCantSend:
		log.Printf("账号: %s 已被限制发送消息", accountName)
		// 此处调用处理发送限制的方法
		err := HandleCanNotSend(ctx, accountFile)
		if err != nil {
			return err
		}
		return fmt.Errorf("账号: %s 已被限制消息发送", accountName)

	case result.HasSendFailed:
		log.Printf("账号: %s 已被限制发送消息", accountName)
		// 此处调用处理发送限制的方法
		err := HandleCanNotSend(ctx, accountFile)
		if err != nil {
			return err
		}
		return fmt.Errorf("账号: %s 已被限制消息发送", accountName)

	case result.HasError:
		log.Printf("检测到账号: %s 发送消息有弹窗", accountName)
		// 此处调用处理错误弹窗的方法
		err := HandleRetryAlert(accountFile)
		if err != nil {
			return err
		}
		return fmt.Errorf("账号: %s 发送消息有弹窗", accountName)

	case result.HasBan:
		log.Printf("检测到账号: %s 被封禁", accountName)

		err := HandleAbnormalAccount(ctx, accountFile)
		if err != nil {
			return err
		}

		return fmt.Errorf("账号: %s 已被停用", accountName)
	}

	return nil
}

func CheckAccountStatus(accountFile string) error {

	opts := utils.SetChromeOptions()
	allocCtx, cancelAlloc := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancelAlloc()

	ctx, cancelCtx := chromedp.NewContext(allocCtx)
	defer cancelCtx()

	if err := utils.LoadCookies(ctx, accountFile); err != nil {
		return fmt.Errorf("加载Cookies失败: %w", err)
	}

	if err := chromedp.Run(ctx,
		chromedp.Navigate("https://www.instagram.com/"),

		chromedp.Sleep(3*time.Second),
	); err != nil {
		return fmt.Errorf("跳转页面失败: %w", err)
	}

	body := ""
	checkUrl := ""
	err := chromedp.Run(ctx,

		chromedp.WaitReady(`img`, chromedp.ByQuery),
		chromedp.Text(`body`, &body, chromedp.ByQuery),

		chromedp.Location(&checkUrl),
	)
	if err != nil {
		return err
	}
	isloginInput := false
	//aria-label="手机号、帐号或邮箱"
	err = chromedp.Run(ctx,

		chromedp.Sleep(3*time.Second),
		chromedp.Evaluate(`document.querySelector('[aria-label="手机号、帐号或邮箱"]') !== null`, &isloginInput),
	)
	if err != nil {
		return err
	}

	if strings.Contains(checkUrl, "login") || isloginInput {

		log.Printf("账号 %s 登录已过期\n", accountFile)

		err := utils.SafeRename(accountFile, accountFile+".账号登录已过期")
		if err != nil {
			return err
		}
		chromedp.Cancel(ctx)

		return fmt.Errorf("账号登录已过期")
	}

	//fmt.Println(body)
	if strings.Contains(body, "你的账户或账户动态违反了我们社群守则") {
		log.Printf("账号%s 已被暂停使用\n", accountFile)

		err := HandleAbnormalAccount(ctx, accountFile)
		if err != nil {
			//isSuccess = false
			chromedp.Cancel(ctx)
			//return nil, err
			return fmt.Errorf("账号已被暂停使用")
		}

	}

	log.Printf("%s 账号正常\n", accountFile)

	chromedp.Cancel(ctx)
	return nil
}
