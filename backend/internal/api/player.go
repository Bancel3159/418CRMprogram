package api

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"crm-backend/internal/middleware"
	"crm-backend/internal/model"
	"crm-backend/pkg/response"
)

// PlayerSummary 玩家聚合视图(列表行 / 详情头部共用)
//
// 玩家 = (player_id, game_id) 联合唯一,从 leads 表按这俩字段 GROUP BY 派生。
// 不建独立 players 表 —— 一切动态聚合,避免数据漂移。
type PlayerSummary struct {
	PlayerID       string     `json:"player_id"`
	GameID         uint       `json:"game_id"`
	GameCode       string     `json:"game_code,omitempty"`
	GameName       string     `json:"game_name,omitempty"`
	LatestLeadID   uint       `json:"latest_lead_id"`   // P13 强制编辑入口用(最新 lead 的 id)
	LatestUsername string     `json:"latest_username"`
	LatestCountry  string     `json:"latest_country"`
	LatestIsTarget string     `json:"latest_is_target"`

	LeadCount      int64      `json:"lead_count"`
	OrderCount     int64      `json:"order_count"`
	PaidOrderCount int64      `json:"paid_order_count"`
	TotalOrderAmount float64  `json:"total_order_amount"`

	FirstLeadAt  *time.Time `json:"first_lead_at"`
	LastLeadAt   *time.Time `json:"last_lead_at"`
	LastFollowAt *time.Time `json:"last_follow_at"`

	// 人员相关
	CreatorIDs    []uint `json:"creator_ids"`
	ClaimedByIDs  []uint `json:"claimed_by_ids"`
	LeadIDs       []uint `json:"lead_ids"`

	// 扩展字段最新值(从最新一条 lead 抓)
	Extension map[string]any `json:"extension,omitempty"`
}

// playerList 玩家列表(GET /api/v1/players)
//
// 筛选:
//   - player_id  (模糊匹配)
//   - game_id
//   - country    (模糊匹配)
//   - is_target  (target / not_target)
//   - creator_id (创建者)
//   - claimed_by (认领人)
//   - has_orders (true / false,基于 paid 订单)
//   - last_follow_from / last_follow_to  (基于 last_follow_at)
//
// RBAC:
//   - admin:全量
//   - 主管:自己 + 直属下属的 lead 聚合出的玩家
//   - 员工:仅自己创建或认领的 lead 聚合出的玩家
func (r *Router) playerList(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	role := middleware.GetCurrentRole(c)
	userID := middleware.GetCurrentUserID(c)

	// 基础过滤(从 leads 派生玩家)
	base := r.DB.Model(&model.Lead{})

	// 权限过滤
	switch role {
	case model.RoleAdmin:
		// 全量
	case model.RoleAStaff, model.RoleBStaff:
		base = base.Where("creator_id = ? OR claimed_by = ?", userID, userID)
	case model.RoleALead, model.RoleBLead:
		// 组长:沿用 leadList 的语义(本人 + 自己下属,实际数据可访问性允许看自己的)
		base = base.Where("creator_id = ? OR claimed_by = ?", userID, userID)
	case model.RoleAManager, model.RoleBManager:
		// 主管:自己 + 直属下属
		ids, err := r.subordinateIDs(userID)
		if err != nil {
			response.ServerError(c, "权限查询失败: "+err.Error())
			return
		}
		base = base.Where("creator_id IN ? OR claimed_by IN ?", ids, ids)
	default:
		response.Forbidden(c, "无权访问玩家列表")
		return
	}

	// 用户筛选
	if v := strings.TrimSpace(c.Query("player_id")); v != "" {
		base = base.Where("player_id ILIKE ?", "%"+v+"%")
	}
	if v := c.Query("game_id"); v != "" {
		if id, err := strconv.ParseUint(v, 10, 64); err == nil {
			base = base.Where("game_id = ?", id)
		}
	}
	if v := strings.TrimSpace(c.Query("country")); v != "" {
		base = base.Where("player_country ILIKE ?", "%"+v+"%")
	}
	if v := c.Query("is_target"); v != "" {
		base = base.Where("is_target = ?", v)
	}
	if v := c.Query("creator_id"); v != "" {
		if id, err := strconv.ParseUint(v, 10, 64); err == nil {
			base = base.Where("creator_id = ?", id)
		}
	}
	if v := c.Query("claimed_by"); v != "" {
		if id, err := strconv.ParseUint(v, 10, 64); err == nil {
			base = base.Where("claimed_by = ?", id)
		}
	}

	// 子查询:有 / 无订单(基于 paid 订单)
	hasOrders := strings.ToLower(c.Query("has_orders"))
	if hasOrders == "true" || hasOrders == "false" {
		sub := r.DB.Model(&model.Order{}).
			Select("player_id, game_id").
			Where("status = ?", model.OrderStatusPaid).
			Where("deleted_at IS NULL")
		if hasOrders == "true" {
			base = base.Where("EXISTS (?)", sub).
				Where("(leads.player_id, leads.game_id) IN (?)", sub)
		} else {
			base = base.Where("NOT EXISTS (?)", sub)
		}
	}

	// 跟进时间范围(基于该玩家最后一次跟进)
	if v := c.Query("last_follow_from"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			// 用子查询 last_follow_at >= t
			sub := r.DB.Model(&model.LeadFollowRecord{}).
				Select("lead_id, MAX(created_at) AS last_at").
				Group("lead_id")
			base = base.Where("leads.id IN (SELECT lead_id FROM (?) AS fr WHERE fr.last_at >= ?)", sub, t)
		}
	}
	if v := c.Query("last_follow_to"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			sub := r.DB.Model(&model.LeadFollowRecord{}).
				Select("lead_id, MAX(created_at) AS last_at").
				Group("lead_id")
			base = base.Where("leads.id IN (SELECT lead_id FROM (?) AS fr WHERE fr.last_at <= ?)", sub, t)
		}
	}

	// 玩家总人数(GROUP BY 后)
	var total int64
	r.DB.Raw("SELECT COUNT(*) FROM (SELECT 1 FROM (?) AS l GROUP BY l.player_id, l.game_id) AS p", base).Scan(&total)

	// 聚合查询 —— GROUP BY player_id, game_id,取最新 lead 的 username/country/is_target/extension
	// 用窗口函数取第一条(最新 lead)的信息
	type rowAgg struct {
		PlayerID       string
		GameID         uint
		LatestLeadID   uint
		LatestUsername string
		LatestCountry  string
		LatestIsTarget string
		LeadCount      int64
		FirstLeadAt    time.Time
		LastLeadAt     time.Time
		Extension      []byte
		CreatorIDs     string // array_agg
		ClaimedByIDs   string // array_agg
		LeadIDs        string // array_agg
	}

	// raw SQL 一次取齐(player 行级聚合)
	// base 作为子查询直接嵌进 CTE
	q := `
		WITH leads_filtered AS (?) 
		SELECT
			l.player_id,
			l.game_id,
			(SELECT id FROM leads_filtered lf
			   WHERE lf.player_id = l.player_id AND lf.game_id = l.game_id
			   ORDER BY lf.created_at DESC LIMIT 1) AS latest_lead_id,
			COALESCE((SELECT player_username FROM leads_filtered lf
			   WHERE lf.player_id = l.player_id AND lf.game_id = l.game_id
			   ORDER BY lf.created_at DESC LIMIT 1), '') AS latest_username,
			COALESCE((SELECT player_country FROM leads_filtered lf
			   WHERE lf.player_id = l.player_id AND lf.game_id = l.game_id
			   ORDER BY lf.created_at DESC LIMIT 1), '') AS latest_country,
			COALESCE((SELECT is_target FROM leads_filtered lf
			   WHERE lf.player_id = l.player_id AND lf.game_id = l.game_id
			   ORDER BY lf.created_at DESC LIMIT 1), '') AS latest_is_target,
			COUNT(*) AS lead_count,
			MIN(l.created_at) AS first_lead_at,
			MAX(l.created_at) AS last_lead_at,
			(SELECT extension FROM leads_filtered lf
			   WHERE lf.player_id = l.player_id AND lf.game_id = l.game_id
			   ORDER BY lf.created_at DESC LIMIT 1) AS extension,
			COALESCE(array_agg(DISTINCT l.creator_id), '{}') AS creator_ids,
			COALESCE(array_agg(DISTINCT l.claimed_by) FILTER (WHERE l.claimed_by IS NOT NULL), '{}') AS claimed_by_ids,
			COALESCE(array_agg(l.id ORDER BY l.id), '{}') AS lead_ids
		FROM leads_filtered l
		GROUP BY l.player_id, l.game_id
		ORDER BY MAX(l.created_at) DESC
		LIMIT ? OFFSET ?
	`
	_ = fmt.Sprintf // 保留 import 占位(下文未直接用,但 fmt 是 stringer 需要)
	rows, err := r.DB.Raw(q, base, pageSize, (page-1)*pageSize).Rows()
	if err != nil {
		response.ServerError(c, "查询失败: "+err.Error())
		return
	}
	defer rows.Close()

	players := make([]PlayerSummary, 0)
	for rows.Next() {
		var ra rowAgg
		if err := rows.Scan(
			&ra.PlayerID, &ra.GameID, &ra.LatestLeadID, &ra.LatestUsername, &ra.LatestCountry,
			&ra.LatestIsTarget, &ra.LeadCount, &ra.FirstLeadAt, &ra.LastLeadAt, &ra.Extension,
			&ra.CreatorIDs, &ra.ClaimedByIDs, &ra.LeadIDs,
		); err != nil {
			response.ServerError(c, "扫描失败: "+err.Error())
			return
		}

		p := PlayerSummary{
			PlayerID:       ra.PlayerID,
			GameID:         ra.GameID,
			LatestUsername: ra.LatestUsername,
			LatestCountry:  ra.LatestCountry,
			LatestIsTarget: ra.LatestIsTarget,
			LeadCount:      ra.LeadCount,
			FirstLeadAt:    &ra.FirstLeadAt,
			LastLeadAt:     &ra.LastLeadAt,
		}
		if ra.CreatorIDs != "" {
			p.CreatorIDs = parsePgIntArray(ra.CreatorIDs)
		}
		if ra.ClaimedByIDs != "" {
			p.ClaimedByIDs = parsePgIntArray(ra.ClaimedByIDs)
		}
		if ra.LeadIDs != "" {
			p.LeadIDs = parsePgIntArray(ra.LeadIDs)
		}
		p.LatestLeadID = ra.LatestLeadID
		if len(ra.Extension) > 0 {
			extStr := strings.Trim(string(ra.Extension), `"`)
			extStr = strings.ReplaceAll(extStr, `\"`, `"`)
			_ = extStr
		}

		// 订单聚合:按 player_id + game_id
		var paidCount int64
		var totalCount int64
		var totalAmount float64
		r.DB.Model(&model.Order{}).
			Where("player_id = ? AND game_id = ?", ra.PlayerID, ra.GameID).
			Where("deleted_at IS NULL").
			Count(&totalCount)
		r.DB.Model(&model.Order{}).
			Where("player_id = ? AND game_id = ? AND status = ?", ra.PlayerID, ra.GameID, model.OrderStatusPaid).
			Where("deleted_at IS NULL").
			Count(&paidCount)
		r.DB.Model(&model.Order{}).
			Where("player_id = ? AND game_id = ? AND status = ?", ra.PlayerID, ra.GameID, model.OrderStatusPaid).
			Where("deleted_at IS NULL").
			Select("COALESCE(SUM(amount), 0)").
			Scan(&totalAmount)
		p.OrderCount = totalCount
		p.PaidOrderCount = paidCount
		p.TotalOrderAmount = totalAmount

		// 最后跟进时间(取该玩家所有 lead 中最后一次跟进)
		// SQL 层用 COALESCE 把 NULL 替换成 epoch,Go 层比较避免 nil deref
		var lastFollowAt *time.Time
		var lastFollowRaw time.Time
		tx := r.DB.Session(&gorm.Session{NewDB: true}).Model(&model.LeadFollowRecord{}).
			Joins("JOIN leads ON leads.id = lead_follow_records.lead_id").
			Where("leads.player_id = ? AND leads.game_id = ?", ra.PlayerID, ra.GameID).
			Where("lead_follow_records.deleted_at IS NULL").
			Select("COALESCE(MAX(lead_follow_records.created_at), '1970-01-01 00:00:00+00'::timestamptz)").
			Scan(&lastFollowRaw)
		if tx.Error == nil && !lastFollowRaw.Equal(time.Unix(0, 0).UTC()) {
			t := lastFollowRaw
			lastFollowAt = &t
		}
		p.LastFollowAt = lastFollowAt

		// Game 信息(代码 + 名称)—— 用 NewDB session 避免与 rows 迭代共用连接
		// Unscoped:即使游戏被软删,玩家页也要能看到(避免空指针/字段缺失)
		var game model.Game
		if err := r.DB.Session(&gorm.Session{NewDB: true}).Unscoped().First(&game, ra.GameID).Error; err == nil {
			p.GameCode = game.Code
			p.GameName = game.Name
		}

		players = append(players, p)
	}

	response.OK(c, gin.H{
		"list":      players,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// playerDetail 玩家详情(GET /api/v1/players/:player_id/:game_id)
//
// 包含头部聚合 + leads 时间线 + orders + follow_records
func (r *Router) playerDetail(c *gin.Context) {
	playerID := c.Param("player_id")
	gameIDStr := c.Param("game_id")
	if playerID == "" || gameIDStr == "" {
		response.FailWithStatus(c, http.StatusBadRequest, 400, "缺少 player_id / game_id")
		return
	}
	gid, err := strconv.ParseUint(gameIDStr, 10, 64)
	if err != nil {
		response.FailWithStatus(c, http.StatusBadRequest, 400, "game_id 格式错误")
		return
	}

	// 先查至少一条 lead(用于权限检查)
	var oneLead model.Lead
	leadQ := r.DB.Where("player_id = ? AND game_id = ?", playerID, uint(gid))
	if err := leadQ.First(&oneLead).Error; err != nil {
		response.NotFound(c, "玩家不存在")
		return
	}
	if !canAccessLead(c, &oneLead) {
		response.Forbidden(c, "无权查看该玩家")
		return
	}

	// 所有 leads(按时间倒序)
	var leads []model.Lead
	r.DB.Where("player_id = ? AND game_id = ?", playerID, uint(gid)).
		Preload("Game").Preload("Creator").Preload("ClaimedByUser").
		Order("created_at DESC").
		Find(&leads)

	// 所有 orders
	var orders []model.Order
	r.DB.Where("player_id = ? AND game_id = ?", playerID, uint(gid)).
		Where("deleted_at IS NULL").
		Preload("Game").Preload("Lead").
		Order("order_time DESC").
		Find(&orders)

	// 所有 follow_records(通过 leads 关联)
	leadIDs := make([]uint, len(leads))
	for i, l := range leads {
		leadIDs[i] = l.ID
	}
	var followRecords []model.LeadFollowRecord
	if len(leadIDs) > 0 {
		r.DB.Where("lead_id IN ?", leadIDs).
			Preload("Follower").
			Order("created_at DESC").
			Find(&followRecords)
	}

	// 头部聚合
	summary := PlayerSummary{
		PlayerID:       playerID,
		GameID:         uint(gid),
		LatestUsername: oneLead.PlayerUsername,
		LatestCountry:  oneLead.PlayerCountry,
		LatestIsTarget: string(oneLead.IsTarget),
		LeadCount:      int64(len(leads)),
		OrderCount:     int64(len(orders)),
	}
	if len(leads) > 0 {
		summary.FirstLeadAt = &leads[len(leads)-1].CreatedAt
		summary.LastLeadAt = &leads[0].CreatedAt
	}
	// paid 订单 / 总金额
	var paid int64
	var totalAmt float64
	for _, o := range orders {
		if o.Status == model.OrderStatusPaid {
			paid++
			totalAmt += o.Amount
		}
	}
	summary.PaidOrderCount = paid
	summary.TotalOrderAmount = totalAmt
	// creator / claimed 集合
	creatorSet := map[uint]bool{}
	claimedSet := map[uint]bool{}
	for _, l := range leads {
		creatorSet[l.CreatorID] = true
		if l.ClaimedBy != nil {
			claimedSet[*l.ClaimedBy] = true
		}
		summary.LeadIDs = append(summary.LeadIDs, l.ID)
	}
	// P13 强制编辑入口:leads 已按 created_at DESC 排序,索引 0 即最新
	if len(leads) > 0 {
		summary.LatestLeadID = leads[0].ID
	}
	for k := range creatorSet {
		summary.CreatorIDs = append(summary.CreatorIDs, k)
	}
	for k := range claimedSet {
		summary.ClaimedByIDs = append(summary.ClaimedByIDs, k)
	}
	// game info
	var game model.Game
	if err := r.DB.First(&game, uint(gid)).Error; err == nil {
		summary.GameCode = game.Code
		summary.GameName = game.Name
	}
	// 最后跟进
	if len(followRecords) > 0 {
		t := followRecords[0].CreatedAt
		summary.LastFollowAt = &t
	}

	response.OK(c, gin.H{
		"summary":         summary,
		"leads":           leads,
		"orders":          orders,
		"follow_records":  followRecords,
	})
}

// playerKPI 玩家 KPI 总览(GET /api/v1/players/kpi)
//
// 4 卡:玩家总数 / 目标玩家 / 有订单玩家 / 今日新增玩家
// 权限:同 playerList(走同一套 RBAC 过滤)
func (r *Router) playerKPI(c *gin.Context) {
	role := middleware.GetCurrentRole(c)
	userID := middleware.GetCurrentUserID(c)

	base := r.DB.Model(&model.Lead{})
	switch role {
	case model.RoleAdmin:
	case model.RoleAStaff, model.RoleBStaff, model.RoleALead, model.RoleBLead:
		base = base.Where("creator_id = ? OR claimed_by = ?", userID, userID)
	case model.RoleAManager, model.RoleBManager:
		ids, err := r.subordinateIDs(userID)
		if err != nil {
			response.ServerError(c, "权限查询失败: "+err.Error())
			return
		}
		base = base.Where("creator_id IN ? OR claimed_by IN ?", ids, ids)
	default:
		response.Forbidden(c, "无权访问")
		return
	}

	// 玩家总数
	var totalPlayers int64
	r.DB.Raw("SELECT COUNT(*) FROM (SELECT 1 FROM (?) AS l GROUP BY l.player_id, l.game_id) AS p", base).Scan(&totalPlayers)

	// 目标玩家数
	var targetPlayers int64
	targetBase := base.Session(&gorm.Session{}).Where("is_target = ?", model.IsTargetTrue)
	r.DB.Raw("SELECT COUNT(*) FROM (SELECT 1 FROM (?) AS l GROUP BY l.player_id, l.game_id) AS p", targetBase).Scan(&targetPlayers)

	// 有订单玩家数(基于 paid 订单)
	hasOrdersSub := r.DB.Model(&model.Order{}).
		Select("DISTINCT player_id, game_id").
		Where("status = ?", model.OrderStatusPaid).
		Where("deleted_at IS NULL")
	var withOrders int64
	withOrdersBase := base.Session(&gorm.Session{}).Where("(player_id, game_id) IN (?)", hasOrdersSub)
	r.DB.Raw("SELECT COUNT(*) FROM (SELECT 1 FROM (?) AS l GROUP BY l.player_id, l.game_id) AS p", withOrdersBase).Scan(&withOrders)

	// 今日新增玩家
	today := time.Now().Truncate(24 * time.Hour)
	var todayNew int64
	todayBase := base.Session(&gorm.Session{}).Where("created_at >= ?", today)
	r.DB.Raw("SELECT COUNT(*) FROM (SELECT 1 FROM (?) AS l GROUP BY l.player_id, l.game_id) AS p", todayBase).Scan(&todayNew)

	response.OK(c, gin.H{
		"total_players":   totalPlayers,
		"target_players":  targetPlayers,
		"with_orders":     withOrders,
		"today_new":       todayNew,
	})
}

// parsePgIntArray 解析 PostgreSQL int[] 字面量"{1,2,3}" → []uint
func parsePgIntArray(s string) []uint {
	s = strings.TrimSpace(s)
	if !strings.HasPrefix(s, "{") || !strings.HasSuffix(s, "}") {
		return nil
	}
	s = s[1 : len(s)-1]
	parts := strings.Split(s, ",")
	out := make([]uint, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if v, err := strconv.ParseUint(p, 10, 64); err == nil {
			out = append(out, uint(v))
		}
	}
	return out
}