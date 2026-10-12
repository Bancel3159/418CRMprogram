package model

import (
	"time"

	"gorm.io/gorm"
)

// LeadStatus 线索状态
type LeadStatus string

const (
	LeadStatusPending  LeadStatus = "pending"  // 公开池（待认领）
	LeadStatusClaimed  LeadStatus = "claimed"  // 已认领
	LeadStatusArchived LeadStatus = "archived" // 已流失 / 归档
)

// IsValid 校验状态
func (s LeadStatus) IsValid() bool {
	switch s {
	case LeadStatusPending, LeadStatusClaimed, LeadStatusArchived:
		return true
	}
	return false
}

// IsTarget 是否目标
type IsTarget string

const (
	IsTargetTrue    IsTarget = "target"     // 目标
	IsTargetFalse   IsTarget = "not_target" // 非目标
)

// IsValid 校验 is_target 枚举
func (t IsTarget) IsValid() bool {
	switch t {
	case IsTargetTrue, IsTargetFalse:
		return true
	}
	return false
}

// Lead 线索（拉新记录，13 字段）
type Lead struct {
	ID uint `gorm:"primaryKey" json:"id"`

	// ============ 13 个核心字段 ============
	GameID         uint     `gorm:"not null;index" json:"game_id"`                   // 1. 游戏（FK → game）
	CreatorID      uint     `gorm:"not null;index" json:"creator_id"`                // 2. 拉新员工（自动取登录者，FK → user）
	Channel        string   `gorm:"size:50;not null" json:"channel"`                 // 3. 渠道（来源）
	PlayerUsername string   `gorm:"size:200;not null" json:"player_username"`        // 4. 玩家用户名（渠道联系方式）
	PlayerGameName string   `gorm:"size:100" json:"player_game_name"`                // 5. 玩家游戏名
	PlayerID       string   `gorm:"size:100;not null;index" json:"player_id"`        // 6. 玩家 ID（订单匹配主键）
	Region         string   `gorm:"size:50" json:"region"`                           // 7. 区服
	PlayerCountry  string   `gorm:"size:50" json:"player_country"`                  // 8. 玩家国家
	PlayerAge      int      `json:"player_age"`                                      // 9. 玩家年龄
	IsTarget       IsTarget `gorm:"size:20;not null" json:"is_target"`              // 10. 是否目标（target/not_target）
	Community      string   `gorm:"size:100" json:"community"`                       // 11. 社群
	TokenCode      string   `gorm:"type:text" json:"token_code"`                     // 12. Discord auth token
	Screenshots    Strings  `gorm:"type:jsonb;default:'[]'" json:"screenshots"`      // 13. 上传截图（路径数组）

	// ============ 辅助字段 ============
	Status         LeadStatus `gorm:"size:20;not null;default:'pending';index" json:"status"` // 状态机
	ClaimedBy      *uint      `gorm:"index" json:"claimed_by"`                               // 认领人（B 部员工）
	ClaimedAt      *time.Time `json:"claimed_at"`                                            // 认领时间
	ReleasedBy     *uint      `json:"released_by"`                                           // 释放人（主管/admin）
	ReleasedAt     *time.Time `json:"released_at"`                                           // 释放时间
	Extension      JSONB      `gorm:"type:jsonb" json:"extension"`                           // 扩展字段（后续可加）
	Remark         string     `gorm:"type:text" json:"remark"`                               // 备注

	// ============ 付费标记 ============
	// 自动:订单系统存在 player_id 对应的 paid 订单
	// 手动:admin / AB 主管 / 组长 在 Lead 详情页强制标记(订单系统没记录时兜底)
	// 最终 is_paid = paid_mark || paid_auto
	PaidMark     bool       `gorm:"default:false;index" json:"paid_mark"`         // 人工标记已付费
	PaidMarkedBy *uint      `gorm:"index" json:"paid_marked_by"`                   // 标记人 FK → user
	PaidMarkedAt *time.Time `json:"paid_marked_at"`                                // 标记时间

	// 运行时计算字段(不持久化,gorm:"-")
	PaidAuto bool `gorm:"-" json:"paid_auto"` // 自动检测:order 里有 paid 单
	IsPaid   bool `gorm:"-" json:"is_paid"`   // 最终:paid_mark || paid_auto

	// ============ P12 流失 / 激活(P11 跟进规则引擎用) ============
	IsLost             bool       `gorm:"default:false;index" json:"is_lost"`              // 是否流失
	LostAt             *time.Time `json:"lost_at"`                                            // 流失判定时间
	LostReason         string     `gorm:"size:50" json:"lost_reason"`                          // 流失原因: auto_new_register_timeout / auto_paid_offline / manual
	ReactivatedAt      *time.Time `json:"reactivated_at"`                                     // 再次激活时间(玩家重新上线后)
	LastPlayerActiveAt *time.Time `gorm:"index" json:"last_player_active_at"`                // 玩家最后活跃时间(由订单 webhook / 合作方数据回传更新;Phase 2 用 last_follow_at 兜底)

	// P9:跟进统计(由 leadGet 填充,前端不用单独查)
	FollowCount   int64      `gorm:"-" json:"follow_count"`
	LastFollowAt  *time.Time `gorm:"-" json:"last_follow_at"`

	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// ============ 关联 ============
	Game       *Game      `gorm:"foreignKey:GameID" json:"game,omitempty"`
	Creator    *User      `gorm:"foreignKey:CreatorID" json:"creator,omitempty"`
	ClaimedByUser *User   `gorm:"foreignKey:ClaimedBy" json:"claimed_by_user,omitempty"`
	ReleasedByUser *User  `gorm:"foreignKey:ReleasedBy" json:"released_by_user,omitempty"`
}

// TableName 表名
func (Lead) TableName() string {
	return "leads"
}