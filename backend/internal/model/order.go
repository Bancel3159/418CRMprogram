package model

import (
	"time"

	"gorm.io/gorm"
)

// OrderStatus 订单状态(归一化枚举,上游原始状态写入 raw_payload)
type OrderStatus string

const (
	OrderStatusPaid      OrderStatus = "paid"      // 已支付(有效业绩)
	OrderStatusRefunded  OrderStatus = "refunded"  // 已退款(扣回业绩)
	OrderStatusPending   OrderStatus = "pending"   // 待支付
	OrderStatusCancelled OrderStatus = "cancelled" // 已取消
)

// IsValid 校验
func (s OrderStatus) IsValid() bool {
	switch s {
	case OrderStatusPaid, OrderStatusRefunded, OrderStatusPending, OrderStatusCancelled:
		return true
	}
	return false
}

// OrderSource 订单来源
type OrderSource string

const (
	OrderSourceManual   OrderSource = "manual"   // 手动上传
	OrderSourceAPI      OrderSource = "api"      // API 定时拉取
	OrderSourceWebhook  OrderSource = "webhook"  // Webhook 推送
)

// AttributionStatus 归属状态
type AttributionStatus string

const (
	AttributionAuto    AttributionStatus = "auto_attributed"     // 自动归属(唯一命中 lead)
	AttributionManual  AttributionStatus = "manual_attributed"   // 人工绑定
	AttributionPending AttributionStatus = "pending_attribution" // 待人工处理(0/多命中)
)

// IsValid 校验
func (s AttributionStatus) IsValid() bool {
	switch s {
	case AttributionAuto, AttributionManual, AttributionPending:
		return true
	}
	return false
}

// Order 订单(固定核心字段 + raw JSONB)
//
// 归因规则(player_id + game_id):
//   - 唯一命中 lead → auto_attributed (lead_id 自动填充)
//   - 0 命中 / 多命中 → pending_attribution (lead_id = NULL,等待人工绑定)
//
// 幂等性:order_no + game_id 联合唯一,重复入库返回已存在记录
type Order struct {
	ID uint `gorm:"primaryKey" json:"id"`

	// ============ 核心字段 ============
	LeadID    *uint      `gorm:"index" json:"lead_id"`                           // 归属 lead (NULL = 待人工)
	GameID    uint       `gorm:"not null;index" json:"game_id"`                  // 游戏
	PlayerID  string     `gorm:"size:100;not null;index" json:"player_id"`       // 玩家 ID(归因主键,匹配 lead.player_id)
	OrderNo   string     `gorm:"size:200;not null;index" json:"order_no"`        // 上游订单号
	Amount    float64    `gorm:"type:numeric(18,4);not null" json:"amount"`      // 充值金额
	Currency  string     `gorm:"size:10;not null;default:'CNY'" json:"currency"` // 币种
	Status    OrderStatus `gorm:"size:20;not null;index" json:"status"`         // 归一化状态
	OrderTime time.Time  `gorm:"not null;index" json:"order_time"`              // 下单时间(可作时间筛选)
	PaidAt    *time.Time `gorm:"index" json:"paid_at"`                          // 支付完成时间(为 NULL = 未支付)

	// ============ 来源 ============
	Source    OrderSource `gorm:"size:20;not null;index" json:"source"` // manual/api/webhook
	PartnerID *uint       `gorm:"index" json:"partner_id"`              // 关联 partner(id)

	// ============ 归属 ============
	AttributionStatus AttributionStatus `gorm:"size:30;not null;default:'pending_attribution';index" json:"attribution_status"`
	AttributedAt      *time.Time        `json:"attributed_at"`
	AttributedBy      *uint             `gorm:"index" json:"attributed_by"`             // 人工归属操作人
	MatchedLeadCount  int               `gorm:"not null;default:0" json:"matched_lead_count"` // 匹配到的 lead 数(诊断用)

	// ============ 元数据 ============
	Remark       string `gorm:"type:text" json:"remark"`            // 备注
	RawPayload   JSONB  `gorm:"type:jsonb" json:"raw_payload"`      // 原始数据备份(全字段)
	ImportBatch  string `gorm:"size:50;index" json:"import_batch"`  // 上传批次号(便于回滚/追溯)
	ImportedBy   *uint  `gorm:"index" json:"imported_by"`          // 手动上传人 (manual 时记录)
	IsFirstCharge bool  `gorm:"not null;default:false" json:"is_first_charge"` // 是否该玩家该游戏首单(进 A 业绩)

	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// ============ 关联 ============
	Game       *Game    `gorm:"foreignKey:GameID" json:"game,omitempty"`
	Lead       *Lead    `gorm:"foreignKey:LeadID" json:"lead,omitempty"`
	Partner    *Partner `gorm:"foreignKey:PartnerID" json:"partner,omitempty"`
	Importer   *User    `gorm:"foreignKey:ImportedBy" json:"importer,omitempty"`
	Attributor *User    `gorm:"foreignKey:AttributedBy" json:"attributor,omitempty"`
}

// TableName 表名
func (Order) TableName() string {
	return "orders"
}

// IsValidOrderSource 校验来源枚举
func IsValidOrderSource(s string) bool {
	switch OrderSource(s) {
	case OrderSourceManual, OrderSourceAPI, OrderSourceWebhook:
		return true
	}
	return false
}