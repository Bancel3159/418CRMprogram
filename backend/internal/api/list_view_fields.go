package api

import (
	"sort"
	"strings"

	"gorm.io/gorm"

	"crm-backend/internal/model"
)

// ListViewAvailableField 字段池里的单个字段元数据
type ListViewAvailableField struct {
	Code      string `json:"code"`       // 字段标识（前端用这个匹配）
	Label     string `json:"label"`      // 默认显示名（用户可改）
	Source    string `json:"source"`     // 数据来源分类
	Formatter string `json:"formatter"`  // 默认格式化器
	GroupKey  string `json:"group_key"`   // 分组 key（前端按组展示）
	GroupName string `json:"group_name"` // 分组中文名
}

// ==================== 玩家视图字段池 ====================

var playerFieldPoolStatic = []ListViewAvailableField{
	// 拉新端核心（最新 lead）
	{Code: "game_code", Label: "游戏", Source: "lead_basic", Formatter: "default", GroupKey: "lead_basic", GroupName: "拉新端核心"},
	{Code: "player_id", Label: "玩家 ID", Source: "lead_basic", Formatter: "default", GroupKey: "lead_basic", GroupName: "拉新端核心"},
	{Code: "latest_username", Label: "玩家名", Source: "lead_basic", Formatter: "default", GroupKey: "lead_basic", GroupName: "拉新端核心"},
	{Code: "latest_country", Label: "国家", Source: "lead_basic", Formatter: "default", GroupKey: "lead_basic", GroupName: "拉新端核心"},
	{Code: "latest_is_target", Label: "是否目标", Source: "lead_basic", Formatter: "target_tag", GroupKey: "lead_basic", GroupName: "拉新端核心"},
	{Code: "latest_region", Label: "区服", Source: "lead_basic", Formatter: "default", GroupKey: "lead_basic", GroupName: "拉新端核心"},
	{Code: "latest_channel", Label: "渠道", Source: "lead_basic", Formatter: "default", GroupKey: "lead_basic", GroupName: "拉新端核心"},
	{Code: "is_paid", Label: "是否付费", Source: "lead_basic", Formatter: "paid_tag", GroupKey: "lead_basic", GroupName: "拉新端核心"},
	{Code: "lead_status", Label: "最新线索状态", Source: "lead_basic", Formatter: "lead_status_tag", GroupKey: "lead_basic", GroupName: "拉新端核心"},

	// 聚合字段
	{Code: "lead_count", Label: "leads", Source: "lead_aggregation", Formatter: "number", GroupKey: "lead_aggregation", GroupName: "聚合"},
	{Code: "creator_names", Label: "拉新员", Source: "lead_aggregation", Formatter: "user_list", GroupKey: "lead_aggregation", GroupName: "聚合"},
	{Code: "claimer_names", Label: "认领员", Source: "lead_aggregation", Formatter: "user_list", GroupKey: "lead_aggregation", GroupName: "聚合"},

	// 时间字段
	{Code: "first_lead_at", Label: "首次线索", Source: "lead_time", Formatter: "datetime", GroupKey: "lead_time", GroupName: "时间"},
	{Code: "last_lead_at", Label: "最后线索", Source: "lead_time", Formatter: "datetime", GroupKey: "lead_time", GroupName: "时间"},

	// 维护端跟进
	{Code: "follow_count", Label: "跟进数", Source: "follow_record", Formatter: "number", GroupKey: "follow_record", GroupName: "维护端跟进"},
	{Code: "last_follow_at", Label: "最后跟进", Source: "follow_record", Formatter: "datetime", GroupKey: "follow_record", GroupName: "维护端跟进"},
	{Code: "last_follow_type", Label: "最后跟进类型", Source: "follow_record", Formatter: "default", GroupKey: "follow_record", GroupName: "维护端跟进"},
	{Code: "last_follow_content", Label: "最后跟进内容", Source: "follow_record", Formatter: "default", GroupKey: "follow_record", GroupName: "维护端跟进"},

	// 订单聚合
	{Code: "order_count", Label: "订单数", Source: "order_summary", Formatter: "number", GroupKey: "order_summary", GroupName: "订单聚合"},
	{Code: "paid_order_count", Label: "已付订单", Source: "order_summary", Formatter: "number", GroupKey: "order_summary", GroupName: "订单聚合"},
	{Code: "total_order_amount", Label: "总金额", Source: "order_summary", Formatter: "money", GroupKey: "order_summary", GroupName: "订单聚合"},
	{Code: "first_paid_at", Label: "首次付费", Source: "order_summary", Formatter: "datetime", GroupKey: "order_summary", GroupName: "订单聚合"},
	{Code: "last_paid_at", Label: "最后付费", Source: "order_summary", Formatter: "datetime", GroupKey: "order_summary", GroupName: "订单聚合"},
}

// ==================== 订单视图字段池 ====================

var orderFieldPoolStatic = []ListViewAvailableField{
	// 订单核心
	{Code: "order_no", Label: "订单号", Source: "order_core", Formatter: "default", GroupKey: "order_core", GroupName: "订单核心"},
	{Code: "player_id", Label: "玩家 ID", Source: "order_core", Formatter: "default", GroupKey: "order_core", GroupName: "订单核心"},
	{Code: "game_code", Label: "游戏", Source: "order_core", Formatter: "default", GroupKey: "order_core", GroupName: "订单核心"},
	{Code: "amount", Label: "金额", Source: "order_core", Formatter: "money", GroupKey: "order_core", GroupName: "订单核心"},
	{Code: "currency", Label: "币种", Source: "order_core", Formatter: "default", GroupKey: "order_core", GroupName: "订单核心"},
	{Code: "status", Label: "订单状态", Source: "order_core", Formatter: "order_status_tag", GroupKey: "order_core", GroupName: "订单核心"},
	{Code: "source", Label: "来源", Source: "order_core", Formatter: "source_tag", GroupKey: "order_core", GroupName: "订单核心"},
	{Code: "attribution_status", Label: "归属状态", Source: "order_core", Formatter: "attribution_tag", GroupKey: "order_core", GroupName: "订单核心"},
	{Code: "is_first_charge", Label: "是否首充", Source: "order_core", Formatter: "first_charge_tag", GroupKey: "order_core", GroupName: "订单核心"},
	{Code: "matched_lead_count", Label: "匹配 lead 数", Source: "order_core", Formatter: "number", GroupKey: "order_core", GroupName: "订单核心"},
	{Code: "remark", Label: "备注", Source: "order_core", Formatter: "default", GroupKey: "order_core", GroupName: "订单核心"},

	// 时间字段
	{Code: "order_time", Label: "下单时间", Source: "order_time", Formatter: "datetime", GroupKey: "order_time", GroupName: "时间"},
	{Code: "paid_at", Label: "支付时间", Source: "order_time", Formatter: "datetime", GroupKey: "order_time", GroupName: "时间"},
	{Code: "attributed_at", Label: "归属时间", Source: "order_time", Formatter: "datetime", GroupKey: "order_time", GroupName: "时间"},
	{Code: "created_at", Label: "入库时间", Source: "order_time", Formatter: "datetime", GroupKey: "order_time", GroupName: "时间"},

	// 关联 lead（通过 lead_id JOIN）
	{Code: "lead_player_username", Label: "线索用户名", Source: "lead_joined", Formatter: "default", GroupKey: "lead_joined", GroupName: "关联线索"},
	{Code: "lead_player_game_name", Label: "线索玩家名", Source: "lead_joined", Formatter: "default", GroupKey: "lead_joined", GroupName: "关联线索"},
	{Code: "lead_player_country", Label: "线索国家", Source: "lead_joined", Formatter: "default", GroupKey: "lead_joined", GroupName: "关联线索"},
	{Code: "lead_region", Label: "线索区服", Source: "lead_joined", Formatter: "default", GroupKey: "lead_joined", GroupName: "关联线索"},
	{Code: "lead_is_target", Label: "线索目标", Source: "lead_joined", Formatter: "target_tag", GroupKey: "lead_joined", GroupName: "关联线索"},
	{Code: "lead_creator", Label: "线索拉新员", Source: "lead_joined", Formatter: "default", GroupKey: "lead_joined", GroupName: "关联线索"},
	{Code: "lead_claimer", Label: "线索认领员", Source: "lead_joined", Formatter: "default", GroupKey: "lead_joined", GroupName: "关联线索"},

	// 订单元数据
	{Code: "attributor", Label: "归属操作人", Source: "order_meta", Formatter: "default", GroupKey: "order_meta", GroupName: "元数据"},
	{Code: "importer", Label: "导入人", Source: "order_meta", Formatter: "default", GroupKey: "order_meta", GroupName: "元数据"},
	{Code: "import_batch", Label: "导入批次", Source: "order_meta", Formatter: "default", GroupKey: "order_meta", GroupName: "元数据"},
	{Code: "partner", Label: "合作方", Source: "order_meta", Formatter: "default", GroupKey: "order_meta", GroupName: "元数据"},
}

// getPlayerFieldPool 玩家可用字段池（含 extension 动态字段）
func (r *Router) getPlayerFieldPool() []ListViewAvailableField {
	out := make([]ListViewAvailableField, 0, len(playerFieldPoolStatic)+16)
	out = append(out, playerFieldPoolStatic...)

	// 动态拼接 extension.* 字段（从 lead_extension_fields 表拉）
	var exts []model.LeadExtensionField
	if err := r.DB.Where("enabled = ?", true).Order("sort ASC, id ASC").Find(&exts).Error; err == nil {
		for _, e := range exts {
			formatter := "default"
			switch e.Type {
			case model.FieldTypeNumber:
				formatter = "number"
			case model.FieldTypeDate:
				formatter = "date"
			}
			out = append(out, ListViewAvailableField{
				Code:      "extension." + e.Code,
				Label:     e.Label,
				Source:    "extension",
				Formatter: formatter,
				GroupKey:  "extension",
				GroupName: "自定义扩展字段",
			})
		}
	}
	return out
}

// getOrderFieldPool 订单可用字段池（含 raw_payload.* 字段）
func (r *Router) getOrderFieldPool() []ListViewAvailableField {
	out := make([]ListViewAvailableField, 0, len(orderFieldPoolStatic)+32)
	out = append(out, orderFieldPoolStatic...)

	// 动态拼接 raw_payload.* 字段：从一条最新订单的 raw_payload 取所有 key
	// (raw_payload 是 JSONB, 不同 actor 不同 ORDB 上传的 key 不同; 取最近 50 条订单 union keys)
	var rows []model.Order
	if err := r.DB.Select("raw_payload").Where("deleted_at IS NULL").
		Order("created_at DESC").Limit(50).Find(&rows).Error; err == nil {
		keySet := map[string]bool{}
		for _, o := range rows {
			if len(o.RawPayload) == 0 {
				continue
			}
			for k := range o.RawPayload {
				if k == "" {
					continue
				}
				keySet[k] = true
			}
		}
		keys := make([]string, 0, len(keySet))
		for k := range keySet {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			out = append(out, ListViewAvailableField{
				Code:      "raw_payload." + k,
				Label:     k,
				Source:    "raw_payload",
				Formatter: "default",
				GroupKey:  "raw_payload",
				GroupName: "数据回传原始字段",
			})
		}
	}
	return out
}

// ==================== 默认视图种子 ====================

// defaultPlayerViewFields 默认视图字段配置（提取自旧的 Players.vue 写死列）
func defaultPlayerViewFields() []map[string]interface{} {
	return []map[string]interface{}{
		{"code": "game_code", "label": "游戏", "source": "lead_basic", "width": 80, "align": "center", "formatter": "default"},
		{"code": "player_id", "label": "玩家 ID", "source": "lead_basic", "width": 160, "align": "left", "formatter": "default"},
		{"code": "latest_username", "label": "玩家名", "source": "lead_basic", "width": 120, "align": "left", "formatter": "default"},
		{"code": "latest_country", "label": "国家", "source": "lead_basic", "width": 80, "align": "left", "formatter": "default"},
		{"code": "latest_is_target", "label": "目标", "source": "lead_basic", "width": 80, "align": "center", "formatter": "target_tag"},
		{"code": "lead_count", "label": "leads", "source": "lead_aggregation", "width": 70, "align": "right", "formatter": "number"},
		{"code": "order_count", "label": "订单", "source": "order_summary", "width": 70, "align": "right", "formatter": "number"},
		{"code": "paid_order_count", "label": "已付", "source": "order_summary", "width": 70, "align": "right", "formatter": "number"},
		{"code": "total_order_amount", "label": "总金额", "source": "order_summary", "width": 110, "align": "right", "formatter": "money"},
		{"code": "last_follow_at", "label": "最后跟进", "source": "follow_record", "width": 160, "align": "left", "formatter": "datetime"},
		{"code": "last_lead_at", "label": "最后 lead", "source": "lead_time", "width": 160, "align": "left", "formatter": "datetime"},
	}
}

// defaultOrderViewFields 默认视图字段配置（提取自旧的 orders/List.vue 写死列）
func defaultOrderViewFields() []map[string]interface{} {
	return []map[string]interface{}{
		{"code": "order_no", "label": "订单号", "source": "order_core", "width": 200, "align": "left", "formatter": "default"},
		{"code": "player_id", "label": "玩家 ID", "source": "order_core", "width": 130, "align": "left", "formatter": "default"},
		{"code": "game_code", "label": "游戏", "source": "order_core", "width": 100, "align": "left", "formatter": "default"},
		{"code": "amount", "label": "金额", "source": "order_core", "width": 100, "align": "right", "formatter": "money"},
		{"code": "status", "label": "状态", "source": "order_core", "width": 80, "align": "center", "formatter": "order_status_tag"},
		{"code": "source", "label": "来源", "source": "order_core", "width": 90, "align": "center", "formatter": "source_tag"},
		{"code": "attribution_status", "label": "归属", "source": "order_core", "width": 100, "align": "center", "formatter": "attribution_tag"},
		{"code": "is_first_charge", "label": "首单", "source": "order_core", "width": 70, "align": "center", "formatter": "first_charge_tag"},
		{"code": "lead_player_country", "label": "线索国家", "source": "lead_joined", "width": 100, "align": "left", "formatter": "default"},
		{"code": "lead_creator", "label": "线索拉新员", "source": "lead_joined", "width": 100, "align": "left", "formatter": "default"},
		{"code": "order_time", "label": "下单时间", "source": "order_time", "width": 160, "align": "left", "formatter": "datetime"},
	}
}

// seedListViews 初始化两个默认视图（idempotent：已存在则跳过）
func seedListViews(db *gorm.DB, adminID uint) error {
	type seedSpec struct {
		entity string
		name   string
		fields []map[string]interface{}
	}
	specs := []seedSpec{
		{entity: "player", name: "默认视图", fields: defaultPlayerViewFields()},
		{entity: "order", name: "默认视图", fields: defaultOrderViewFields()},
	}

	for _, s := range specs {
		var exist model.ListView
		err := db.Where("entity_type = ? AND name = ? AND deleted_at IS NULL", s.entity, s.name).First(&exist).Error
		if err == nil {
			continue
		}
		v := model.ListView{
			EntityType: model.ListViewEntityType(s.entity),
			Name:       s.name,
			IsDefault:  true,
			IsSystem:   true,
			Fields:     model.JSONList(s.fields),
			CreatedBy:  &adminID,
		}
		if err := db.Create(&v).Error; err != nil {
			return err
		}
	}
	return nil
}

// fieldCodeBelongsTo 解析 code 是否属于特定 source 前缀（用于前端 code 校验）
func fieldCodeBelongsTo(code, source string) bool {
	switch source {
	case "lead_basic", "lead_aggregation", "lead_time", "follow_record", "order_summary":
		return strings.HasPrefix(code, "") || strings.Contains(code, ".")
	case "order_core", "order_time", "order_meta", "lead_joined":
		return !strings.Contains(code, ".")
	case "extension":
		return strings.HasPrefix(code, "extension.")
	case "raw_payload":
		return strings.HasPrefix(code, "raw_payload.")
	}
	return false
}

// SeedListViewsForCLI 提供给 cmd/seed 调用的入口
//
// 用 Router 包装一个临时实例,只填 DB 字段,直接调用 seedListViews
func SeedListViewsForCLI(r *Router, adminID uint) error {
	return seedListViews(r.DB, adminID)
}