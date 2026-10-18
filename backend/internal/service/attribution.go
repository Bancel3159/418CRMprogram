package service

import (
	"errors"
	"time"

	"gorm.io/gorm"

	"crm-backend/internal/model"
)

// AttributionResult 归属结果
type AttributionResult struct {
	Order            *model.Order
	MatchedLeadCount int
}

// Attributor 订单归属服务
//
// 规则(player_id + game_id 联合匹配):
//   - 唯一命中 (count == 1): auto_attributed, lead_id 填充
//   - 0 命中 / 多命中 (count == 0 || count >= 2): pending_attribution, lead_id = NULL
//   - 同时检查是否该玩家该游戏首单 (IsFirstCharge)
type Attributor struct {
	DB *gorm.DB
}

// NewAttributor 创建归属服务
func NewAttributor(db *gorm.DB) *Attributor {
	return &Attributor{DB: db}
}

// Attribute 对单笔订单执行归属逻辑
func (a *Attributor) Attribute(order *model.Order) error {
	if order.GameID == 0 || order.PlayerID == "" {
		return errors.New("订单缺少 game_id 或 player_id,无法归属")
	}

	// 查询匹配的 lead (player_id + game_id)
	//
	// P14:分两级匹配
	//   1. 优先 pending / claimed(活跃线索)
	//      —— 唯一命中才自动归属;多条命中仍是 pending(历史行为,人工介入)
	//   2. 若一条都没命中,再退到 archived —— 已归档线索的历史订单仍需归到它头上,
	//      否则玩家被归档后他的所有订单永久悬空(收入算不进该线索 / 玩家页显示 0)
	var leads []model.Lead
	if err := a.DB.Where(
		"player_id = ? AND game_id = ? AND status IN ?",
		order.PlayerID, order.GameID,
		[]model.LeadStatus{model.LeadStatusPending, model.LeadStatusClaimed},
	).Find(&leads).Error; err != nil {
		return err
	}

	usedArchivedFallback := false
	if len(leads) == 0 {
		// fallback:已归档线索(取最新一条,历史订单归到最后一次归档的那条)
		var archived []model.Lead
		if err := a.DB.Where(
			"player_id = ? AND game_id = ? AND status = ?",
			order.PlayerID, order.GameID, model.LeadStatusArchived,
		).Order("id DESC").Limit(1).Find(&archived).Error; err != nil {
			return err
		}
		leads = archived
		usedArchivedFallback = len(archived) > 0
	}

	order.MatchedLeadCount = len(leads)

	// 自动归属条件:唯一命中,或走 archived fallback(此时因 Limit(1) 恒为 1 条)
	if len(leads) == 1 || (usedArchivedFallback && len(leads) >= 1) {
		lead := leads[0]
		order.LeadID = &lead.ID
		order.AttributionStatus = model.AttributionAuto
		now := time.Now()
		order.AttributedAt = &now
	} else {
		// 0 命中(玩家不在任何 lead 里)/ 多条活跃线索命中 → 待人工处理
		order.LeadID = nil
		order.AttributionStatus = model.AttributionPending
	}

	// 判断是否首单 (针对已归属的订单,查询该玩家+游戏的已支付订单数)
	if order.LeadID != nil && order.Status == model.OrderStatusPaid {
		var count int64
		// 当前订单未入库,先排除自己(用 order_no 排除)
		query := a.DB.Model(&model.Order{}).Where(
			"player_id = ? AND game_id = ? AND status = ? AND order_no <> ?",
			order.PlayerID, order.GameID, model.OrderStatusPaid, order.OrderNo,
		)
		if err := query.Count(&count).Error; err != nil {
			return err
		}
		order.IsFirstCharge = (count == 0)
	}

	return nil
}

// AttributeManual 人工绑定(管理员手动指定 lead_id)
func (a *Attributor) AttributeManual(orderID uint, leadID uint, userID uint) error {
	// 校验 lead 存在
	var lead model.Lead
	if err := a.DB.First(&lead, leadID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("目标 lead 不存在")
		}
		return err
	}

	// 更新订单
	now := time.Now()
	updates := map[string]interface{}{
		"lead_id":             lead.ID,
		"attribution_status":  model.AttributionManual,
		"attributed_at":       &now,
		"attributed_by":       userID,
		"matched_lead_count":  1,
	}
	return a.DB.Model(&model.Order{}).
		Where("id = ?", orderID).
		Updates(updates).Error
}

// AttributeBatch 批量归属(用于上传后的批量处理)
func (a *Attributor) AttributeBatch(orders []model.Order) error {
	for i := range orders {
		if err := a.Attribute(&orders[i]); err != nil {
			return err
		}
	}
	return nil
}

// IsFirstPaidOrderForPlayer 查询玩家在该游戏是否已有已支付订单(不包括当前订单号)
func (a *Attributor) IsFirstPaidOrderForPlayer(gameID uint, playerID string, excludeOrderNo string) (bool, error) {
	var count int64
	err := a.DB.Model(&model.Order{}).Where(
		"game_id = ? AND player_id = ? AND status = ? AND order_no <> ?",
		gameID, playerID, model.OrderStatusPaid, excludeOrderNo,
	).Count(&count).Error
	return count == 0, err
}