package api

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"crm-backend/internal/middleware"
	"crm-backend/internal/model"
	"crm-backend/pkg/response"
)

// ========== A 员工"我的拉新"数据看板(只看到数字,不暴露线索细节) ==========

// 个人漏斗状态:
//   - engaged  已对接 = B 部已认领的线索数
//   - paid     已付费 = 已对接 + (paid_mark OR 订单系统 paid 匹配)
//   - archived 已流失
//   - pending  待对接(历史遗留)
type personalStatus struct {
	Engaged  int64 `json:"engaged"`
	Paid     int64 `json:"paid"`
	Archived int64 `json:"archived"`
	Pending  int64 `json:"pending"`
}

// DetailedPaidBreakup 把 paid 分成两个来源
type DetailedPaidBreakup struct {
	ByMark  int64 `json:"by_mark"`
	ByOrder int64 `json:"by_order"`
	Both    int64 `json:"both"`
}

// PersonalRankItem 个人在 A 部门拉新员中的排名
type PersonalRankItem struct {
	Position    int   `json:"position"`
	TotalAStaff int   `json:"total_a_staff"`
	LeadCount   int64 `json:"lead_count"`
	Total       int64 `json:"total"`
}

// 标识当前数据看板看的是谁:
//   scope = "self"     → 本人
//   scope = "specific" → 主管/组长 选了某个 A 员工
//   scope = "all"      → 全体 A 员工聚合
type TargetInfo struct {
	Scope      string `json:"scope"`
	ScopeLabel string `json:"scope_label"` // "本人" / "晴天" / "全体 A 部 (4 人)"
	CreatorID  uint   `json:"creator_id,omitempty"`
	RealName   string `json:"real_name,omitempty"`
	Username   string `json:"username,omitempty"`
}

type personalOverview struct {
	Today      int64           `json:"today"`
	Yesterday  int64           `json:"yesterday"`
	Last7Days  int64           `json:"last_7_days"`
	Last30Days int64           `json:"last_30_days"`
	Total      int64           `json:"total"`

	Trend     []trendItem     `json:"trend"`
	ByChannel []byChannelItem `json:"by_channel"`
	ByCountry []byCountryItem `json:"by_country"`

	ByStatus       personalStatus      `json:"by_status"`
	PaidBreakup    DetailedPaidBreakup `json:"paid_breakup"`
	ConversionRate float64             `json:"conversion_rate"`
	Rank           PersonalRankItem    `json:"rank"`

	Target *TargetInfo `json:"target,omitempty"`

	// 全体/选中员工模式下:同时返回该用户信息(供前端展示头像 + 角色 + 部门)
	User *UserBrief `json:"user,omitempty"`

	// 全体模式下,有多少活跃拉新员(给"全体 4 人"文案用)
	TeamSize int `json:"team_size,omitempty"`
}

type UserBrief struct {
	ID       uint   `json:"id"`
	Username string `json:"username"`
	RealName string `json:"real_name"`
	Role     string `json:"role"`
	DeptName string `json:"dept_name,omitempty"`
}

type dayCount struct {
	Date  time.Time
	Count int64
}

// 内部辅助:对单个 creator 计算看板数字
//   creatorID == nil  → 全体 A 部聚合(不走 creator_id 过滤)
//   其他            → 该用户个人
// rank 字段在 creatorID == nil 时无意义,填充零值
func (r *Router) aggregateLeadStats(creatorID *uint, days int) (*personalOverview, error) {
	now := time.Now()
	startOfToday := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	startOfYesterday := startOfToday.AddDate(0, 0, -1)
	startOf7 := startOfToday.AddDate(0, 0, -6)
	startOf30 := startOfToday.AddDate(0, 0, -29)
	startTrend := startOfToday.AddDate(0, 0, -(days - 1))
	startRank := startOf30

	resp := &personalOverview{}

	// 工具:对当前 scope 做一次 COUNT
	countScope := func(extra string, args ...any) int64 {
		var n int64
		q := r.DB.Model(&model.Lead{})
		if creatorID != nil {
			q = q.Where("creator_id = ?", *creatorID)
		}
		if extra != "" {
			q = q.Where(extra, args...)
		}
		q.Count(&n)
		return n
	}

	// ----- 5 个数字卡 -----
	resp.Today = countScope("created_at >= ?", startOfToday)
	resp.Yesterday = countScope("created_at >= ? AND created_at < ?", startOfYesterday, startOfToday)
	resp.Last7Days = countScope("created_at >= ?", startOf7)
	resp.Last30Days = countScope("created_at >= ?", startOf30)
	resp.Total = countScope("")

	// ----- 状态分布 -----
	resp.ByStatus.Engaged = countScope("status = ?", model.LeadStatusClaimed)
	resp.ByStatus.Pending = countScope("status = ?", model.LeadStatusPending)
	resp.ByStatus.Archived = countScope("status = ?", model.LeadStatusArchived)

	// ----- 已付费(已对接 ∩ (paid_mark ∪ paid_order)) -----
	var paidPlayerIDs []string
	r.DB.Model(&model.Order{}).
		Distinct("player_id").
		Where("status = ?", model.OrderStatusPaid).
		Pluck("player_id", &paidPlayerIDs)

	scopeWhere := ""
	var scopeArgs []any
	if creatorID != nil {
		scopeWhere = "creator_id = ?"
		scopeArgs = []any{*creatorID}
	}

	mkCount := func(extra string, args ...any) int64 {
		var n int64
		q := r.DB.Model(&model.Lead{}).
			Where("status = ?", model.LeadStatusClaimed)
		if creatorID != nil {
			q = q.Where("creator_id = ?", *creatorID)
		}
		if extra != "" {
			q = q.Where(extra, args...)
		}
		q.Count(&n)
		return n
	}

	var autoPaid int64
	if len(paidPlayerIDs) > 0 {
		q := r.DB.Model(&model.Lead{}).Where("status = ?", model.LeadStatusClaimed)
		if creatorID != nil {
			q = q.Where("creator_id = ?", *creatorID)
		}
		autoPaid = mkCount("player_id IN ?", paidPlayerIDs)
		_ = q
	}

	var markPaid int64
	if creatorID != nil {
		markPaid = mkCount("paid_mark = ?", true)
	} else {
		markPaid = mkCount("paid_mark = ?", true)
	}

	// 总体 paid = claimed ∩ (paid_mark=true OR player_id IN paid_set)
	totalPaid := int64(0)
	if markPaid > 0 || len(paidPlayerIDs) > 0 {
		q := r.DB.Model(&model.Lead{}).Where("status = ?", model.LeadStatusClaimed)
		if creatorID != nil {
			q = q.Where("creator_id = ?", *creatorID)
		}
		switch {
		case markPaid > 0 && len(paidPlayerIDs) > 0:
			q = q.Where("paid_mark = ? OR player_id IN ?", true, paidPlayerIDs)
		case markPaid > 0:
			q = q.Where("paid_mark = ?", true)
		default:
			q = q.Where("player_id IN ?", paidPlayerIDs)
		}
		q.Count(&totalPaid)
	}
	resp.ByStatus.Paid = totalPaid

	// both 计数
	if markPaid > 0 && len(paidPlayerIDs) > 0 {
		both := int64(0)
		q := r.DB.Model(&model.Lead{}).
			Where("status = ? AND paid_mark = ?", model.LeadStatusClaimed, true)
		if creatorID != nil {
			q = q.Where("creator_id = ?", *creatorID)
		}
		q.Where("player_id IN ?", paidPlayerIDs).Count(&both)
		resp.PaidBreakup.Both = both
	}
	resp.PaidBreakup.ByMark = markPaid - resp.PaidBreakup.Both
	resp.PaidBreakup.ByOrder = autoPaid - resp.PaidBreakup.Both

	if resp.ByStatus.Engaged > 0 {
		resp.ConversionRate = float64(resp.ByStatus.Paid) / float64(resp.ByStatus.Engaged) * 100
	}

	// ----- 趋势(按天) -----
	var tRows []dayCount
	q := r.DB.Model(&model.Lead{}).
		Select("DATE(created_at) AS date, COUNT(*) AS count").
		Where("created_at >= ?", startTrend)
	if creatorID != nil {
		q = q.Where("creator_id = ?", *creatorID)
	}
	q.Group("DATE(created_at)").Order("date ASC").Scan(&tRows)
	resp.Trend = buildTrendWithGaps(tRows, days, startTrend)

	// ----- 渠道分布(30 天) -----
	type chRow struct {
		Channel string
		Count   int64
	}
	var chRows []chRow
	q = r.DB.Model(&model.Lead{}).
		Select("channel, COUNT(*) AS count").
		Where("created_at >= ?", startOf30).
		Where("channel <> ''")
	if creatorID != nil {
		q = q.Where("creator_id = ?", *creatorID)
	}
	q.Group("channel").Order("count DESC").Scan(&chRows)
	var chTotal int64
	for _, v := range chRows {
		chTotal += v.Count
	}
	for _, v := range chRows {
		pct := 0.0
		if chTotal > 0 {
			pct = float64(v.Count) / float64(chTotal) * 100
		}
		resp.ByChannel = append(resp.ByChannel, byChannelItem{
			Channel: v.Channel,
			Count:   v.Count,
			Percent: pct,
		})
	}

	// ----- 国家分布(30 天,top 10) -----
	type coRow struct {
		Country string
		Count   int64
	}
	var coRows []coRow
	q = r.DB.Model(&model.Lead{}).
		Select("player_country AS country, COUNT(*) AS count").
		Where("created_at >= ?", startOf30).
		Where("player_country <> ''")
	if creatorID != nil {
		q = q.Where("creator_id = ?", *creatorID)
	}
	q.Group("player_country").Order("count DESC").Limit(10).Scan(&coRows)
	var coTotal int64
	for _, v := range coRows {
		coTotal += v.Count
	}
	for _, v := range coRows {
		pct := 0.0
		if coTotal > 0 {
			pct = float64(v.Count) / float64(coTotal) * 100
		}
		resp.ByCountry = append(resp.ByCountry, byCountryItem{
			Country: v.Country,
			Count:   v.Count,
			Percent: pct,
		})
	}

	// ----- 排名 -----
	type rankRow struct {
		CreatorID uint
		Cnt       int64
	}
	var ranks []rankRow
	q = r.DB.Model(&model.Lead{}).
		Select("creator_id, COUNT(*) AS cnt").
		Where("created_at >= ?", startRank).
		Group("creator_id").
		Order("cnt DESC")
	q.Scan(&ranks)

	resp.Rank.TotalAStaff = len(ranks)
	for _, v := range ranks {
		resp.Rank.Total += v.Cnt
	}
	if creatorID != nil {
		for i, v := range ranks {
			if v.CreatorID == *creatorID {
				resp.Rank.Position = i + 1
				resp.Rank.LeadCount = v.Cnt
				break
			}
		}
	}

	resp.TeamSize = len(ranks)
	_ = scopeWhere
	_ = scopeArgs
	return resp, nil
}

// 工具:补齐 N 天趋势
func buildTrendWithGaps(rows []dayCount, days int, start time.Time) []trendItem {
	m := make(map[string]int64, days)
	for _, v := range rows {
		m[v.Date.Format("2006-01-02")] = v.Count
	}
	out := make([]trendItem, 0, days)
	for i := 0; i < days; i++ {
		d := start.AddDate(0, 0, i)
		key := d.Format("2006-01-02")
		out = append(out, trendItem{
			Date:  key,
			Count: m[key],
		})
	}
	return out
}

// subordinateIDs 取 actor 的"自己 + 直属下属"user id 列表(用于 scope=all 聚合)
// 直属下属 = superior_id == actor.id
// 注意:不含跨层(actor 下属的下属 不算进来)
func (r *Router) subordinateIDs(actorID uint) ([]uint, error) {
	var ids []uint
	err := r.DB.Model(&model.User{}).
		Where("id = ? OR superior_id = ?", actorID, actorID).
		Where("status = ?", model.UserStatusEnabled).
		Pluck("id", &ids).Error
	if err != nil {
		return nil, err
	}
	return ids, nil
}

// aggregateLeadStatsByCreators 对多个 creator_id 聚合看板数字(用于 scope=all)
//
// 与 aggregateLeadStats 的区别:后者只接单个 *uint,前者接 creator id 切片
// 不计算 rank(rank 是个人榜,聚合场景无意义)
func (r *Router) aggregateLeadStatsByCreators(creatorIDs []uint, days int) (*personalOverview, error) {
	now := time.Now()
	startOfToday := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	startOfYesterday := startOfToday.AddDate(0, 0, -1)
	startOf7 := startOfToday.AddDate(0, 0, -6)
	startOf30 := startOfToday.AddDate(0, 0, -29)
	startTrend := startOfToday.AddDate(0, 0, -(days - 1))

	resp := &personalOverview{}

	// 工具:对当前 scope 做一次 COUNT
	countScope := func(extra string, args ...any) int64 {
		var n int64
		q := r.DB.Model(&model.Lead{}).Where("creator_id IN ?", creatorIDs)
		if extra != "" {
			q = q.Where(extra, args...)
		}
		q.Count(&n)
		return n
	}

	// ----- 5 个数字卡 -----
	resp.Today = countScope("created_at >= ?", startOfToday)
	resp.Yesterday = countScope("created_at >= ? AND created_at < ?", startOfYesterday, startOfToday)
	resp.Last7Days = countScope("created_at >= ?", startOf7)
	resp.Last30Days = countScope("created_at >= ?", startOf30)
	resp.Total = countScope("")

	// ----- 状态分布 -----
	resp.ByStatus.Engaged = countScope("status = ?", model.LeadStatusClaimed)
	resp.ByStatus.Pending = countScope("status = ?", model.LeadStatusPending)
	resp.ByStatus.Archived = countScope("status = ?", model.LeadStatusArchived)

	// ----- 已付费 -----
	var paidPlayerIDs []string
	r.DB.Model(&model.Order{}).
		Distinct("player_id").
		Where("status = ?", model.OrderStatusPaid).
		Pluck("player_id", &paidPlayerIDs)

	var markPaid int64
	r.DB.Model(&model.Lead{}).
		Where("creator_id IN ? AND status = ? AND paid_mark = ?", creatorIDs, model.LeadStatusClaimed, true).
		Count(&markPaid)

	var autoPaid int64
	if len(paidPlayerIDs) > 0 {
		r.DB.Model(&model.Lead{}).
			Where("creator_id IN ? AND status = ? AND player_id IN ?", creatorIDs, model.LeadStatusClaimed, paidPlayerIDs).
			Count(&autoPaid)
	}

	var both int64
	if markPaid > 0 && len(paidPlayerIDs) > 0 {
		r.DB.Model(&model.Lead{}).
			Where("creator_id IN ? AND status = ? AND paid_mark = ? AND player_id IN ?",
				creatorIDs, model.LeadStatusClaimed, true, paidPlayerIDs).
			Count(&both)
	}

	totalPaid := markPaid + autoPaid - both
	resp.ByStatus.Paid = totalPaid
	resp.PaidBreakup.ByMark = markPaid - both
	resp.PaidBreakup.ByOrder = autoPaid - both
	resp.PaidBreakup.Both = both

	if resp.ByStatus.Engaged > 0 {
		resp.ConversionRate = float64(resp.ByStatus.Paid) / float64(resp.ByStatus.Engaged) * 100
	}

	// ----- 趋势 -----
	var tRows []dayCount
	r.DB.Model(&model.Lead{}).
		Select("DATE(created_at) AS date, COUNT(*) AS count").
		Where("creator_id IN ? AND created_at >= ?", creatorIDs, startTrend).
		Group("DATE(created_at)").Order("date ASC").Scan(&tRows)
	resp.Trend = buildTrendWithGaps(tRows, days, startTrend)

	// ----- 渠道分布 -----
	type chRow struct {
		Channel string
		Count   int64
	}
	var chRows []chRow
	r.DB.Model(&model.Lead{}).
		Select("channel, COUNT(*) AS count").
		Where("creator_id IN ? AND created_at >= ? AND channel <> ''", creatorIDs, startOf30).
		Group("channel").Order("count DESC").Scan(&chRows)
	var chTotal int64
	for _, v := range chRows {
		chTotal += v.Count
	}
	for _, v := range chRows {
		pct := 0.0
		if chTotal > 0 {
			pct = float64(v.Count) / float64(chTotal) * 100
		}
		resp.ByChannel = append(resp.ByChannel, byChannelItem{
			Channel: v.Channel,
			Count:   v.Count,
			Percent: pct,
		})
	}

	// ----- 国家分布 -----
	type coRow struct {
		Country string
		Count   int64
	}
	var coRows []coRow
	r.DB.Model(&model.Lead{}).
		Select("player_country AS country, COUNT(*) AS count").
		Where("creator_id IN ? AND created_at >= ? AND player_country <> ''", creatorIDs, startOf30).
		Group("player_country").Order("count DESC").Limit(10).Scan(&coRows)
	var coTotal int64
	for _, v := range coRows {
		coTotal += v.Count
	}
	for _, v := range coRows {
		pct := 0.0
		if coTotal > 0 {
			pct = float64(v.Count) / float64(coTotal) * 100
		}
		resp.ByCountry = append(resp.ByCountry, byCountryItem{
			Country: v.Country,
			Count:   v.Count,
			Percent: pct,
		})
	}

	// ----- 排名(聚合场景无个人排名,只填 TeamSize) -----
	resp.TeamSize = len(creatorIDs)
	return resp, nil
}

// statsMine 我的拉新看板(只看到自己)
func (r *Router) statsMine(c *gin.Context) {
	daysStr := c.DefaultQuery("days", "30")
	days, _ := strconv.Atoi(daysStr)
	if days <= 0 || days > 365 {
		days = 30
	}
	userID := middleware.GetCurrentUserID(c)
	resp, err := r.aggregateLeadStats(&userID, days)
	if err != nil {
		response.ServerError(c, "聚合失败: "+err.Error())
		return
	}
	// Target: 自己
	if u, err := r.fetchUserBrief(userID); err == nil {
		resp.User = u
	}
	resp.Target = &TargetInfo{
		Scope:      "self",
		ScopeLabel: "本人",
		CreatorID:  userID,
		RealName:   safeStr(resp.User, "real_name"),
		Username:   safeStr(resp.User, "username"),
	}
	response.OK(c, resp)
}

// statsTeam 主管/组长 看板:支持选人 / 全体
//
//   GET /api/v1/stats/team?days=30                  → 我的(同 /stats/mine)
//   GET /api/v1/stats/team?days=30&creator_id=<id> → 指定 A 员工个人(必须 actor 能看)
//   GET /api/v1/stats/team?days=30&scope=all       → 全体聚合(范围 = 自己 + 直属下属)
func (r *Router) statsTeam(c *gin.Context) {
	role := middleware.GetCurrentRole(c)
	userID := middleware.GetCurrentUserID(c)

	daysStr := c.DefaultQuery("days", "30")
	days, _ := strconv.Atoi(daysStr)
	if days <= 0 || days > 365 {
		days = 30
	}

	scope := c.Query("scope")                  // "all" 或空
	creatorQ := c.Query("creator_id")          // 数字或空
	var creatorID *uint

	switch {
	case scope == "all":
		// 全体聚合:仅 admin / 主管 / 组长 可看
		if !canViewTeamDashboard(role) {
			response.Forbidden(c, "仅 admin / 主管 / 组长 可看全体数据")
			return
		}
		// admin 看全体 A 部;主管/组长看"自己 + 直属下属"
		if role != model.RoleAdmin {
			ids, err := r.subordinateIDs(userID)
			if err != nil {
				response.ServerError(c, "查询下属失败: "+err.Error())
				return
			}
			// 即使下属列表为空,也允许请求成功(响应 team_size=0)
			resp := &personalOverview{
				Target: &TargetInfo{
					Scope:      "all",
					ScopeLabel: "我的下属",
				},
				TeamSize: len(ids),
			}
			if len(ids) > 0 {
				agg, err := r.aggregateLeadStatsByCreators(ids, days)
				if err != nil {
					response.ServerError(c, "聚合失败: "+err.Error())
					return
				}
				resp = agg
				resp.Target = &TargetInfo{
					Scope:      "all",
					ScopeLabel: "我的下属",
				}
				resp.TeamSize = len(ids)
			}
			response.OK(c, resp)
			return
		}
		creatorID = nil
	case creatorQ != "":
		// 指定某个员工
		v, err := strconv.ParseUint(creatorQ, 10, 64)
		if err != nil {
			response.FailWithStatus(c, 400, 400, "creator_id 格式错误")
			return
		}
		uid := uint(v)
		// 权限:自己看自己 / 或 admin/主管/组长 走上下级规则看他人
		if uid != userID && !canViewSpecificUser(userID, role, uid, r.DB) {
			response.Forbidden(c, "无权查看该用户的数据")
			return
		}
		creatorID = &uid
	default:
		// 无参 → 默认看自己
		creatorID = &userID
	}

	resp, err := r.aggregateLeadStats(creatorID, days)
	if err != nil {
		response.ServerError(c, "聚合失败: "+err.Error())
		return
	}

	// Target/User 注解
	if creatorID != nil {
		if u, err := r.fetchUserBrief(*creatorID); err == nil {
			resp.User = u
		}
		isSelf := *creatorID == userID
		resp.Target = &TargetInfo{
			Scope:      ternary(isSelf, "self", "specific"),
			ScopeLabel: ternary(isSelf, "本人", safeStr(resp.User, "real_name")+"("+safeStr(resp.User, "username")+")"),
			CreatorID:  *creatorID,
			RealName:   safeStr(resp.User, "real_name"),
			Username:   safeStr(resp.User, "username"),
		}
	} else {
		resp.Target = &TargetInfo{
			Scope:      "all",
			ScopeLabel: "全体 A 部",
		}
	}

	response.OK(c, resp)
}

// canViewTeamDashboard 是否能进入 team 看板(全员 / 他人聚合)
// 角色门槛:admin / 主管 / 组长
// 不检查具体能看哪些人(由 canViewSpecificUser 决定)
func canViewTeamDashboard(role model.Role) bool {
	switch role {
	case model.RoleAdmin, model.RoleAManager, model.RoleALead, model.RoleBManager, model.RoleBLead:
		return true
	}
	return false
}

// canViewSpecificUser actor 是否有权查看 targetID 的数据
// 规则:
//   - admin → 任何人
//   - 自己 → 永远可以
//   - 主管(a_manager/b_manager) → 同部门任意用户(DeptType 相同)
//   - 组长(a_lead/b_lead) → 直属下属(superior_id == actor.id)
//   - 员工 → 不可看他人
func canViewSpecificUser(actorID uint, actorRole model.Role, targetID uint, db *gorm.DB) bool {
	if actorRole == model.RoleAdmin {
		return true
	}
	if actorID == targetID {
		return true
	}
	var target model.User
	if err := db.First(&target, targetID).Error; err != nil {
		return false
	}
	switch actorRole {
	case model.RoleAManager, model.RoleBManager:
		// 同部门:actor 与 target 的 DeptType 相同即可
		return actorRole.DeptType() == target.Role.DeptType()
	case model.RoleALead, model.RoleBLead:
		// 直属下属
		return target.SuperiorID != nil && *target.SuperiorID == actorID
	}
	return false
}

// fetchUserBrief 取一个用户的简要信息
func (r *Router) fetchUserBrief(userID uint) (*UserBrief, error) {
	type row struct {
		ID        uint
		Username  string
		RealName  string
		Role      string
		DeptName  *string
	}
	var rr row
	err := r.DB.Model(&model.User{}).
		Select("users.id, users.username, users.real_name, users.role, departments.name AS dept_name").
		Joins("LEFT JOIN departments ON departments.id = users.dept_id").
		Where("users.id = ?", userID).
		Scan(&rr).Error
	if err != nil {
		return nil, err
	}
	out := &UserBrief{
		ID: rr.ID, Username: rr.Username, RealName: rr.RealName, Role: rr.Role,
	}
	if rr.DeptName != nil {
		out.DeptName = *rr.DeptName
	}
	return out, nil
}

func safeStr(p *UserBrief, key string) string {
	if p == nil {
		return ""
	}
	switch key {
	case "real_name":
		return p.RealName
	case "username":
		return p.Username
	case "role":
		return p.Role
	}
	return ""
}

func ternary[T any](cond bool, a, b T) T {
	if cond {
		return a
	}
	return b
}
