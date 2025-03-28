package DB

import (
	"InsAutoMessage/config"
	"bufio"
	"fmt"
	"gorm.io/gorm"
	"log"
	"os"
	"strings"
)

// 模型定义
type LoginAccount struct {
	ID       int64  `gorm:"primaryKey;autoIncrement"`
	Account  string `gorm:"uniqueIndex;size:50;not null"`
	Password string `gorm:"size:50;not null"`
	Hash     string `gorm:"size:50"`
	State    int64  `gorm:"default:0"`
	Result   string `gorm:"size:100"`
	Islogin  int64  `gorm:"unsigned;column:islogin;default:0"` // 显式指定列名
}

/*
// 自定义表名（可选）
func (LoginAccount) TableName() string {
	return "login_accounts"
}

*/

func (LoginAccount) TableName() string {
	//return "india_bloggers"
	return config.AppCfg.LoginAccountTableName
}

// 创建账号（处理唯一约束）
func CreateAccount(db *gorm.DB, account *LoginAccount) error {
	err := db.Create(account).Error
	if err != nil {
		if IsDuplicateError(err) {
			//return fmt.Errorf("账号 %s 已存在", account.Account)
			log.Printf("账号 %s 已存在", account.Account)
		}
		return err
	}
	return nil
}

// 通过ID获取账号
func GetAccountByID(db *gorm.DB, id uint) (*LoginAccount, error) {
	var account LoginAccount
	err := db.First(&account, id).Error
	return &account, err
}

// 通过账号名获取（利用唯一索引）
func GetAccountByAccount(db *gorm.DB, account string) (*LoginAccount, error) {
	var la LoginAccount
	err := db.Where("account = ?", account).First(&la).Error
	return &la, err
}

// 更新账号信息（处理唯一约束）
func UpdateAccount(db *gorm.DB, account *LoginAccount) error {
	// 检查账号是否已被其他用户使用
	var count int64
	db.Model(&LoginAccount{}).
		Where("account = ? AND id != ?", account.Account, account.ID).
		Count(&count)

	if count > 0 {
		return fmt.Errorf("账号 %s 已被占用", account.Account)
	}

	return db.Save(account).Error
}

// 随机获取正常账号
func GetRandomAccount(db *gorm.DB, limit int) ([]LoginAccount, error) {
	var loginAccounts []LoginAccount
	err := db.
		Where("state = ? and islogin = ?", 0, 0). // 状态0
		Order("RAND()").                          // 核心：随机排序
		Limit(limit).                             // 限制数量
		Find(&loginAccounts).                     // 获取结果
		Error
	return loginAccounts, err
}

func GetAvailableAccountCount(db *gorm.DB) (int64, error) {
	var count int64
	err := db.
		Where("state = ? and islogin = ?", 0, 0). // 状态0
		Model(&LoginAccount{}).
		Count(&count). // 获取结果
		Error
	return count, err
}

// 通过ID删除
func DeleteAccountByID(db *gorm.DB, id uint) error {
	return db.Delete(&LoginAccount{}, id).Error
}

// 通过账号删除
func DeleteAccountByAccount(db *gorm.DB, account string) error {
	return db.Where("account = ?", account).Delete(&LoginAccount{}).Error
}

// 文件导入主逻辑
func ImportAccountsFromFile(db *gorm.DB, path string) (int, int, error) {
	// 打开文件
	file, err := os.Open(path)
	if err != nil {
		return 0, 0, fmt.Errorf("文件打开失败: %w", err)
	}
	defer file.Close()

	// 准备数据通道
	accounts := make(chan LoginAccount)
	errors := make(chan error)

	// 启动处理协程
	go processLines(file, accounts, errors)

	// 执行数据库插入
	return saveAccounts(db, accounts, errors)
}

// 处理文本行
func processLines(file *os.File, accounts chan<- LoginAccount, errors chan<- error) {
	defer close(accounts)
	defer close(errors)

	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		//fmt.Println(line)
		if line == "" {
			continue
		}

		// 分割数据
		parts := strings.Split(line, "----")
		if len(parts) != 3 {
			errors <- fmt.Errorf("第%d行格式错误: %s", lineNum, line)
			continue
		}

		accounts <- LoginAccount{
			Account:  parts[0],
			Password: parts[1],
			Hash:     parts[2],
		}
		fmt.Println(parts[0])
	}

	if err := scanner.Err(); err != nil {
		errors <- fmt.Errorf("文件读取错误: %w", err)
	}
}

// 保存到数据库（带批量插入优化）
func saveAccounts(db *gorm.DB, accounts <-chan LoginAccount, errChan <-chan error) (int, int, error) {
	const batchSize = 100 // 每批插入100条
	var buffer []LoginAccount
	success, fail := 0, 0

	// 错误处理协程
	done := make(chan struct{})
	go func() {
		for err := range errChan {
			log.Printf("错误: %v", err)
			fail++
		}
		close(done)
	}()

	// 主处理循环
	for {
		select {
		case acc, ok := <-accounts:
			if !ok {
				// 插入剩余数据
				if len(buffer) > 0 {
					if err := insertBatch(db, buffer); err != nil {
						return success, fail, err
					}
					success += len(buffer)
				}
				<-done
				return success, fail, nil
			}
			buffer = append(buffer, acc)
			if len(buffer) >= batchSize {
				if err := insertBatch(db, buffer); err != nil {
					return success, fail, err
				}
				success += len(buffer)
				buffer = buffer[:0]
			}

		case <-done:
			return success, fail, nil
		}
	}
}

// 批量插入处理
func insertBatch(db *gorm.DB, accounts []LoginAccount) error {
	return db.Transaction(func(tx *gorm.DB) error {
		for _, acc := range accounts {
			result := tx.Create(&acc)
			if result.Error != nil {
				if IsDuplicateError(result.Error) {
					log.Printf("账号 %s 已存在，跳过", acc.Account)
					continue
				}
				return result.Error
			}
			log.Printf("已插入账号 %s ", acc.Account)
		}
		return nil
	})
}
