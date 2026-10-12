package service

import (
	"errors"
	"strconv"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"crm-backend/internal/model"
	"crm-backend/pkg/logger"
)

// ============ P12 实时通知配置(从 system_configs 读) ============

// RealtimeConfig 实时通知配置
type RealtimeConfig struct {
	HighValueAmount float64 // 高额订单阈值(元);>= 即触发 order.high_value 事件(默认 100)
	Enabled         bool    // 全局开关;false 时订单照常入库但不推 WS(true=默认)
}

// DefaultRealtimeConfig 默认值
func DefaultRealtimeConfig() RealtimeConfig {
	return RealtimeConfig{
		HighValueAmount: 100,
		Enabled:         true,
	}
}

// LoadRealtimeConfig 从 system_configs 读(缺失项用默认值)
//   - realtime.high_value_amount  数字(>0,<=100000)
//   - realtime.enabled            "true" / "false"(其他值视作 true)
func LoadRealtimeConfig(db *gorm.DB) RealtimeConfig {
	cfg := DefaultRealtimeConfig()
	rows := []model.SystemConfig{}
	if err := db.Where("key IN ?", []string{
		model.RealtimeHighValueAmount,
		model.RealtimeEnabled,
	}).Find(&rows).Error; err != nil {
		logger.Warnf("LoadRealtimeConfig 失败,用默认值: %v", err)
		return cfg
	}
	for _, r := range rows {
		switch r.Key {
		case model.RealtimeHighValueAmount:
			if v, err := strconv.ParseFloat(r.Value, 64); err == nil && v > 0 && v <= 100000 {
				cfg.HighValueAmount = v
			}
		case model.RealtimeEnabled:
			cfg.Enabled = r.Value != "false" && r.Value != "0"
		}
	}
	return cfg
}

// EnsureRealtimeDefaults 把实时通知默认值写入 system_configs(启动时调用)
func EnsureRealtimeDefaults(db *gorm.DB) {
	defaults := []model.SystemConfig{
		{Key: model.RealtimeHighValueAmount, Value: "100", Description: "高额订单实时通知阈值(元,>= 即视为高额)"},
		{Key: model.RealtimeEnabled, Value: "true", Description: "全局实时通知开关(false=关,异常时一键停推)"},
	}
	for _, c := range defaults {
		if err := db.Clauses(clause.OnConflict{DoNothing: true}).Create(&c).Error; err != nil {
			logger.Warnf("EnsureRealtimeDefaults %s 失败: %v", c.Key, err)
		}
	}
}

// ============ P12 玩家活跃时间更新(供 webhook handler / 手动 API / 后续合作方推送复用) ============

// UpdateLeadActiveTime 更新玩家最后一次活跃时间
//
//   - 优先按 (player_id, game_id) 匹配 lead;匹配上则更新 lead.LastPlayerActiveAt
//     - 如果 lead.IsLost=true 且新时间 > 当前时间 → 视为「再次激活」,置 IsLost=false + LostReason 清空 + 写 ReactivatedAt
//   - 匹配不上 → 静默 no-op(玩家不在任何 lead 里,后续可由手动录入 lead 后再回填)
//   - eventTime 传空 → 默认 time.Now()
//
// 设计:不写 player_active_events 审计表(简化);如果合作方需要审计 → 后续可加
func UpdateLeadActiveTime(db *gorm.DB, playerID string, gameID uint, eventTime *time.Time) error {
	if playerID == "" {
		return errors.New("player_id 不能为空")
	}
	t := time.Now()
	if eventTime != nil && !eventTime.IsZero() {
		t = *eventTime
	}

	// 找该 player 在该 game 下的 lead(取最近一条非 archived 的)
	var lead model.Lead
	err := db.Where("player_id = ? AND game_id = ? AND status <> ?", playerID, gameID, model.LeadStatusArchived).
		Order("id DESC").
		First(&lead).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil // 玩家不在 lead 里,跳过
		}
		return err
	}

	updates := map[string]interface{}{
		"last_player_active_at": t,
	}
	// 流失状态下被新事件激活
	if lead.IsLost {
		updates["is_lost"] = false
		updates["lost_reason"] = ""
		updates["reactivated_at"] = t
	}

	if err := db.Model(&model.Lead{}).Where("id = ?", lead.ID).Updates(updates).Error; err != nil {
		return err
	}
	return nil
}