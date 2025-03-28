package DB

import (
	"InsAutoMessage/config"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"log"
)

// 状态常量定义
const (
	StateUncollected = 0 // 未采集
	StateCollected   = 1 // 采集完毕
)

// 模型定义
type Location struct {
	LocationID   string `gorm:"primaryKey;column:locationid;uniqueIndex"`
	LocationName string `gorm:"column:LocationName;type:varchar(100);not null"`
	Count        int    `gorm:"column:count;default:null"` // 使用指针处理 NULL
	State        int    `gorm:"column:state;default:0"`
}

// 自定义表名
func (Location) TableName() string {
	//return "india_bloggers"
	return config.AppCfg.LocationTableName
}

// 创建地点（处理唯一约束）
func CreateLocation(db *gorm.DB, loc *Location) error {
	err := db.Create(loc).Error
	if err != nil {
		if IsDuplicateError(err) {
			log.Fatalf("地点ID %d 已存在", loc.LocationID)
		}
		return err
	}
	return nil
}

// 通过LocationID获取
func GetLocationByID(db *gorm.DB, locationId string) (*Location, error) {
	var loc Location
	err := db.First(&loc, locationId).Error
	return &loc, err
}

// 通过名称查询（需添加索引建议）
func GetLocationsByLocationId(db *gorm.DB, locationId string) ([]Location, error) {
	var locations []Location
	err := db.Where("locationid = ?", locationId).Find(&locations).Error
	return locations, err
}

// 更新地点信息
func UpdateLocation(db *gorm.DB, loc *Location) error {
	// 检查ID是否存在
	var count int64
	db.Model(&Location{}).Where("locationid = ?", loc.LocationID).Count(&count)
	if count == 0 {
		return gorm.ErrRecordNotFound
	}

	return db.Save(loc).Error
}

// 更新状态
func UpdateLocationState(db *gorm.DB, locationId string, state int) error {
	return db.Model(&Location{}).
		Where("locationid = ?", locationId).
		Update("state", state).
		Error
}

// 通过ID删除
func DeleteLocationByID(db *gorm.DB, locationId string) error {
	return db.Delete(&Location{}, locationId).Error
}

// 在db包中新增以下方法

// GetAndLockLocation 原子化获取并锁定地点
func GetAndLockLocation(db *gorm.DB) (*Location, error) {
	var location Location
	err := db.Transaction(func(tx *gorm.DB) error {
		// 锁定并获取一个未处理的地点
		if err := tx.Where("state = 0").Clauses(clause.Locking{Strength: "UPDATE"}).
			Order("RAND()").First(&location).Error; err != nil {
			return err
		}

		// 标记为处理中
		return tx.Model(&location).Update("state", 2).Error
	})
	return &location, err
}

// 随机获取指定数量的未采集地点
func GetRandomUncollectedLocations(db *gorm.DB, limit int) ([]Location, error) {
	var locations []Location
	err := db.
		Where("state = ?", StateUncollected). // 状态0
		Order("RAND()").                      // 核心：随机排序
		Limit(limit).                         // 限制数量
		Find(&locations).                     // 获取结果
		Error
	return locations, err
}

func GetRandomUncollectedLocationsOne(db *gorm.DB) (Location, error) {
	var locations Location
	err := db.
		Where("state = ?", StateUncollected). // 状态0
		Order("RAND()").                      // 核心：随机排序
		Find(&locations).                     // 获取结果
		Error
	return locations, err
}
