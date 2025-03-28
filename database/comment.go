package DB

import (
	mysql1 "github.com/go-sql-driver/mysql"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"log"
	"os"
	"time"
)

//var Dsn = "caijue:Zhijian123@tcp(rm-bp1z2996838g3yp5jko.mysql.rds.aliyuncs.com:3306)/ins?charset=utf8mb4&parseTime=True"

// 初始化数据库
func InitDB(dsn string) (*gorm.DB, error) {

	newLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags), // io writer
		logger.Config{
			SlowThreshold: time.Second,   // 慢 SQL 阈值
			LogLevel:      logger.Silent, // 日志级别，只记录错误日志
			Colorful:      false,         // 禁用彩色打印
		},
	)

	db, err := gorm.Open(mysql.Open(dsn),
		&gorm.Config{
			Logger: newLogger,
		},
	)

	if err != nil {
		return nil, err
	}

	// 自动迁移（生产环境建议使用迁移工具）
	if err := db.AutoMigrate(&LoginAccount{}); err != nil {
		return nil, err
	}
	return db, nil
}

// IsDuplicateError 判断是否为重复错误
func IsDuplicateError(err error) bool {
	if err == nil {
		return false
	}
	// MySQL 错误码 1062: ER_DUP_ENTRY
	if mysqlErr, ok := err.(*mysql1.MySQLError); ok {
		return mysqlErr.Number == 1062
	}
	return false
}
