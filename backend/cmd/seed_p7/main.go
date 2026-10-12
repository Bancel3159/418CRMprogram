package main

import (
	"fmt"
	"log"
	"time"

	"crm-backend/internal/config"
	"crm-backend/internal/model"
	"crm-backend/pkg/database"
)

// P7 订单数据种子
// 用法: go run cmd/seed_p7/main.go
//
// 创建:
//   - 1 个 webhook 合作方 + 1 个 api 合作方
//   - 每个游戏 1 个上传模板
//   - 4-5 条示例订单
func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("配置加载失败: %v", err)
	}

	db, err := database.NewPostgres(cfg.Database)
	if err != nil {
		log.Fatalf("PostgreSQL 连接失败: %v", err)
	}
	defer func() {
		sqlDB, _ := db.DB()
		if sqlDB != nil {
			sqlDB.Close()
		}
	}()

	log.Println("🌱 P7 订单种子数据写入...")

	// 0. 取 admin 用户
	var admin model.User
	db.Where("username = ?", "admin").First(&admin)

	// 1. Webhook 合作方
	var webhookPartner model.Partner
	db.Where("code = ?", "demo-webhook").First(&webhookPartner)
	if webhookPartner.ID == 0 {
		webhookPartner = model.Partner{
			Code:       "demo-webhook",
			Name:       "演示支付平台(Webhook)",
			Type:       model.PartnerTypeWebhook,
			Enabled:    true,
			Remark:     "P7 示例 - 接收 webhook 推送",
			HMACSecret: "demo-secret-rotate-me",
		}
		db.Create(&webhookPartner)
		log.Printf("✓ Webhook 合作方 (id=%d)", webhookPartner.ID)
	} else {
		log.Printf("• Webhook 合作方已存在 (id=%d)", webhookPartner.ID)
	}

	// 2. API 合作方
	var apiPartner model.Partner
	db.Where("code = ?", "demo-api").First(&apiPartner)
	if apiPartner.ID == 0 {
		apiPartner = model.Partner{
			Code:     "demo-api",
			Name:     "演示支付平台(API)",
			Type:     model.PartnerTypeAPI,
			Enabled:  true,
			Remark:   "P7 示例 - 定时 API 拉取",
			BaseURL:  "https://api.demo.com/v1",
			CronExpr: "0 0 * * * *",
			AuthConfig: model.JSONB{
				"type":  "bearer",
				"token": "demo-api-token",
			},
			FieldMappings: model.JSONB{
				"orders":     "data",
				"order_no":   "transaction_id",
				"player_id":  "user_id",
				"amount":     "amount_cents",
				"currency":   "currency",
				"status":     "tx_status",
				"order_time": "created_at",
			},
		}
		db.Create(&apiPartner)
		log.Printf("✓ API 合作方 (id=%d)", apiPartner.ID)
	} else {
		log.Printf("• API 合作方已存在 (id=%d)", apiPartner.ID)
	}

	// 3. 上传模板(每个游戏一个)
	var games []model.Game
	db.Find(&games)
	for _, game := range games {
		var tpl model.UploadTemplate
		db.Where("game_id = ? AND name = ?", game.ID, fmt.Sprintf("%s 默认模板", game.Code)).First(&tpl)
		if tpl.ID == 0 {
			tpl = model.UploadTemplate{
				Name:   fmt.Sprintf("%s 默认模板", game.Code),
				GameID: game.ID,
				ColumnMappings: model.JSONB{
					"order_no":   "订单号",
					"player_id":  "玩家ID",
					"amount":     "充值金额",
					"currency":   "币种",
					"status":     "支付状态",
					"order_time": "下单时间",
					"paid_at":    "支付时间",
					"remark":     "备注",
				},
				TimeFormat: "2006-01-02 15:04:05",
				StatusMapping: model.JSONB{
					"已支付": "paid",
					"已退款": "refunded",
					"待支付": "pending",
					"已取消": "cancelled",
				},
				Currency:  "CNY",
				Remark:    fmt.Sprintf("%s 默认上传模板", game.Name),
				CreatedBy: &admin.ID,
			}
			db.Create(&tpl)
			log.Printf("✓ 模板: %s (game=%s)", tpl.Name, game.Code)
		} else {
			log.Printf("• 模板已存在: %s", tpl.Name)
		}
	}

	// 4. 示例订单
	var leads []model.Lead
	db.Where("status = ?", model.LeadStatusClaimed).Limit(3).Find(&leads)
	if len(leads) == 0 {
		db.Where("status <> ?", model.LeadStatusArchived).Limit(3).Find(&leads)
	}

	if len(leads) == 0 {
		log.Println("⚠ 无 lead,跳过订单 seed")
		log.Println("✅ P7 种子完成(无订单)")
		return
	}

	now := time.Now()
	orders := []model.Order{
		{
			LeadID:            &leads[0].ID,
			GameID:            leads[0].GameID,
			PlayerID:          leads[0].PlayerID,
			OrderNo:           fmt.Sprintf("DEMO-%s-001", leads[0].PlayerID),
			Amount:            198.00,
			Currency:          "CNY",
			Status:            model.OrderStatusPaid,
			OrderTime:         now.Add(-48 * time.Hour),
			PaidAt:            timePtr(now.Add(-48 * time.Hour)),
			Source:            model.OrderSourceManual,
			AttributionStatus: model.AttributionAuto,
			IsFirstCharge:     true,
			AttributedAt:      timePtr(now.Add(-47 * time.Hour)),
			MatchedLeadCount:  1,
			Remark:            "演示数据 - 首单 198(A 业绩)",
			RawPayload:        model.JSONB{"seed": true, "demo": "first_charge"},
		},
		{
			LeadID:            &leads[0].ID,
			GameID:            leads[0].GameID,
			PlayerID:          leads[0].PlayerID,
			OrderNo:           fmt.Sprintf("DEMO-%s-002", leads[0].PlayerID),
			Amount:            328.00,
			Currency:          "CNY",
			Status:            model.OrderStatusPaid,
			OrderTime:         now.Add(-24 * time.Hour),
			PaidAt:            timePtr(now.Add(-24 * time.Hour)),
			Source:            model.OrderSourceWebhook,
			PartnerID:         &webhookPartner.ID,
			AttributionStatus: model.AttributionAuto,
			IsFirstCharge:     false,
			AttributedAt:      timePtr(now.Add(-23 * time.Hour)),
			MatchedLeadCount:  1,
			Remark:            "演示数据 - 续单 328(B 业绩)",
			RawPayload:        model.JSONB{"seed": true, "demo": "repeat"},
		},
		{
			LeadID:            nil,
			GameID:            leads[0].GameID,
			PlayerID:          "UNKNOWN_PLAYER_DEMO",
			OrderNo:           "DEMO-UNKNOWN-001",
			Amount:            50.00,
			Currency:          "CNY",
			Status:            model.OrderStatusPaid,
			OrderTime:         now.Add(-12 * time.Hour),
			PaidAt:            timePtr(now.Add(-12 * time.Hour)),
			Source:            model.OrderSourceAPI,
			PartnerID:         &apiPartner.ID,
			AttributionStatus: model.AttributionPending,
			MatchedLeadCount:  0,
			Remark:            "演示数据 - 玩家未提交过线索,待人工绑定",
			RawPayload:        model.JSONB{"seed": true},
		},
	}

	if len(leads) >= 2 {
		orders = append(orders, model.Order{
			LeadID:            &leads[1].ID,
			GameID:            leads[1].GameID,
			PlayerID:          leads[1].PlayerID,
			OrderNo:           fmt.Sprintf("DEMO-%s-001", leads[1].PlayerID),
			Amount:            88.00,
			Currency:          "CNY",
			Status:            model.OrderStatusRefunded,
			OrderTime:         now.Add(-72 * time.Hour),
			PaidAt:            timePtr(now.Add(-72 * time.Hour)),
			Source:            model.OrderSourceManual,
			AttributionStatus: model.AttributionAuto,
			IsFirstCharge:     true,
			AttributedAt:      timePtr(now.Add(-71 * time.Hour)),
			MatchedLeadCount:  1,
			Remark:            "演示数据 - 退款订单 88",
			RawPayload:        model.JSONB{"seed": true, "demo": "refunded"},
		})
	}

	for _, o := range orders {
		var existing model.Order
		err := db.Where("order_no = ? AND game_id = ?", o.OrderNo, o.GameID).First(&existing).Error
		if err == nil {
			log.Printf("• 订单已存在: %s", o.OrderNo)
			continue
		}
		if err := db.Create(&o).Error; err != nil {
			log.Printf("❌ 订单 %s 入库失败: %v", o.OrderNo, err)
		} else {
			log.Printf("✓ 订单 %s (¥%.2f, 来源=%s, 归属=%s)",
				o.OrderNo, o.Amount, o.Source, o.AttributionStatus)
		}
	}

	log.Println("✅ P7 种子数据写入完成")
}

func timePtr(t time.Time) *time.Time {
	return &t
}