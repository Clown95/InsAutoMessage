package DB

import (
	"InsAutoMessage/config"
	"fmt"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
	"log"
	"os"
	"time"
)

// 模型定义
type IndiaBlogger struct {
	ID        uint      `gorm:"primaryKey;autoIncrement;unsigned"`          // 无符号自增主键
	Nickname  string    `gorm:"uniqueIndex:uniq_nickname;size:50;not null"` // 唯一昵称
	Issend    uint      `gorm:"unsigned;default:0"`                         // 发送状态
	Bloghome  string    `gorm:"size:255;not null"`                          // 博主主页
	Isreply   uint      `gorm:"column:isreply;unsigned;default:0"`          // 回复状态
	UpdatedAt time.Time `gorm:"autoUpdateTime"`                             // 自动更新时间
}

// 自定义表名
func (IndiaBlogger) TableName() string {
	//return "india_bloggers"
	return config.AppCfg.IndiaBloggersTableName

}

// 状态常量定义
const (
	SendStatusUnsent   = 0 // 未发送
	SendStatusSent     = 1 // 已发送
	ReplyStatusNoNeed  = 0 // 无需回复
	ReplyStatusPending = 1 // 待回复
)

// 初始化数据库
func InitDB1() (*gorm.DB, error) {
	dsn := "user:pass@tcp(127.0.0.1:3306)/dbname?charset=utf8mb4&collation=utf8mb4_unicode_ci&parseTime=True&loc=Local"

	newLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags), // io writer
		logger.Config{
			SlowThreshold: time.Second,  // 慢 SQL 阈值
			LogLevel:      logger.Error, // 日志级别，只记录错误日志
			Colorful:      false,        // 禁用彩色打印
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

	/*
		// 自动迁移（生产环境建议使用迁移工具）
		if err := db.AutoMigrate(&IndiaBlogger{}); err != nil {
			return nil, err
		}

	*/
	return db, nil
}

// 创建博主（处理唯一昵称）
func CreateBlogger(db *gorm.DB, blogger *IndiaBlogger) error {
	err := db.Create(blogger).Error
	if err != nil {
		if IsDuplicateError(err) {
			//return fmt.Errorf("昵称 %s 已存在", blogger.Nickname)
			log.Printf("昵称 %s 已存在", blogger.Nickname)
		}
		return err
	}
	return nil
}

// 通过ID获取博主
func GetBloggerByID(db *gorm.DB, id uint) (*IndiaBlogger, error) {
	var blogger IndiaBlogger
	err := db.First(&blogger, id).Error
	return &blogger, err
}

// 通过昵称获取博主
func GetBloggerByNickname(db *gorm.DB, nickname string) (*IndiaBlogger, error) {
	var blogger IndiaBlogger
	err := db.Where("nickname = ?", nickname).First(&blogger).Error
	return &blogger, err
}

// 随机获取指定数量的博主
func GetRandomBlogger(db *gorm.DB, limit int) ([]IndiaBlogger, error) {
	var blogger []IndiaBlogger
	err := db.
		Where("issend = ?", 0). // 状态0
		Order("RAND()").        // 核心：随机排序
		Limit(limit).           // 限制数量
		Find(&blogger).         // 获取结果
		Error
	return blogger, err
}
func GetAvailableBloggerCount(db *gorm.DB) (int64, error) {
	var count int64
	err := db.
		Where("issend = ?", 0). // 状态0
		Model(&IndiaBlogger{}).
		Count(&count). // 获取结果
		Error
	return count, err
}

// 更新博主信息（带唯一性检查）
func UpdateBlogger(db *gorm.DB, blogger *IndiaBlogger) error {
	// 检查昵称是否重复
	var count int64
	db.Model(&IndiaBlogger{}).
		Where("nickname = ? AND id != ?", blogger.Nickname, blogger.ID).
		Count(&count)

	if count > 0 {
		return fmt.Errorf("昵称 %s 已被使用", blogger.Nickname)
	}

	return db.Save(blogger).Error
}

// 更新博主信息（带唯一性检查）
func UpdateIsReplyByBloghome(db *gorm.DB, bloghome string, isReply int) error {

	return db.Model(&IndiaBlogger{}).
		Where("bloghome = ?", bloghome).
		Update("isreply", isReply).
		Error
}

func UpdateIssendByBloghome(db *gorm.DB, bloghome string, state int) error {
	return db.Model(&Location{}).
		Where("bloghome = ?", bloghome).
		Update("issend", state).
		Error
}

func UpdateIsReplyByNickeName(db *gorm.DB, nickNamne string, isReply int) error {

	return db.Model(&IndiaBlogger{}).
		Where("nickname = ?", nickNamne).
		Update("isreply", isReply).
		Error
}

// BatchCreateBlogger 实现批量插入，批量大小设置为100，使用 INSERT IGNORE 忽略重复数据错误
func BatchCreateBlogger(db *gorm.DB, bloggers []*IndiaBlogger) error {
	if len(bloggers) == 0 {
		return nil
	}
	// 使用 GORM 的 Clause 实现 MySQL 的 INSERT IGNORE
	return db.Clauses(clause.Insert{Modifier: "IGNORE"}).
		CreateInBatches(bloggers, 100).Error
}

// 批量更新发送状态
func BatchUpdateSendStatus(db *gorm.DB, ids []uint, status uint) error {
	return db.Model(&IndiaBlogger{}).
		Where("id IN ?", ids).
		Update("issend", status).
		Error
}

// 根据状态组合查询（使用复合索引）
func GetBloggersByStatus(db *gorm.DB, issend, isreply uint) ([]IndiaBlogger, error) {
	var bloggers []IndiaBlogger
	err := db.Where("issend = ? AND isreply = ?", issend, isreply).Find(&bloggers).Error
	return bloggers, err
}
