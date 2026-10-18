package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"os"
	"time"

	"gorm.io/gorm"

	"crm-backend/internal/config"
	"crm-backend/internal/model"
	"crm-backend/internal/service"
	"crm-backend/pkg/database"
	"crm-backend/pkg/password"
)

// seed_p14 全链路种子数据(P14)
//
// 目的:现有数据太薄(9 线索 / 30 订单),follow 端复用 web 逻辑时
// 大量边界分支没被覆盖。本脚本造一批贴近真实的全链路数据,
// 让 follow 端开发时能立刻暴露数据通路问题。
//
// 覆盖场景(每个玩家走一条不同的生命周期):
//   L1 新注册未认领        → 公开池 / 今日待办 follow_new_register
//   L2 已认领未跟进        → 今日待办 follow_new_register
//   L3 已跟进已付费        → 今日待办 follow_paid_daily
//   L4 付费玩家即将离线    → 今日待办 follow_paid_offline
//   L5 付费玩家已离线流失  → is_lost = true
//   L6 流失后重新激活      → is_lost = false + reactivated_at
//   L7 已归档 + 历史订单   → 验证归档 lead 的订单归属(P14 修的点)
//   L8 多渠道/多区服      → 筛选和分组测试
//   L9 扩展字段有值       → list_view extension.* 渲染
//   L10 超长文本/特殊字符 → 表格 ellipsis / tooltip
//
// 用法: go run cmd/seed_p14/main.go [-clean]
//
//	-clean  先清掉本脚本造的数据(player_id 前缀 P14-),再重新生成

const prefix = "P14-"

func main() {
	clean := hasCleanFlag()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("配置加载失败: %v", err)
	}
	db, err := database.NewPostgres(cfg.Database)
	if err != nil {
		log.Fatalf("数据库连接失败: %v", err)
	}
	defer func() {
		if sqlDB, e := db.DB(); e == nil && sqlDB != nil {
			sqlDB.Close()
		}
	}()

	log.Println("🌱 P14 全链路种子数据...")

	if clean {
		cleanAll(db)
		log.Println("✓ 已清空 P14 数据")
	}

	// ---- 1. 游戏(6 个,覆盖多游戏场景) ----
	games := seedGames(db)

	// ---- 2. 用户(A 部拉新 / B 部维护) ----
	aStaff, bStaff, bLeads := seedUsers(db)

	// ---- 3. 玩家 + 线索 + 订单 + 跟进记录(10 条生命周期) ----
	players := seedPlayers(db, games, aStaff, bStaff, bLeads)

	// ---- 4. 触发归属(验证修好的 fallback 是否生效) ----
	reattributeAll(db)

	// ---- 5. 触发跟进规则引擎 ----
	genReminders(db, players)

	// ---- 6. 汇总 ----
	report(db)
}

func hasCleanFlag() bool {
	for _, a := range os.Args {
		if a == "-clean" || a == "--clean" {
			return true
		}
	}
	return false
}

var rng = rand.New(rand.NewSource(20261018)) // 固定种子 → 可复现

// ==================== 清理 ====================

func cleanAll(db *gorm.DB) {
	// 按 player_id 前缀删
	db.Where("player_id LIKE ?", prefix+"%").Delete(&model.Order{})
	db.Where("player_id LIKE ?", prefix+"%").Delete(&model.Lead{})
	db.Exec("DELETE FROM lead_follow_records WHERE lead_id IN (SELECT id FROM leads WHERE player_id LIKE ?)", prefix+"%")
	db.Exec("DELETE FROM lead_reminders WHERE lead_id IN (SELECT id FROM leads WHERE player_id LIKE ?)", prefix+"%")
	log.Println("  已清理 orders / leads / follow_records / reminders")
}

// ==================== 游戏 ====================

type seedGame struct {
	ID     uint
	Code   string
	Name   string
	HasWH  bool // 是否配 webhook(测 webhook 通路)
	Secret string
}

func seedGames(db *gorm.DB) []seedGame {
	defs := []struct {
		Code, Name, Remark string
		WH                 bool
	}{
		{"P14TWD", "The Walking Dead Survivors 生存者", "主线产品", true},
		{"P14GGW", "Grand Gangster War 黑帮大作战", "东南亚", true},
		{"P14RIV", "Rivals of Aether 元素对决", "欧美", true},
		{"P14COD", "Call of Duty Mobile 使命召唤", "全球", false},
		{"P14APEX", "Apex Legends Mobile 顶点", "全球", false},
		{"P14PUBG", "PUBG Mobile 吃鸡", "日韩+东南亚", false},
	}

	out := make([]seedGame, 0, len(defs))
	for i, d := range defs {
		var g model.Game
		err := db.Where("code = ?", d.Code).First(&g).Error
		if err != nil {
			g = model.Game{
				Code:   d.Code,
				Name:   d.Name,
				Remark: d.Remark,
				Status: model.GameStatusEnabled,
			}
			if d.WH {
				secret := fmt.Sprintf("p14-secret-%s", d.Code)
				g.PartnerType = ptrPartnerType(model.PartnerTypeWebhook)
				g.PartnerConfig = model.JSONB{"hmac_secret": secret}
			}
			if err := db.Create(&g).Error; err != nil {
				log.Printf("  ❌ 游戏 %s 创建失败: %v", d.Code, err)
				continue
			}
		}
		out = append(out, seedGame{ID: g.ID, Code: g.Code, Name: g.Name, HasWH: d.WH, Secret: fmt.Sprintf("p14-secret-%s", d.Code)})
		_ = i
	}
	log.Printf("✓ 游戏 %d 个", len(out))
	return out
}

func ptrPartnerType(t model.PartnerType) *model.PartnerType { return &t }

// ==================== 用户 ====================

func seedUsers(db *gorm.DB) (aStaff, bStaff, bLeads []model.User) {
	defs := []struct {
		User, Name string
		Role       model.Role
		Superior   string
	}{
		{"p14_qingtian", "晴天", model.RoleALead, ""},
		{"p14_beike", "贝壳", model.RoleAStaff, "p14_qingtian"},
		{"p14_xingdong", "兴东", model.RoleAManager, ""},
		{"p14_liuliu", "刘六", model.RoleAStaff, "p14_xingdong"},

		{"p14_ningchuan", "宁川", model.RoleBLead, ""},
		{"p14_wangguan", "小关", model.RoleBManager, ""},
		{"p14_guoyuan", "果圆", model.RoleBStaff, "p14_wangguan"},
	}
	for _, d := range defs {
		var u model.User
		if err := db.Where("username = ?", d.User).First(&u).Error; err != nil {
			// 密码统一 test123(测试用)
			h, _ := hashPassword("test123")
			u = model.User{
				Username:     d.User,
				PasswordHash: h,
				RealName:     d.Name,
				Role:         d.Role,
				Status:       model.UserStatusEnabled, // 枚举是 enabled/disabled,不是 active
			}
			if d.Superior != "" {
				var sup model.User
				if err := db.Where("username = ?", d.Superior).First(&sup).Error; err == nil {
					u.SuperiorID = &sup.ID
				}
			}
			if err := db.Create(&u).Error; err != nil {
				log.Printf("  ❌ 用户 %s 失败: %v", d.User, err)
				continue
			}
		}
		switch d.Role {
		case model.RoleALead:
			aStaff = append(aStaff, u)
		case model.RoleAStaff:
			if aStaff[0].ID == u.ID {
				// p14_beike
			} else {
				aStaff = append(aStaff, u)
			}
		case model.RoleBLead:
			bLeads = append(bLeads, u)
		case model.RoleBManager:
			bStaff = append(bStaff, u)
		case model.RoleBStaff:
			bStaff = append(bStaff, u)
		}
	}
	log.Printf("✓ 用户 A部%d / B部%d", len(aStaff), len(bStaff)+len(bLeads))
	return
}

func hashPassword(pw string) (string, error) {
	return password.Hash(pw)
}

// ==================== 玩家 / 线索 / 订单 ====================

type seedPlayer struct {
	PlayerID string
	GameID   uint
	LeadID   uint
	Scenario string
}

var (
	channels   = []string{"tiktok", "discord", "whatsapp", "youtube", "instagram", "telegram"}
	countries  = []string{"美国", "BR", "JP", "ID", "TH", "VN", "MY", "新加坡", "菲律宾", "越南"}
	regions    = []string{"SEA-01", "NA-02", "EU-03", "JP-04", "KR-05"}
	commus     = []string{"discord_gg", "discord_twd", "telegram_main", "qq_group_1", "wa_community"}
	gameNames  = []string{"Pro", "Lt", "Zer", "Vip", "Noob", "Ace", "Star", "King"}
	longRemark = "这是一条很长的备注用来测试表格 ellipsis 和 tooltip 展示效果，包含中文、English、特殊符号 !@#$%^&*() 和数字 1234567890 以及 emoji 🚀🎯💰"
)

func seedPlayers(db *gorm.DB, games []seedGame, aStaff, bStaff, bLeads []model.User) []seedPlayer {
	if len(aStaff) == 0 || len(games) == 0 {
		log.Println("❌ 缺基础数据(用户/游戏),无法生成玩家")
		return nil
	}

	// 10 个场景 × 3 个玩家 = 30 个玩家,分散到 6 个游戏
	scenarios := []string{
		"L1_新注册未认领",
		"L2_已认领未跟进",
		"L3_已跟进已付费",
		"L4_付费即将离线",
		"L5_付费已离线流失",
		"L6_流失后重新激活",
		"L7_已归档带历史订单",
		"L8_多渠道多区服",
		"L9_有扩展字段",
		"L10_超长文本",
	}

	now := time.Now()
	out := make([]seedPlayer, 0, len(scenarios)*3)
	idx := 0

	for si, scen := range scenarios {
		for k := 0; k < 3; k++ {
			idx++
			g := games[(si*3+k)%len(games)]
			creator := aStaff[(si+k)%len(aStaff)]
			playerID := fmt.Sprintf("%sP%02d_%s_%s", prefix, si+1, g.Code, gameNames[rng.Intn(len(gameNames))])

			now2 := now.Add(-time.Duration(rng.Intn(30*24)) * time.Hour)
			lead := model.Lead{
				GameID:         g.ID,
				CreatorID:      creator.ID,
				Channel:        channels[(si*3+k)%len(channels)],
				PlayerUsername: fmt.Sprintf("player_%02d", idx),
				PlayerGameName: fmt.Sprintf("%s_%d", g.Code, idx),
				PlayerID:       playerID,
				Region:         regions[rng.Intn(len(regions))],
				PlayerCountry:  countries[rng.Intn(len(countries))],
				PlayerAge:      18 + rng.Intn(25),
				IsTarget:       pickTarget(),
				Community:      commus[rng.Intn(len(commus))],
				TokenCode:      "",
				Screenshots:    model.Strings{},
				Status:         model.LeadStatusPending,
				Remark:         "",
			}
			if si == 9 {
				lead.Remark = longRemark
			}
			if si == 8 {
				lead.Extension = model.JSONB{
					"discord_id": fmt.Sprintf("user#%d", 1000+idx),
					"level":      10 + idx,
					"guild":      "warriors",
					"notes":      "从 API 拉取的扩展字段",
				}
			}
			lead.CreatedAt = now2
			if err := db.Create(&lead).Error; err != nil {
				log.Printf("  ❌ lead %s 创建失败: %v", playerID, err)
				continue
			}

			applyScenario(db, &lead, si, scen, now, now2, bStaff, bLeads)
			out = append(out, seedPlayer{
				PlayerID: playerID, GameID: g.ID, LeadID: lead.ID, Scenario: scen,
			})
		}
	}
	log.Printf("✓ 线索 %d 条(10 场景 × 3)", len(out))
	return out
}

func pickTarget() model.IsTarget {
	if rng.Intn(10) < 7 {
		return model.IsTargetTrue
	}
	return model.IsTargetFalse
}

// applyScenario 按场景设置 lead 状态 + 生成订单 / 跟进记录
func applyScenario(db *gorm.DB, lead *model.Lead, si int, scen string, now, createdAt time.Time, bStaff, bLeads []model.User) {
	claimer := func() *uint {
		pool := bLeads
		if len(pool) == 0 {
			pool = bStaff
		}
		if len(pool) == 0 {
			return nil
		}
		u := pool[rng.Intn(len(pool))]
		return &u.ID
	}

	switch si {
	case 0: // L1 新注册未认领 —— 保持 pending,无订单
		// do nothing

	case 1: // L2 已认领未跟进 —— claimed + claimed_at,但没有 follow_record
		lead.Status = model.LeadStatusClaimed
		lead.ClaimedBy = claimer()
		t := createdAt.Add(2 * time.Hour)
		lead.ClaimedAt = &t

	case 2: // L3 已跟进已付费 —— claimed + 1 条跟进 + 1-2 笔 paid 订单
		lead.Status = model.LeadStatusClaimed
		lead.ClaimedBy = claimer()
		t := createdAt.Add(2 * time.Hour)
		lead.ClaimedAt = &t
		createFollow(db, lead.ID, lead.ClaimedBy, t.Add(1*time.Hour), "text", "已在 Discord 首次触达,玩家回复了价格问题")
		mkOrders(db, lead, 1, 2, t.Add(3*time.Hour), now)

	case 3: // L4 付费玩家即将离线 —— 离线 2-3 天(未到流失线)
		lead.Status = model.LeadStatusClaimed
		lead.ClaimedBy = claimer()
		t := createdAt.Add(2 * time.Hour)
		lead.ClaimedAt = &t
		createFollow(db, lead.ID, lead.ClaimedBy, t.Add(time.Hour), "text", "完成首充引导,赠送了新手礼包")
		mkOrders(db, lead, 2, 4, t.Add(2*time.Hour), now.Add(-72*time.Hour))

	case 4: // L5 付费玩家已离线流失 —— is_lost = true
		lead.Status = model.LeadStatusClaimed
		lead.ClaimedBy = claimer()
		t := createdAt.Add(2 * time.Hour)
		lead.ClaimedAt = &t
		createFollow(db, lead.ID, lead.ClaimedBy, t.Add(time.Hour), "text", "玩家已充值,持续跟进中")
		mkOrders(db, lead, 2, 5, t.Add(2*time.Hour), now.Add(-10*24*time.Hour))
		lost := now.Add(-9 * 24 * time.Hour)
		lead.IsLost = true
		lead.LostAt = &lost
		lead.LostReason = "auto_paid_offline"

	case 5: // L6 流失后重新激活 —— 先 lost 后 active
		lead.Status = model.LeadStatusClaimed
		lead.ClaimedBy = claimer()
		t := createdAt.Add(2 * time.Hour)
		lead.ClaimedAt = &t
		createFollow(db, lead.ID, lead.ClaimedBy, t.Add(time.Hour), "text", "首充完成")
		mkOrders(db, lead, 1, 2, t.Add(2*time.Hour), now.Add(-20*24*time.Hour))
		lost := now.Add(-15 * 24 * time.Hour)
		lead.IsLost = true
		lead.LostAt = &lost
		lead.LostReason = "auto_paid_offline"
		react := now.Add(-5 * 24 * time.Hour)
		lead.IsLost = false
		lead.LostReason = ""
		lead.ReactivatedAt = &react
		active := now.Add(-5 * 24 * time.Hour)
		lead.LastPlayerActiveAt = &active

	case 6: // L7 已归档 + 历史订单 —— P14 修的归属 fallback 关键场景
		lead.Status = model.LeadStatusArchived
		t := createdAt.Add(2 * time.Hour)
		lead.ClaimedAt = &t
		lead.ClaimedBy = claimer()
		archivedAt := now.Add(-20 * 24 * time.Hour)
		lead.ReleasedAt = &archivedAt
		lead.ReleasedBy = lead.ClaimedBy
		mkOrders(db, lead, 1, 3, t.Add(time.Hour), now.Add(-25*24*time.Hour))

	case 7: // L8 多渠道多区服 —— claimed + 未付费 + 多条跟进
		lead.Status = model.LeadStatusClaimed
		lead.ClaimedBy = claimer()
		t := createdAt.Add(time.Hour)
		lead.ClaimedAt = &t
		createFollow(db, lead.ID, lead.ClaimedBy, t.Add(time.Hour), "text", "首次电话沟通,玩家在比价")
		createFollow(db, lead.ID, lead.ClaimedBy, t.Add(48*time.Hour), "wechat", "发送了活动截图")
		createFollow(db, lead.ID, lead.ClaimedBy, t.Add(96*time.Hour), "text", "玩家说下周再决定")

	case 8: // L9 有扩展字段 —— claimed + 付费
		lead.Status = model.LeadStatusClaimed
		lead.ClaimedBy = claimer()
		t := createdAt.Add(2 * time.Hour)
		lead.ClaimedAt = &t
		createFollow(db, lead.ID, lead.ClaimedBy, t.Add(time.Hour), "text", "已加上游戏内好友")
		mkOrders(db, lead, 1, 3, t.Add(2*time.Hour), now.Add(-48*time.Hour))

	case 9: // L10 超长文本 —— claimed + 付费 + 超长备注
		lead.Status = model.LeadStatusClaimed
		lead.ClaimedBy = claimer()
		t := createdAt.Add(2 * time.Hour)
		lead.ClaimedAt = &t
		createFollow(db, lead.ID, lead.ClaimedBy, t.Add(time.Hour), "text", longRemark)
		mkOrders(db, lead, 2, 6, t.Add(2*time.Hour), now.Add(-24*time.Hour))
	}

	db.Save(lead)
}

func createFollow(db *gorm.DB, leadID uint, userID *uint, at time.Time, ftype, content string) {
	if userID == nil {
		return
	}
	r := model.LeadFollowRecord{
		LeadID:     leadID,
		FollowerID: userID, // model 字段本身就是 *uint
		FollowType: ftype,
		Content:    content,
		CreatedAt:  at,
	}
	if err := db.Create(&r).Error; err != nil {
		log.Printf("    ❌ 跟进记录失败: %v", err)
	}
}

// mkOrders 生成 n~maxN 笔订单,时间分布在 [firstAt, lastAt]
func mkOrders(db *gorm.DB, lead *model.Lead, minN, maxN int, firstAt, lastAt time.Time) {
	if lastAt.Before(firstAt) {
		lastAt = firstAt.Add(24 * time.Hour)
	}
	n := minN
	if maxN > minN {
		n = minN + rng.Intn(maxN-minN+1)
	}
	span := lastAt.Sub(firstAt)
	for i := 0; i < n; i++ {
		at := firstAt
		if span > 0 {
			at = firstAt.Add(time.Duration(rng.Int63n(int64(span))))
		}
		// 70% 已支付,30% pending
		status := model.OrderStatusPaid
		if rng.Intn(10) < 3 {
			status = model.OrderStatusPending
		}
		amounts := []float64{6.99, 9.99, 19.99, 29.99, 49.99, 68.00, 98.00, 128.00, 198.00, 298.00, 488.00}
		amt := amounts[rng.Intn(len(amounts))]
		// 15% 造高额订单(测 realtime.high_value 阈值)
		if rng.Intn(100) < 15 {
			amt = 100 + float64(rng.Intn(800)) // 100-900
		}
		orderNo := fmt.Sprintf("%sORD-%d-%d", prefix, lead.ID, i+1)
		var paidAt *time.Time
		if status == model.OrderStatusPaid {
			p := at
			paidAt = &p
			lead.PaidMark = true
		}
		o := model.Order{
			GameID:    lead.GameID,
			PlayerID:  lead.PlayerID,
			OrderNo:   orderNo,
			Amount:    amt,
			Currency:  "CNY",
			Status:    status,
			OrderTime: at,
			PaidAt:    paidAt,
			Source:    model.OrderSourceWebhook,
			ImportBatch: "p14-seed",
			RawPayload: model.JSONB{"seed": true, "scenario": lead.Remark[:min(20, len(lead.Remark))]},
		}
		if err := db.Create(&o).Error; err != nil {
			log.Printf("    ❌ 订单 %s 失败: %v", orderNo, err)
		}
	}
}

// ==================== 重跑归属 ====================

func reattributeAll(db *gorm.DB) {
	attr := service.NewAttributor(db)
	var orders []model.Order
	if err := db.Where("import_batch = ?", "p14-seed").Find(&orders).Error; err != nil {
		log.Printf("❌ 读取订单失败: %v", err)
		return
	}
	ok, pending := 0, 0
	for i := range orders {
		if err := attr.Attribute(&orders[i]); err != nil {
			continue
		}
		// Attribute 只改内存,需要手动落库
		if orders[i].AttributionStatus == model.AttributionAuto {
			db.Model(&model.Order{}).Where("id = ?", orders[i].ID).
				Updates(map[string]interface{}{
					"lead_id":            orders[i].LeadID,
					"attribution_status": orders[i].AttributionStatus,
					"attributed_at":      orders[i].AttributedAt,
					"matched_lead_count": orders[i].MatchedLeadCount,
					"is_first_charge":    orders[i].IsFirstCharge,
				})
			ok++
		} else {
			pending++
		}
	}
	log.Printf("✓ 归属重跑: 自动 %d / 待人工 %d", ok, pending)
}

// ==================== 触发规则引擎 ====================

func genReminders(db *gorm.DB, players []seedPlayer) {
	// 走引擎的全量扫描(和 cron 每天跑的是同一条路径)
	engine := service.NewFollowReminderEngine(db)
	res, err := engine.DailyScan(context.Background())
	if err != nil {
		log.Printf("❌ DailyScan 失败: %v", err)
		return
	}
	log.Printf("✓ 提醒生成: 新注册 %d / 付费每日 %d / 离线升级 %d / 过期清理 %d / 自动流失(新注册 %d,付费 %d)",
		res.NewReminders, res.PaidDailyReminders, res.OfflineAlerts, res.ExpiredReminders,
		res.AutoLostNewRegister, res.AutoLostPaidOffline)
}

// ==================== 汇总 ====================

func report(db *gorm.DB) {
	var leads, orders, follows, rems, attributed int64
	var paidOrders int64
	var lostCount int64

	db.Model(&model.Lead{}).Where("player_id LIKE ?", prefix+"%").Count(&leads)
	db.Model(&model.Order{}).Where("player_id LIKE ?", prefix+"%").Count(&orders)
	db.Model(&model.LeadFollowRecord{}).
		Where("lead_id IN (SELECT id FROM leads WHERE player_id LIKE ?)", prefix+"%").Count(&follows)
	db.Model(&model.LeadReminder{}).
		Where("lead_id IN (SELECT id FROM leads WHERE player_id LIKE ?)", prefix+"%").Count(&rems)
	db.Model(&model.Order{}).Where("player_id LIKE ? AND attribution_status <> ?", prefix+"%", model.AttributionPending).Count(&attributed)
	db.Model(&model.Order{}).Where("player_id LIKE ? AND status = ?", prefix+"%", model.OrderStatusPaid).Count(&paidOrders)
	db.Model(&model.Lead{}).Where("player_id LIKE ? AND is_lost = true", prefix+"%").Count(&lostCount)

	fmt.Println()
	fmt.Println("========== P14 种子汇总 ==========")
	fmt.Printf("  线索          %d\n", leads)
	fmt.Printf("  订单          %d (已付 %d)\n", orders, paidOrders)
	fmt.Printf("  已归属        %d (待人工 %d)\n", attributed, orders-attributed)
	fmt.Printf("  跟进记录      %d\n", follows)
	fmt.Printf("  提醒          %d\n", rems)
	fmt.Printf("  流失线索      %d\n", lostCount)
	fmt.Println("==================================")
	fmt.Println()
	fmt.Println("测试账号(密码都是 test123):")
	fmt.Println("  A 部: p14_xingdong(主管) / p14_qingtian(组长) / p14_beike(员工)")
	fmt.Println("  B 部: p14_wangguan(主管) / p14_ningchuan(组长) / p14_guoyuan(员工)")
	fmt.Println()
}
