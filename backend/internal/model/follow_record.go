package model

import (
	"time"

	"gorm.io/gorm"
)

// LeadFollowRecord 跟进记录
//
// 每次 B 部跟进员工(或 admin 手动补录)与玩家发生一次互动,落一条记录。
// 设计要点:
//   - 与 Lead 是 N:1(一条 lead 可有多条跟进)
//   - 软删除(沿用 gorm.DeletedAt,跟其他表一致)
//   - 不强枚举类型(follow_type 自由文本,用户选择)
//   - content 必填,带时间戳 created_at 强制
type LeadFollowRecord struct {
	ID         uint           `gorm:"primaryKey" json:"id"`
	LeadID     uint           `gorm:"not null;index" json:"lead_id"`
	FollowerID *uint          `gorm:"index" json:"follower_id"`                       // 操作人(B 部员工 / admin),允许 null(系统自动标记场景预留)
	FollowType string         `gorm:"size:20;default:'text'" json:"follow_type"`      // 自由文本(text/call/online/...)
	Content    string         `gorm:"type:text;not null" json:"content"`               // 跟进内容
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"-"`

	// 关联(可选)
	Lead     *Lead `gorm:"foreignKey:LeadID" json:"lead,omitempty"`
	Follower *User `gorm:"foreignKey:FollowerID" json:"follower,omitempty"`
}

// TableName 表名
func (LeadFollowRecord) TableName() string {
	return "lead_follow_records"
}