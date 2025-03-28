package transaction

import (
	"InsAutoMessage/config"
	DB "InsAutoMessage/database"
	"InsAutoMessage/utils"
	"context"
	"fmt"
	"github.com/chromedp/chromedp"
	"log"
	"sync"
	"time"
)

// 第一种登录方式
func HandleFirstLogin(ctx context.Context, account DB.LoginAccount) error {

	err := chromedp.Run(ctx,
		chromedp.WaitReady(`[aria-label="手机号、帐号或邮箱"]`, chromedp.ByQuery),
		chromedp.SendKeys(`input[name="username"]`, account.Account, chromedp.ByQuery),
		chromedp.SendKeys(`input[name="password"]`, account.Password, chromedp.ByQuery),
		chromedp.Click(`button[type="submit"]`, chromedp.ByQuery),
		chromedp.Sleep(2*time.Second),
		//chromedp.Location(&currentURL),
	)
	if err != nil {
		return fmt.Errorf("登录过程中失败: %w", err)
	}
	//log.Println("当前 URL:", currentURL)

	// 封装检查密码错误的函数，返回 true 表示页面中存在“密码错误”提示
	checkPasswordError := func() (bool, error) {
		var hasPwdErr bool
		err := chromedp.Run(ctx,
			// 等待 2 秒，确保页面渲染完成
			chromedp.Sleep(2*time.Second),
			chromedp.Evaluate(`document.evaluate(
			"//div[contains(text(), '很抱歉，密码有误，请检查密码。')]",
			document,
			null,
			XPathResult.FIRST_ORDERED_NODE_TYPE,
			null
		).singleNodeValue !== null`, &hasPwdErr),
		)
		return hasPwdErr, err
	}

	// 第一次检查是否出现密码错误
	hasPwdErr, err := checkPasswordError()
	if err != nil {
		log.Printf("检查密码错误时出错: %v", err)
	}
	if hasPwdErr {
		log.Println("检测到密码错误提示，进行二次尝试点击提交")
		// 再次点击提交按钮，尝试触发登录逻辑
		err = chromedp.Run(ctx,
			chromedp.WaitReady(`button[type="submit"]`, chromedp.ByQuery),
			chromedp.Click(`button[type="submit"]`, chromedp.ByQuery),
		)
		if err != nil {
			return fmt.Errorf("再次点击登录按钮失败: %w", err)
		}

		// 二次检查密码错误
		hasPwdErrSecond, err := checkPasswordError()
		if err != nil {
			log.Printf("二次检查密码错误时出错: %v", err)
		}
		if hasPwdErrSecond {
			// 登录失败，更新账户状态
			account.Result = "账号密码错误"
			account.State = 1
			account.Islogin = 1
			if err := DB.UpdateAccount(config.GormDb, &account); err != nil {
				return fmt.Errorf("更新数据库失败: %w", err)
			}
			log.Println("账号密码错误，已更新账户状态")

			return fmt.Errorf("账号密码错误")
		} else {
			log.Println("二次尝试后未检测到密码错误提示")
		}
	} else {
		log.Println("未检测到密码错误")
	}

	return nil
}

// 第二种登录方法
func HandleSecondLogin(ctx context.Context, account DB.LoginAccount) error {

	err := chromedp.Run(ctx,
		chromedp.WaitReady(`input[name="email"]`, chromedp.ByQuery),
		chromedp.SendKeys(`input[name="email"]`, account.Account, chromedp.ByQuery),
		chromedp.SendKeys(`input[name="pass"]`, account.Password, chromedp.ByQuery),
		chromedp.Click(`//span[text()='登录']`, chromedp.BySearch),
		chromedp.Sleep(2*time.Second),
		//chromedp.Location(&currentURL),
	)

	//recaptcha-checkbox-borderAnimation
	//role="presentation"

	hasVerify := false
	chromedp.Run(ctx,
		// 等待 2 秒，确保页面渲染完成
		chromedp.Sleep(5*time.Second),
		chromedp.Evaluate(`document.querySelector('div[role="presentation"]') !== null`, &hasVerify),
	)

	if !hasVerify {
		return fmt.Errorf("出现身份验证，请手动验证")
	}
	//你输入的登录信息有误。

	if err != nil {
		return fmt.Errorf("登录过程中失败: %w", err)
	}
	//log.Println("当前 URL:", currentURL)
	//x9f619 x1n2onr6 x1ja2u2z x78zum5 xdt5ytf x193iq5w xeuugli x1r8uery x1iyjqo2 xs83m0k xsyo7zv x16hj40l x10b6aqq x1yrsyyn
	// 封装检查密码错误的函数，返回 true 表示页面中存在“密码错误”提示
	checkPasswordError := func() (bool, error) {
		var hasPwdErr bool
		err := chromedp.Run(ctx,
			// 等待 2 秒，确保页面渲染完成
			chromedp.Sleep(2*time.Second),
			chromedp.Evaluate(`document.evaluate(
			"//div[contains(text(), '你输入的登录信息有误')]",
			document,
			null,
			XPathResult.FIRST_ORDERED_NODE_TYPE,
			null
		).singleNodeValue !== null`, &hasPwdErr),
		)
		return hasPwdErr, err
	}

	// 第一次检查是否出现密码错误
	hasPwdErr, err := checkPasswordError()
	if err != nil {
		log.Printf("检查密码错误时出错: %v", err)
	}
	if hasPwdErr {
		log.Println("检测到密码错误提示，进行二次尝试点击提交")
		// 再次点击提交按钮，尝试触发登录逻辑
		err = chromedp.Run(ctx,
			chromedp.Click(`//span[text()='登录']`, chromedp.BySearch),
			//chromedp.Click(`button[type="submit"]`, chromedp.ByQuery),
		)
		if err != nil {
			return fmt.Errorf("再次点击登录按钮失败: %w", err)
		}

		// 二次检查密码错误
		hasPwdErrSecond, err := checkPasswordError()
		if err != nil {
			log.Printf("二次检查密码错误时出错: %v", err)
		}
		if hasPwdErrSecond {
			// 登录失败，更新账户状态
			account.Result = "账号密码错误"
			account.State = 1
			account.Islogin = 1
			if err := DB.UpdateAccount(config.GormDb, &account); err != nil {
				return fmt.Errorf("更新数据库失败: %w", err)
			}
			log.Println("账号密码错误，已更新账户状态")

			return fmt.Errorf("账号密码错误")
		} else {
			log.Println("二次尝试后未检测到密码错误提示")
		}
	} else {
		log.Println("未检测到密码错误，登录成功")
	}

	return nil
}

// 选择登录方案
func SelectLogin(ctx context.Context, account DB.LoginAccount) error {
	// 设置总超时
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	// 定义一个结构体变量用于接收 JavaScript 返回结果
	var result struct {
		HasFirst  bool `json:"hasFirst"`
		HasSecond bool `json:"hasSecond"`
	}

	err := chromedp.Run(ctx,
		chromedp.EvaluateAsDevTools(`
            (() => {
                // 同时检测2种场景
				const HasFirst =document.querySelector('[aria-label="手机号、帐号或邮箱"]') !== null;
				const HasSecond = document.querySelector('input[name="email"]') !== null;
                return {
                    hasFirst: !!HasFirst,
					hasSecond : !!HasSecond,
                };
            })()
        `, &result),
	)

	if err != nil {
		return fmt.Errorf("状态检测失败: %w", err)
	}

	// 互斥处理逻辑，根据返回结果执行不同的处理逻辑
	switch {
	case result.HasFirst:
		log.Println("检测到第一种登录方法")
		err := HandleFirstLogin(ctx, account)
		if err != nil {
			return err
		}

	case result.HasSecond:
		log.Println("检测到第二种登录方法")

		err := HandleSecondLogin(ctx, account)
		if err != nil {
			return err
		}
	}

	return nil
}

// LoginJob 完成单个账号的登录、2FA处理和Cookies保存
func LoginJob(account DB.LoginAccount) error {
	log.Printf("处理账号: %s", account.Account)

	opts := utils.SetChromeOptions()
	allocCtx, cancelAlloc := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancelAlloc()

	ctx, cancel := chromedp.NewContext(allocCtx, chromedp.WithLogf(log.Printf))
	defer cancel()

	// 设置整体超时，防止任务长时间阻塞
	ctx, cancelTimeout := context.WithTimeout(ctx, 1*time.Minute)
	defer cancelTimeout()

	// 打开Instagram登录页面
	if err := chromedp.Run(ctx,
		chromedp.Navigate("https://www.instagram.com/accounts/login/"),
		chromedp.WaitReady(`input`, chromedp.ByQuery),
	); err != nil {
		return fmt.Errorf("无法跳转到登录页面: %w", err)
	}

	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()
		timeout := time.After(60 * time.Second)
		log.Println("正在检测账号状态")
		for {
			select {
			case <-ticker.C:
				stateErr := CheckLoggingState(ctx, account.Account)
				if stateErr != nil {
					//fmt.Println("stateErr2", stateErr)
					//cancelAlloc()
					cancel() //
					break
				}

			case <-timeout:
				return
			}

		}
	}()

	// 输入账号密码
	if err := SelectLogin(ctx, account); err != nil {
		return err
	}

	log.Println("检查是否需要输入验证码")
	const maxRetry = 3

	for i := 0; i < maxRetry; i++ {

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			// 检查是否需要处理2FA
			var hasVerificationInput bool
			if err := chromedp.Run(ctx,
				chromedp.Sleep(5*time.Second),
				chromedp.Evaluate(`document.querySelector('input[name="verificationCode"]') !== null`, &hasVerificationInput),
			); err != nil {
				log.Printf("检查2FA输入框失败: %v", err)
			}

			if hasVerificationInput {
				log.Printf("第%d次,处理验证码", i+1)
				code, err := utils.GetCode("https://2fa.show/2fa/" + account.Hash)
				if err != nil || code == "" {
					log.Printf("获取验证码失败 (重试 %d/%d): %v", i+1, maxRetry, err)

				}
				if code != "" {
					log.Printf("账号 %s 验证码为: %s", account.Account, code)

					chromedp.Run(ctx,
						chromedp.SendKeys(`input[name="verificationCode"]`, code, chromedp.ByQuery),
						chromedp.Sleep(500),
						chromedp.Click(`//button[contains(text(), '确认')]`, chromedp.BySearch),
						chromedp.Sleep(500),
					)
					break
				}
				time.Sleep(time.Second * time.Duration(i*i)) // 指数退避
			}
		}
	}

	//等待主页出现
	chromedp.Run(ctx,
		chromedp.WaitReady(`div[class="x1n2onr6 x6s0dn4 x78zum5"]`, chromedp.ByQuery),
		chromedp.Sleep(3*time.Second),
	)

	account.Islogin = 1
	account.State = 0

	if err := DB.UpdateAccount(config.GormDb, &account); err != nil {
		log.Printf("更新账号 %s 状态失败: %v", account.Account, err)
		//tx.Rollback()
		return err
	}

	// 保存Cookies
	cookieFile := fmt.Sprintf("cookies/%s.json", account.Account)
	if err := utils.SaveCookies(ctx, cookieFile); err != nil {
		return fmt.Errorf("保存Cookies失败: %w", err)
	}
	log.Printf("账号: %s 登录成功，Cookie已保存", account.Account)

	//return nil
	return nil
}

// LoginWork 根据传入的并发数和账号数量限制分发任务
func LoginWork(num, limit int) {

	accounts, err := DB.GetRandomAccount(config.GormDb, limit)
	if err != nil {
		log.Println("获取账号失败:", err)
		return
	}

	if len(accounts) == 0 {
		log.Println("没有可登录账号")
		return
	}

	jobChan := make(chan DB.LoginAccount, len(accounts))
	var wg sync.WaitGroup

	sem := make(chan struct{}, num) // 并发控制信号量

	// 启动worker池
	for i := 0; i < num; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			log.Printf("Worker %d 启动", workerID)
			for account := range jobChan {

				sem <- struct{}{} // 获取信号量
				go func(account DB.LoginAccount) {
					defer func() { <-sem }() // 释放信号量
					log.Printf("[Worker %d] 开始处理账号 %s", workerID, account.Account)
					start := time.Now()

					//tx := config.GormDb.Begin()
					// 更新账号状态为99 代表正在登录
					account.State = 99
					if err := DB.UpdateAccount(config.GormDb, &account); err != nil {
						log.Printf("[Worker %d] 更新账号 %s 状态失败: %v", workerID, account.Account, err)
						//tx.Rollback()
						return
					}

					if err := LoginJob(account); err != nil {
						log.Printf("[Worker %d] 账号 %s 登录失败: %v", workerID, account.Account, err)
						//tx.Rollback()
						return
					}
					/*
						account.Islogin = 1
						account.State = 0
						if err := DB.UpdateAccount(config.GormDb, &account); err != nil {
							log.Printf("Worker %d 更新账号 %s 状态失败: %v", workerID, account.Account, err)
							//tx.Rollback()
							return
						}

							if err := tx.Commit(); err != nil {
								log.Printf("Worker %d 提交事务失败: %v", workerID, err)
								return
							}

					*/

					log.Printf("[Worker %d] 成功处理账号 %s (耗时: %s)",
						workerID, account.Account, time.Since(start).Round(time.Second))
				}(account)
			}
			log.Printf("Worker %d 退出", workerID)
		}(i + 1)
	}

	// 将所有账号加入任务队列
	for _, account := range accounts {
		jobChan <- account
	}
	close(jobChan)
	wg.Wait()
	log.Printf("全部 %d 个账号处理完成", len(accounts))
}

func AddNewAccount(workerID int) error {
	accounts, err := DB.GetRandomAccount(config.GormDb, 1)
	if err != nil {
		log.Println("获取账号失败:", err)
		return fmt.Errorf("获取账号失败: %w", err)
	}

	if len(accounts) == 0 {
		log.Println("没有可登录账号")
		return fmt.Errorf("没有可登录账号")
	}

	account := accounts[0]
	log.Printf("[Worker %d] 开始处理补号 %s", workerID, account.Account)
	start := time.Now()

	tx := config.GormDb.Begin()
	account.State = 99
	err = DB.UpdateAccount(config.GormDb, &account)
	if err != nil {
		log.Printf("更新账号状态失败: %v", err)
		tx.Rollback()
		return fmt.Errorf("更新账号状态失败: %w", err)
	}

	if err := LoginJob(account); err != nil {
		log.Printf("[Worker %d] 账号 %s 处理失败: %v", workerID, account.Account, err)
		tx.Rollback()

		AddNewAccount(workerID)

		return fmt.Errorf("账号 %s 处理失败: %w", account.Account, err)
	} else {
		account.Islogin = 1
		account.State = 0
		if err := DB.UpdateAccount(config.GormDb, &account); err != nil {
			tx.Rollback()
			log.Printf("[Worker %d] 更新账号 %s 状态失败: %v", workerID, account.Account, err)
			return fmt.Errorf("更新账号 %s 状态失败: %w", account.Account, err)
		} else {
			tx.Commit()
			log.Printf("[Worker %d] 成功补号 %s (耗时: %s)",
				workerID, account.Account, time.Since(start).Round(time.Second))
			return nil
		}
	}
}
