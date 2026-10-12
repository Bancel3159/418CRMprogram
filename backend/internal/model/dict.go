package model

import (
	"time"

	"gorm.io/gorm"
)

// DictCategory 字典分类
type DictCategory string

const (
	// 当前只有 channel 一个分类:维护"来源渠道"下拉选项
	DictCategoryChannel DictCategory = "channel"
)

// DictStatus 字典状态
type DictStatus string

const (
	DictStatusEnabled  DictStatus = "enabled"
	DictStatusDisabled DictStatus = "disabled"
)

// Dict 字典
type Dict struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	Category  DictCategory   `gorm:"size:50;not null;index:idx_dict_category_code,unique" json:"category"`
	Code      string         `gorm:"size:50;not null;index:idx_dict_category_code,unique" json:"code"`
	Label     string         `gorm:"size:100;not null" json:"label"`
	Sort      int            `gorm:"default:0" json:"sort"`
	Status    DictStatus     `gorm:"size:20;not null;default:'enabled'" json:"status"`
	Remark    string         `gorm:"type:text" json:"remark"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName 表名
func (Dict) TableName() string {
	return "dicts"
}