package transaction

import (
	"InsAutoMessage/config"
	DB "InsAutoMessage/database"
	"fmt"
	"log"
)

func TxtToDb(path string) error {
	// 处理文件导入

	successCount, failCount, err := DB.ImportAccountsFromFile(config.GormDb, path)
	if err != nil {
		log.Printf("导入失败: %v", err)
		return fmt.Errorf("导入失败: %w", err) // 确保返回错误给调用者
	}
	log.Printf("导入完成！成功: %d 条，失败: %d 条", successCount, failCount)
	return nil
}
