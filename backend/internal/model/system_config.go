package model

import "time"

// ============ P11 系统配置(key-value,放全局阈值 / cron 表达式等) ============

// SystemConfig 系统配置(全局生效)
//
// 用途:
//   - follow.new_register_days   新注册跟进 X 天(默认 3)
//   - follow.paid_offline_days   付费玩家离线多少天升级提醒(默认 2)
//   - follow.paid_offline_max_days 付费玩家离线多少天自动流失(默认 5)
//   - follow.daily_cron          每日提醒扫描 cron 表达式(默认 "0 0 8 * * *" — UTC 8:00)
//   - 后续可扩展:游戏 / 部门级阈值
//
// 设计要点:
//   - 单表 k-v(string),不分类别(简化)
//   - value 存 JSON-encoded 字符串或纯文本(如数字/字符串)
//   - admin 可读写,普通用户只读
type SystemConfig struct {
	Key         string    `gorm:"primaryKey;size:100" json:"key"`
	Value       string    `gorm:"type:text;not null" json:"value"`
	Description string    `gorm:"size:255" json:"description"`
	UpdatedBy   *uint     `gorm:"index" json:"updated_by"`
	UpdatedAt   time.Time `json:"updated_at"`
	CreatedAt   time.Time `json:"created_at"`
}

// TableName 表名
func (SystemConfig) TableName() string {
	return "system_configs"
}

// FollowConfigKey 系列 follow 配置 key 常量(避免字符串硬编码散落)
const (
	FollowConfigNewRegisterDays    = "follow.new_register_days"
	FollowConfigPaidOfflineDays    = "follow.paid_offline_days"
	FollowConfigPaidOfflineMaxDays = "follow.paid_offline_max_days"
	FollowConfigDailyCron          = "follow.daily_cron"
	// P12 实时通知
	RealtimeHighValueAmount = "realtime.high_value_amount" // 高额订单阈值(元,默认 100)
	RealtimeEnabled         = "realtime.enabled"           // 全局实时通知开关(默认 true;"false" 时订单入库但不推送)
)
