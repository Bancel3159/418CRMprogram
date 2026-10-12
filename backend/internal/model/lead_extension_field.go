package model

import (
	"time"

	"gorm.io/gorm"
)

// FieldType 扩展字段类型
type FieldType string

const (
	FieldTypeText     FieldType = "text"     // 单行文本
	FieldTypeTextarea FieldType = "textarea" // 多行文本
	FieldTypeNumber   FieldType = "number"   // 数字
	FieldTypeSelect   FieldType = "select"   // 下拉(从 options 列表)
	FieldTypeDate     FieldType = "date"     // 日期
)

// IsValid 校验
func (t FieldType) IsValid() bool {
	switch t {
	case FieldTypeText, FieldTypeTextarea, FieldTypeNumber, FieldTypeSelect, FieldTypeDate:
		return true
	}
	return false
}

// LeadExtensionField 扩展字段定义(admin 配置,所有用户提交线索时按定义渲染)
//
// 设计目的:
//   - 默认线索表只有 13 个核心字段
//   - 管理员可在系统管理 → 扩展字段管理 增删改字段
//   - 字段存到 leads.extension jsonb 中(key = field.code)
//   - 前端按 enabled 字段动态渲染表单项
//
// DictCategory 联动(type=select 时):
//   - 设置后,选项从字典(dict 表)实时拉取;options 字段被忽略
//   - 适用于"渠道/国家/社群/区服"等系统级下拉
//   - 不设置则用 options(JSON 数组)作为静态选项
type LeadExtensionField struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Code      string    `gorm:"size:50;uniqueIndex;not null" json:"code"`   // 字段 key(英文/拼音,作为 JSON key)
	Label     string    `gorm:"size:100;not null" json:"label"`            // 字段显示名(中文)
	Type      FieldType `gorm:"size:20;not null" json:"type"`               // 字段类型
	Required  bool      `gorm:"not null;default:false" json:"required"`     // 是否必填
	Enabled   bool      `gorm:"not null;default:true;index" json:"enabled"` // 是否启用
	Sort      int       `gorm:"not null;default:0" json:"sort"`            // 排序
	Options   JSONB     `gorm:"type:jsonb" json:"options"`                  // select 时的候选项 [{"label":"微信","value":"wechat"}]
	DictCategory string `gorm:"size:50" json:"dict_category"`               // 引用字典分类(type=select 时); 设置后从字典表实时拉选项
	Placeholder string   `gorm:"size:200" json:"placeholder"`               // 输入提示
	Remark    string    `gorm:"type:text" json:"remark"`                     // 管理员备注

	CreatedBy *uint      `gorm:"index" json:"created_by"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName 表名
func (LeadExtensionField) TableName() string {
	return "lead_extension_fields"
}