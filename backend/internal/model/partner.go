package model

import (
	"time"

	"gorm.io/gorm"
)

// PartnerType 合作方类型
type PartnerType string

const (
	PartnerTypeAPI     PartnerType = "api"     // API 定时拉取
	PartnerTypeWebhook PartnerType = "webhook" // Webhook 接收推送
)

// IsValid 校验
func (t PartnerType) IsValid() bool {
	switch t {
	case PartnerTypeAPI, PartnerTypeWebhook:
		return true
	}
	return false
}

// Partner 合作方/支付平台配置
//
// API 类型(type=api)字段:
//   base_url     - API 根地址
//   auth_config  - {type: bearer|basic|apikey, token/username/password/key, header}
//   cron_expr    - cron 表达式(空字符串 = 仅手动触发)
//   field_mappings - 响应字段路径映射 {orders: "data.list", order_no: "id", amount: "money", ...}
//   last_pulled_at - 上次拉取时间(增量用)
//
// Webhook 类型(type=webhook)字段:
//   hmac_secret  - HMAC-SHA256 验签密钥(header X-Signature)
//   webhook_url  - 暴露给上游的 URL 形如 /api/v1/orders/webhook/{code}
type Partner struct {
	ID   uint   `gorm:"primaryKey" json:"id"`
	Code string `gorm:"size:50;uniqueIndex;not null" json:"code"` // 合作方代码(URL/标识用,如 'payplatform-a')
	Name string `gorm:"size:100;not null" json:"name"`
	Type PartnerType `gorm:"size:20;not null;index" json:"type"`

	// 通用
	Enabled bool   `gorm:"not null;default:true;index" json:"enabled"`
	Remark  string `gorm:"type:text" json:"remark"`

	// API 配置 (type=api)
	BaseURL       string     `gorm:"type:text" json:"base_url"`
	AuthConfig    JSONB      `gorm:"type:jsonb" json:"auth_config"`         // 认证配置
	CronExpr      string     `gorm:"size:50" json:"cron_expr"`              // cron 表达式
	FieldMappings JSONB      `gorm:"type:jsonb" json:"field_mappings"`      // 响应字段路径映射
	LastPulledAt  *time.Time `json:"last_pulled_at"`
	LastPullStatus string    `gorm:"size:255" json:"last_pull_status"`      // 上次拉取结果描述
	LastPullCount  int       `gorm:"not null;default:0" json:"last_pull_count"`

	// Webhook 配置 (type=webhook)
	HMACSecret string `gorm:"size:100" json:"hmac_secret,omitempty"`

	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// 计算字段(非持久化,前端展示用)
	WebhookURL string `gorm:"-" json:"webhook_url,omitempty"`
}

// TableName 表名
func (Partner) TableName() string {
	return "partners"
}

// ComputeWebhookURL 拼接 webhook URL(供前端展示)
func (p *Partner) ComputeWebhookURL(baseURL string) string {
	if p.Type != PartnerTypeWebhook {
		return ""
	}
	return baseURL + "/api/v1/orders/webhook/" + p.Code
}