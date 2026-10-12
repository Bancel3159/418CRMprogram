package model

import (
	"time"

	"gorm.io/gorm"
)

// ============ P11 跟进提醒 / 流失规则引擎 ============

// ReminderType 提醒类型
type ReminderType string

const (
	ReminderFollowNewRegister ReminderType = "follow_new_register"  // 新注册跟进(前 X 天每日提醒)
	ReminderFollowPaidDaily   ReminderType = "follow_paid_daily"    // 付费玩家每日跟进
	ReminderFollowPaidOffline ReminderType = "follow_paid_offline"  // 付费玩家离线升级提醒
	ReminderLostCheck         ReminderType = "lost_check"           // 流失检查提醒
)

// ReminderStatus 提醒状态
type ReminderStatus string

const (
	ReminderStatusPending   ReminderStatus = "pending"   // 待处理
	ReminderStatusDone      ReminderStatus = "done"      // 已完成(写了 follow_records)
	ReminderStatusExpired   ReminderStatus = "expired"   // 已过期(隔天扫到的)
	ReminderStatusDismissed ReminderStatus = "dismissed" // 手动忽略
)

// ReminderPriority 优先级(展示用,值越大越靠前)
//
//	0 = 普通(新注册 / 付费每日)
//	1 = 中(离线 2 天)
//	2 = 高(离线 3 天+)
type ReminderPriority int

const (
	ReminderPriorityNormal ReminderPriority = 0
	ReminderPriorityMedium ReminderPriority = 1
	ReminderPriorityHigh   ReminderPriority = 2
)

// LeadReminder 跟进提醒(由每日 cron 生成,follow 端「今日待办」消费)
//
// 设计要点:
//   - 每个 (lead_id, remind_date, remind_type) 唯一(去重 — 多次扫描不重复插入)
//   - remind_date 是日期(00:00:00)而非时间戳 — 跨天用日期边界判断
//   - 优先级会随时间升级(下次扫描时覆盖)
//   - 软删除沿用 gorm.DeletedAt(管理员可清理)
type LeadReminder struct {
	ID         uint           `gorm:"primaryKey" json:"id"`
	LeadID     uint           `gorm:"not null;uniqueIndex:idx_reminder_unique,priority:1" json:"lead_id"`
	RemindDate time.Time      `gorm:"type:date;not null;uniqueIndex:idx_reminder_unique,priority:2" json:"remind_date"` // 日期(时分秒归零)
	RemindType ReminderType   `gorm:"size:50;not null;uniqueIndex:idx_reminder_unique,priority:3" json:"remind_type"`
	Priority   ReminderPriority `gorm:"default:0;index" json:"priority"`
	Status     ReminderStatus `gorm:"size:20;not null;default:'pending';index" json:"status"`
	Note       string         `gorm:"size:200" json:"note"` // 给员工看的备注(如 "已离线 3 天,优先级升级")

	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	CompletedAt *time.Time     `json:"completed_at"` // 跟进完成的时间
	CompletedBy *uint          `gorm:"index" json:"completed_by"` // 完成人

	// 关联
	Lead *Lead `gorm:"foreignKey:LeadID" json:"lead,omitempty"`
}

// TableName 表名
func (LeadReminder) TableName() string {
	return "lead_reminders"
}

// BeforeCreate 创建前把 remind_date 归零到当日 00:00:00(避免时分秒干扰唯一索引)
func (r *LeadReminder) BeforeCreate(tx *gorm.DB) error {
	r.RemindDate = time.Date(r.RemindDate.Year(), r.RemindDate.Month(), r.RemindDate.Day(), 0, 0, 0, 0, r.RemindDate.Location())
	return nil
}
