package model

// GetAllModels 返回所有需要 AutoMigrate 的 model
// 后续阶段按需扩展
func GetAllModels() []interface{} {
	return []interface{}{
		// P1
		&User{},
		&Department{},

		// P2
		&Game{},
		&Dict{},
		&BroadcastTemplate{},

		// P3
		&Lead{},

		// P7 订单数据接入
		&Order{},
		&UploadTemplate{},
		&Partner{},

		// P8 扩展字段配置(管理员定制 lead 表单)
		&LeadExtensionField{},

		// P8 操作审计
		&AuditLog{},

		// P9 玩家列表 —— 跟进记录(聚合到玩家页时间线)
		&LeadFollowRecord{},

		// P11 跟进规则引擎(每日提醒 + 流失判定)
		&LeadReminder{},
		&SystemConfig{},

		// P10 列表视图(玩家 / 订单 列表字段可配置)
		&ListView{},
	}
}