package system

import (
	"log"
	"time"

	"gorm.io/gorm/clause"
	"server/config"
	"server/plugin/db"
)

// UserHistory 用户观看历史 (按 user_id + mid 唯一; 同一影片重复观看会 upsert).
// 不嵌入 gorm.Model: 这里走硬删除避免 deleted_at 列让 (user_id, mid) 唯一约束失效.
type UserHistory struct {
	ID           uint      `json:"id" gorm:"primaryKey"`
	UserID       uint      `json:"userId" gorm:"uniqueIndex:idx_user_mid_history"`
	Mid          int64     `json:"mid" gorm:"uniqueIndex:idx_user_mid_history"`
	Cid          int64     `json:"cid"`
	Pid          int64     `json:"pid"`
	Name         string    `json:"name"`
	Picture      string    `json:"picture"`
	PlayFrom     string    `json:"playFrom"`     // 站点 ID
	PlayFromName string    `json:"playFromName"` // 站点名
	Episode      int       `json:"episode"`      // 集索引
	EpisodeName  string    `json:"episodeName"`  // 集名 (例如 "01" / "第 1 集")
	Progress     int64     `json:"progress"`     // 已观看秒数
	Duration     int64     `json:"duration"`     // 总时长 (秒, 可选)
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

// TableName 设置历史记录表名
func (UserHistory) TableName() string {
	return config.UserHistoryTableName
}

// CreateUserHistoryTable 不存在则创建表
func CreateUserHistoryTable() {
	if db.Mdb == nil {
		return
	}
	if !db.Mdb.Migrator().HasTable(&UserHistory{}) {
		if err := db.Mdb.AutoMigrate(&UserHistory{}); err != nil {
			log.Println("Create Table UserHistory Failed: ", err)
		}
	}
}

// UpsertUserHistory upsert 一条观看记录, 同 (user_id, mid) 已存在则覆盖.
// 写入字段集中只放进度相关 + 用于卡片展示的冗余字段.
func UpsertUserHistory(h *UserHistory) error {
	if db.Mdb == nil {
		return ErrDBNotInitialized
	}
	return db.Mdb.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "user_id"}, {Name: "mid"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"cid", "pid", "name", "picture", "play_from", "play_from_name",
			"episode", "episode_name", "progress", "duration", "updated_at",
		}),
	}).Create(h).Error
}

// ListUserHistory 分页拉取用户观看历史, 按更新时间倒序
func ListUserHistory(uid uint, page *Page) []UserHistory {
	var list []UserHistory
	if db.Mdb == nil {
		return list
	}
	q := db.Mdb.Model(&UserHistory{}).Where("user_id = ?", uid)
	GetPage(q, page)
	if err := q.Limit(page.PageSize).Offset((page.Current-1)*page.PageSize).
		Order("updated_at DESC").Find(&list).Error; err != nil {
		log.Println("ListUserHistory err: ", err)
		return nil
	}
	return list
}

// DeleteUserHistoryByID 删除用户某条历史 (用 user_id 兜底防越权)
func DeleteUserHistoryByID(uid, id uint) error {
	if db.Mdb == nil {
		return ErrDBNotInitialized
	}
	return db.Mdb.Where("user_id = ? AND id = ?", uid, id).Delete(&UserHistory{}).Error
}

// DeleteUserHistoryByMid 通过 mid 删 (前端传 mid 更直接)
func DeleteUserHistoryByMid(uid uint, mid int64) error {
	if db.Mdb == nil {
		return ErrDBNotInitialized
	}
	return db.Mdb.Where("user_id = ? AND mid = ?", uid, mid).Delete(&UserHistory{}).Error
}

// ClearUserHistory 清空当前用户的全部历史
func ClearUserHistory(uid uint) error {
	if db.Mdb == nil {
		return ErrDBNotInitialized
	}
	return db.Mdb.Where("user_id = ?", uid).Delete(&UserHistory{}).Error
}
