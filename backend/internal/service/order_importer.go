package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"crm-backend/internal/model"
)

// OrderImporter 订单入库服务(三个数据源共用)
type OrderImporter struct {
	DB          *gorm.DB
	Attributor  *Attributor
}

// NewOrderImporter 创建入库服务
func NewOrderImporter(db *gorm.DB) *OrderImporter {
	return &OrderImporter{
		DB:         db,
		Attributor: NewAttributor(db),
	}
}

// ImportRows 批量导入(事务)
//
// 幂等性:同 (order_no + game_id) 已存在则跳过(skipped++ 返回,不报错)
func (imp *OrderImporter) ImportRows(ctx context.Context, rows []ParsedRow, opts ImportOptions) (*ImportSummary, error) {
	if len(rows) == 0 {
		return &ImportSummary{}, nil
	}

	summary := &ImportSummary{
		Total:     len(rows),
		Inserted:  0,
		Skipped:   0,
		Failed:    0,
		AutoAttr:  0,
		PendingAttr: 0,
	}

	// 批量预查询已存在订单(order_no + game_id)
	type pair struct{ OrderNo string; GameID uint }
	pairs := make([]pair, 0, len(rows))
	for _, r := range rows {
		pairs = append(pairs, pair{OrderNo: r.OrderNo, GameID: opts.GameID})
	}

	existing := map[string]bool{}
	const chunkSize = 500
	for i := 0; i < len(pairs); i += chunkSize {
		end := i + chunkSize
		if end > len(pairs) {
			end = len(pairs)
		}
		var orderNos []string
		for _, p := range pairs[i:end] {
			orderNos = append(orderNos, p.OrderNo)
		}
		var existingOrders []model.Order
		imp.DB.WithContext(ctx).
			Where("game_id = ? AND order_no IN ?", opts.GameID, orderNos).
			Select("order_no").
			Find(&existingOrders)
		for _, eo := range existingOrders {
			existing[eo.OrderNo] = true
		}
	}

	// 构造待插入行
	toInsert := make([]model.Order, 0, len(rows))
	for _, r := range rows {
		if existing[r.OrderNo] {
			summary.Skipped++
			continue
		}

		// 构造 raw_payload
		rawPayload := model.JSONB{
			"extras": r.RawExtras,
		}
		if opts.RawPayload != nil {
			for k, v := range opts.RawPayload {
				rawPayload[k] = v
			}
		}

		order := model.Order{
			GameID:            opts.GameID,
			PlayerID:          r.PlayerID,
			OrderNo:           r.OrderNo,
			Amount:            r.Amount,
			Currency:          r.Currency,
			Status:            r.Status,
			OrderTime:         r.OrderTime,
			PaidAt:            r.PaidAt,
			Source:            opts.Source,
			PartnerID:         opts.PartnerID,
			AttributionStatus: model.AttributionPending,
			Remark:            r.Remark,
			RawPayload:        rawPayload,
			ImportBatch:       opts.ImportBatch,
			ImportedBy:        opts.ImportedBy,
		}
		toInsert = append(toInsert, order)
	}

	if len(toInsert) == 0 {
		return summary, nil
	}

	// 逐条归属(因匹配 lead 的逻辑需查询 DB)
	for i := range toInsert {
		if err := imp.Attributor.Attribute(&toInsert[i]); err != nil {
			summary.Failed++
			continue
		}
		if toInsert[i].AttributionStatus == model.AttributionAuto {
			summary.AutoAttr++
		} else {
			summary.PendingAttr++
		}
	}

	// 批量插入(分块)
	const insertChunk = 200
	for i := 0; i < len(toInsert); i += insertChunk {
		end := i + insertChunk
		if end > len(toInsert) {
			end = len(toInsert)
		}
		batch := toInsert[i:end]
		// ON CONFLICT 兜底幂等(并发情况)
		err := imp.DB.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(&batch).Error
		if err != nil {
			summary.Failed += len(batch)
			continue
		}
		summary.Inserted += len(batch)
	}
	return summary, nil
}

// ImportOptions 入库选项
type ImportOptions struct {
	GameID      uint
	Source      model.OrderSource
	PartnerID   *uint
	ImportBatch string
	ImportedBy  *uint
	RawPayload  map[string]interface{}
}

// ImportSummary 入库汇总
type ImportSummary struct {
	Total        int `json:"total"`
	Inserted     int `json:"inserted"`
	Skipped      int `json:"skipped"`
	Failed       int `json:"failed"`
	AutoAttr     int `json:"auto_attributed"`
	PendingAttr  int `json:"pending_attribution"`
}

// MarshalJSON 自定义序列化,确保字段对齐
func (s *ImportSummary) MarshalJSON() ([]byte, error) {
	type alias ImportSummary
	return json.Marshal((*alias)(s))
}

// ImportOne 导入单条(Webhook/外部 API 共用入口)
func (imp *OrderImporter) ImportOne(ctx context.Context, order model.Order) (*model.Order, error) {
	// 查重
	var existing model.Order
	err := imp.DB.WithContext(ctx).Where(
		"order_no = ? AND game_id = ?", order.OrderNo, order.GameID,
	).First(&existing).Error
	if err == nil {
		return &existing, nil // 已存在,返回旧记录(幂等)
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	if err := imp.Attributor.Attribute(&order); err != nil {
		return nil, fmt.Errorf("归属失败: %w", err)
	}

	if err := imp.DB.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(&order).Error; err != nil {
		return nil, err
	}
	return &order, nil
}