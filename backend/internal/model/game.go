package model

import (
	"time"

	"gorm.io/gorm"
)

// GameStatus 游戏状态
type GameStatus string

const (
	GameStatusEnabled  GameStatus = "enabled"
	GameStatusDisabled GameStatus = "disabled"
)

// Game 游戏
//
// 每个游戏可有可无一个"数据回传合作方"配置(原 partner 表的内联版):
//   - PartnerType 为空 → 未启用数据回传
//   - PartnerType = "api" → PartnerConfig 存 {base_url, auth_config, cron_expr, field_mappings, last_pulled_at}
//   - PartnerType = "webhook" → PartnerConfig 存 {hmac_secret}; URL = /api/v1/orders/webhook/<code>
type Game struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	Code      string         `gorm:"size:50;uniqueIndex;not null" json:"code"`        // 缩写，如 TWD
	Name      string         `gorm:"size:100;not null" json:"name"`                  // 全称
	Company   string         `gorm:"size:100" json:"company"`                        // 所属游戏公司（可选）
	Status    GameStatus     `gorm:"size:20;not null;default:'enabled'" json:"status"`
	Remark    string         `gorm:"type:text" json:"remark"`

	// ========== 数据回传配置(原 partner 表的内联) ==========
	PartnerType   *PartnerType `gorm:"size:20" json:"partner_type"`        // null/空 = 不启用;"api" | "webhook"
	PartnerConfig JSONB        `gorm:"type:jsonb" json:"partner_config"`   // 根据 type 存不同字段
	PartnerEnabled bool        `gorm:"-" json:"partner_enabled"`           // 运行时计算:partner_type != nil && != ""

	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName 表名
func (Game) TableName() string {
	return "games"
}

// ComputePartnerEnabled 是否启用数据回传
func (g *Game) ComputePartnerEnabled() bool {
	return g.PartnerType != nil && *g.PartnerType != ""
}