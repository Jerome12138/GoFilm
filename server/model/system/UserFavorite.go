package system

import (
	"errors"
	"log"
	"time"

	"gorm.io/gorm/clause"
	"server/config"
	"server/plugin/db"
)

// ErrDBNotInitialized 在 db.Mdb 未初始化时由 model 层操作返回, 单元测试场景常见.
var ErrDBNotInitialized = errors.New("mysql is not initialized")

// UserFavorite 用户收藏 (按 user_id + mid 唯一).
// 同样硬删除避免 deleted_at 列影响 unique 索引.
type UserFavorite struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	UserID    uint      `json:"userId" gorm:"uniqueIndex:idx_user_mid_fav"`
	Mid       int64     `json:"mid" gorm:"uniqueIndex:idx_user_mid_fav"`
	Cid       int64     `json:"cid"`
	Pid       int64     `json:"pid"`
	Name      string    `json:"name"`
	Picture   string    `json:"picture"`
	Remarks   string    `json:"remarks"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// TableName 设置收藏表名
func (UserFavorite) TableName() string {
	return config.UserFavoriteTableName
}

// CreateUserFavoriteTable 不存在则创建表
func CreateUserFavoriteTable() {
	if db.Mdb == nil {
		return
	}
	if !db.Mdb.Migrator().HasTable(&UserFavorite{}) {
		if err := db.Mdb.AutoMigrate(&UserFavorite{}); err != nil {
			log.Println("Create Table UserFavorite Failed: ", err)
		}
	}
}

// AddUserFavorite 添加收藏, 已存在则忽略 (幂等). DB 层 OnConflict DoNothing.
func AddUserFavorite(f *UserFavorite) error {
	if db.Mdb == nil {
		return ErrDBNotInitialized
	}
	return db.Mdb.Clauses(clause.OnConflict{DoNothing: true}).Create(f).Error
}

// RemoveUserFavorite 取消收藏
func RemoveUserFavorite(uid uint, mid int64) error {
	if db.Mdb == nil {
		return ErrDBNotInitialized
	}
	return db.Mdb.Where("user_id = ? AND mid = ?", uid, mid).Delete(&UserFavorite{}).Error
}

// ListUserFavorite 分页拉取用户收藏, 按收藏时间倒序
func ListUserFavorite(uid uint, page *Page) []UserFavorite {
	var list []UserFavorite
	if db.Mdb == nil {
		return list
	}
	q := db.Mdb.Model(&UserFavorite{}).Where("user_id = ?", uid)
	GetPage(q, page)
	if err := q.Limit(page.PageSize).Offset((page.Current-1)*page.PageSize).
		Order("created_at DESC").Find(&list).Error; err != nil {
		log.Println("ListUserFavorite err: ", err)
		return nil
	}
	return list
}

// IsUserFavorite 判断用户是否已收藏当前影片
func IsUserFavorite(uid uint, mid int64) bool {
	if db.Mdb == nil {
		return false
	}
	var count int64
	db.Mdb.Model(&UserFavorite{}).Where("user_id = ? AND mid = ?", uid, mid).Count(&count)
	return count > 0
}
