package model

import (
	"time"

	"gorm.io/gorm"
)

// ListViewEntityType 列表视图所属实体
type ListViewEntityType string

const (
	ListViewEntityPlayer ListViewEntityType = "player" // 玩家列表
	ListViewEntityOrder  ListViewEntityType = "order"  // 订单列表
)

// IsValid 校验 entity_type
func (e ListViewEntityType) IsValid() bool {
	switch e {
	case ListViewEntityPlayer, ListViewEntityOrder:
		return true
	}
	return false
}

// ListView 列表视图（管理员可配置）
//
// 同一份数据源（玩家 / 订单）可以有多个视图，每个视图独立配置字段顺序、宽度、对齐、formatter。
// MVP 阶段 UI 只编辑每个 entity 的默认视图（is_default=true 且 is_system=true 的那个）。
// 数据模型为多视图预留，后续可扩展 UI 暴露视图切换器。
//
// 设计要点：
//   - fields 是 JSONList（[{code,label,source,width,align,formatter}, ...]）
//   - is_system=true 的视图不能删除（保证每个 entity 至少一个视图可用）
//   - is_default=true 的视图是列表页默认使用的（一个 entity 同时只允许一个默认）
type ListView struct {
	ID         uint                `gorm:"primaryKey" json:"id"`
	EntityType ListViewEntityType  `gorm:"size:30;not null;index" json:"entity_type"`
	Name       string              `gorm:"size:50;not null" json:"name"`
	IsDefault  bool                `gorm:"not null;default:false;index" json:"is_default"`
	IsSystem   bool                `gorm:"not null;default:false" json:"is_system"`
	Fields     JSONList            `gorm:"type:jsonb;not null;default:'[]'" json:"fields"`
	CreatedBy  *uint                `gorm:"index" json:"created_by"`
	CreatedAt  time.Time           `json:"created_at"`
	UpdatedAt  time.Time           `json:"updated_at"`
	DeletedAt  gorm.DeletedAt      `gorm:"index" json:"-"`
}

// TableName 表名
func (ListView) TableName() string {
	return "list_views"
}