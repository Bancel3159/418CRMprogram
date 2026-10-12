package model

import (
	"time"

	"gorm.io/gorm"
)

// BroadcastTemplate 广播模板（admin 配置，新线索入库时随机抽取渲染）
type BroadcastTemplate struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	Name      string         `gorm:"size:100;not null" json:"name"`        // 模板名（管理用）
	Template  string         `gorm:"type:text;not null" json:"template"`   // 模板内容，用 {字段} 占位
	Enabled   bool           `gorm:"not null;default:true" json:"enabled"` // 是否启用
	Weight    int            `gorm:"not null;default:1" json:"weight"`     // 抽取权重（默认 1）
	Remark    string         `gorm:"type:text" json:"remark"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName 表名
func (BroadcastTemplate) TableName() string {
	return "broadcast_templates"
}

// 模板占位符定义（来自 Lead 表字段）
// {游戏} {游戏代码} {国家} {年龄} {区服} {渠道} {社群} {是否目标} {玩家游戏名} {玩家用户名} {token码}
// 注：token 码是公司自有 Discord 账号，参与模板渲染