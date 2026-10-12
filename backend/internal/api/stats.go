package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"crm-backend/internal/middleware"
	"crm-backend/internal/model"
	"crm-backend/pkg/response"
)

// ========== P6: 数据看板 / 报表接口 ==========

// 时间范围辅助
type TimeRange struct {
	Start time.Time
	End   time.Time
}

func parseTimeRange(c *gin.Context) TimeRange {
	now := time.Now()
	startStr := c.Query("start")
	endStr := c.Query("end")

	var start, end time.Time
	if startStr != "" {
		if t, err := time.Parse("2006-01-02", startStr); err == nil {
			start = t
		}
	}
	if endStr != "" {
		if t, err := time.Parse("2006-01-02", endStr); err == nil {
			end = t.Add(24 * time.Hour)
		}
	}

	if start.IsZero() {
		// 默认最近 30 天
		start = now.AddDate(0, 0, -30)
	}
	if end.IsZero() {
		end = now
	}
	return TimeRange{Start: start, End: end}
}

// ========== 1. 总览（4 个数字卡）==========

// overviewResp 总览响应
type overviewResp struct {
	LeadCount      int64 `json:"lead_count"`       // 总线索数
	PoolCount      int64 `json:"pool_count"`       // 公开池（待认领）
	ClaimedCount   int64 `json:"claimed_count"`    // 已认领
	ArchivedCount  int64 `json:"archived_count"`   // 已流失
	TodayCount     int64 `json:"today_count"`      // 今日新增
	WeekCount      int64 `json:"week_count"`       // 本周新增
	MonthCount     int64 `json:"month_count"`      // 本月新增
	ConversionRate float64 `json:"conversion_rate"` // 转化率（已认领 / 总数）
}

// statsOverview 看板总览
func (r *Router) statsOverview(c *gin.Context) {
	tr := parseTimeRange(c)
	resp := overviewResp{}

	// 总数
	r.DB.Model(&model.Lead{}).Where("created_at BETWEEN ? AND ?", tr.Start, tr.End).Count(&resp.LeadCount)
	r.DB.Model(&model.Lead{}).Where("status = ? AND created_at BETWEEN ? AND ?", model.LeadStatusPending, tr.Start, tr.End).Count(&resp.PoolCount)
	r.DB.Model(&model.Lead{}).Where("status = ? AND created_at BETWEEN ? AND ?", model.LeadStatusClaimed, tr.Start, tr.End).Count(&resp.ClaimedCount)
	r.DB.Model(&model.Lead{}).Where("status = ? AND created_at BETWEEN ? AND ?", model.LeadStatusArchived, tr.Start, tr.End).Count(&resp.ArchivedCount)

	// 今日 / 本周 / 本月
	now := time.Now()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	startOfWeek := startOfDay.AddDate(0, 0, -int(now.Weekday())+1) // 周一开始
	if startOfWeek.Weekday() == time.Sunday {
		startOfWeek = startOfWeek.AddDate(0, 0, -6)
	}
	startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())

	r.DB.Model(&model.Lead{}).Where("created_at >= ?", startOfDay).Count(&resp.TodayCount)
	r.DB.Model(&model.Lead{}).Where("created_at >= ?", startOfWeek).Count(&resp.WeekCount)
	r.DB.Model(&model.Lead{}).Where("created_at >= ?", startOfMonth).Count(&resp.MonthCount)

	// 转化率
	if resp.LeadCount > 0 {
		resp.ConversionRate = float64(resp.ClaimedCount) / float64(resp.LeadCount) * 100
	}

	response.OK(c, resp)
}

// ========== 2. 拉新趋势（按天）==========

type trendItem struct {
	Date  string `json:"date"`  // YYYY-MM-DD
	Count int64  `json:"count"`
}

// statsTrend 拉新趋势（按天）
func (r *Router) statsTrend(c *gin.Context) {
	tr := parseTimeRange(c)

	type Row struct {
		Date  time.Time
		Count int64
	}
	var rows []Row

	// 按天聚合
	err := r.DB.Model(&model.Lead{}).
		Select("DATE(created_at) AS date, COUNT(*) AS count").
		Where("created_at BETWEEN ? AND ?", tr.Start, tr.End).
		Group("DATE(created_at)").
		Order("date ASC").
		Scan(&rows).Error

	if err != nil {
		response.ServerError(c, "查询失败")
		return
	}

	result := make([]trendItem, 0, len(rows))
	for _, r := range rows {
		result = append(result, trendItem{
			Date:  r.Date.Format("2006-01-02"),
			Count: r.Count,
		})
	}

	response.OK(c, result)
}

// ========== 3. 按游戏分布 ==========

type byGameItem struct {
	GameID   uint    `json:"game_id"`
	Code     string  `json:"code"`
	Name     string  `json:"name"`
	Count    int64   `json:"count"`
	Percent  float64 `json:"percent"`
}

// statsByGame 按游戏分布
func (r *Router) statsByGame(c *gin.Context) {
	tr := parseTimeRange(c)

	type Row struct {
		GameID uint
		Code   string
		Name   string
		Count  int64
	}
	var rows []Row

	r.DB.Model(&model.Lead{}).
		Select("leads.game_id, games.code, games.name, COUNT(*) AS count").
		Joins("LEFT JOIN games ON games.id = leads.game_id").
		Where("leads.created_at BETWEEN ? AND ?", tr.Start, tr.End).
		Group("leads.game_id, games.code, games.name").
		Order("count DESC").
		Scan(&rows)

	// 计算总数（按部门过滤后）
	role := middleware.GetCurrentRole(c)
	userID := middleware.GetCurrentUserID(c)
	_ = userID

	total := int64(0)
	for _, r := range rows {
		total += r.Count
	}

	result := make([]byGameItem, 0, len(rows))
	for _, r := range rows {
		pct := 0.0
		if total > 0 {
			pct = float64(r.Count) / float64(total) * 100
		}
		result = append(result, byGameItem{
			GameID:  r.GameID,
			Code:    r.Code,
			Name:    r.Name,
			Count:   r.Count,
			Percent: pct,
		})
	}

	_ = role
	response.OK(c, result)
}

// ========== 4. 按渠道分布 ==========

type byChannelItem struct {
	Channel string  `json:"channel"`
	Count   int64   `json:"count"`
	Percent float64 `json:"percent"`
}

// statsByChannel 按渠道分布
func (r *Router) statsByChannel(c *gin.Context) {
	tr := parseTimeRange(c)

	type Row struct {
		Channel string
		Count   int64
	}
	var rows []Row

	r.DB.Model(&model.Lead{}).
		Select("channel, COUNT(*) AS count").
		Where("created_at BETWEEN ? AND ?", tr.Start, tr.End).
		Group("channel").
		Order("count DESC").
		Scan(&rows)

	total := int64(0)
	for _, r := range rows {
		total += r.Count
	}

	result := make([]byChannelItem, 0, len(rows))
	for _, r := range rows {
		pct := 0.0
		if total > 0 {
			pct = float64(r.Count) / float64(total) * 100
		}
		result = append(result, byChannelItem{
			Channel: r.Channel,
			Count:   r.Count,
			Percent: pct,
		})
	}

	response.OK(c, result)
}

// ========== 5. 按国家分布 ==========

type byCountryItem struct {
	Country string `json:"country"`
	Count   int64  `json:"count"`
	Percent float64 `json:"percent"`
}

// statsByCountry 按国家分布
func (r *Router) statsByCountry(c *gin.Context) {
	tr := parseTimeRange(c)

	type Row struct {
		Country string
		Count   int64
	}
	var rows []Row

	r.DB.Model(&model.Lead{}).
		Select("player_country AS country, COUNT(*) AS count").
		Where("created_at BETWEEN ? AND ?", tr.Start, tr.End).
		Group("player_country").
		Order("count DESC").
		Limit(20).
		Scan(&rows)

	total := int64(0)
	for _, r := range rows {
		total += r.Count
	}

	result := make([]byCountryItem, 0, len(rows))
	for _, r := range rows {
		pct := 0.0
		if total > 0 {
			pct = float64(r.Count) / float64(total) * 100
		}
		result = append(result, byCountryItem{
			Country: r.Country,
			Count:   r.Count,
			Percent: pct,
		})
	}

	response.OK(c, result)
}

// ========== 6. A→B 转化漏斗 ==========

// funnelResp 转化漏斗
type funnelResp struct {
	Created  int64 `json:"created"`   // 拉新总数
	Claimed  int64 `json:"claimed"`   // 已被 B 部认领
	Archived int64 `json:"archived"`  // 流失
	Active   int64 `json:"active"`    // 当前维护中（已认领未归档）

	// 转化率（百分比）
	ClaimRate  float64 `json:"claim_rate"`   // claimed / created
	ActiveRate float64 `json:"active_rate"`  // active / created
	LossRate   float64 `json:"loss_rate"`    // archived / created

	// 平均时长
	AvgClaimDuration float64 `json:"avg_claim_duration_seconds"` // 拉新到认领平均秒
	AvgClaimCount    int64   `json:"avg_claim_count"`             // 参与平均时长计算的样本数
}

// statsFunnel A→B 转化漏斗
func (r *Router) statsFunnel(c *gin.Context) {
	tr := parseTimeRange(c)
	resp := funnelResp{}

	r.DB.Model(&model.Lead{}).Where("created_at BETWEEN ? AND ?", tr.Start, tr.End).Count(&resp.Created)
	r.DB.Model(&model.Lead{}).Where("status = ? AND created_at BETWEEN ? AND ?", model.LeadStatusClaimed, tr.Start, tr.End).Count(&resp.Claimed)
	r.DB.Model(&model.Lead{}).Where("status = ? AND created_at BETWEEN ? AND ?", model.LeadStatusArchived, tr.Start, tr.End).Count(&resp.Archived)
	r.DB.Model(&model.Lead{}).Where("status = ? AND created_at BETWEEN ? AND ?", model.LeadStatusClaimed, tr.Start, tr.End).Count(&resp.Active)

	if resp.Created > 0 {
		resp.ClaimRate = float64(resp.Claimed) / float64(resp.Created) * 100
		resp.ActiveRate = float64(resp.Active) / float64(resp.Created) * 100
		resp.LossRate = float64(resp.Archived) / float64(resp.Created) * 100
	}

	// 平均认领时长（拉新到认领的秒数）
	type durRow struct {
		AvgDur float64
		Count  int64
	}
	var dr durRow
	r.DB.Model(&model.Lead{}).
		Select("AVG(EXTRACT(EPOCH FROM (claimed_at - created_at))) AS avg_dur, COUNT(*) AS count").
		Where("status IN ? AND created_at BETWEEN ? AND ?", []model.LeadStatus{model.LeadStatusClaimed, model.LeadStatusArchived}, tr.Start, tr.End).
		Where("claimed_at IS NOT NULL").
		Scan(&dr)
	resp.AvgClaimDuration = dr.AvgDur
	resp.AvgClaimCount = dr.Count

	response.OK(c, resp)
}

// ========== 7. A 员工业绩排行（拉新数）==========

type staffItem struct {
	UserID    uint    `json:"user_id"`
	RealName  string  `json:"real_name"`
	Username  string  `json:"username"`
	Role      string  `json:"role"`
	DeptName  string  `json:"dept_name"`
	LeadCount int64   `json:"lead_count"`
}

// statsTopStaff 员工业绩排行（按拉新数）
func (r *Router) statsTopStaff(c *gin.Context) {
	limitStr := c.DefaultQuery("limit", "10")
	limit, _ := strconv.Atoi(limitStr)
	if limit <= 0 || limit > 100 {
		limit = 10
	}
	role := c.Query("role") // 可选：a_staff / b_staff 等过滤
	tr := parseTimeRange(c)

	type Row struct {
		UserID    uint
		RealName  string
		Username  string
		Role      string
		DeptName  string
		LeadCount int64
	}
	var rows []Row

	q := r.DB.Model(&model.Lead{}).
		Select("users.id AS user_id, users.real_name, users.username, users.role, departments.name AS dept_name, COUNT(DISTINCT leads.id) AS lead_count").
		Joins("INNER JOIN users ON users.id = leads.creator_id").
		Joins("LEFT JOIN departments ON departments.id = users.dept_id").
		Where("leads.created_at BETWEEN ? AND ?", tr.Start, tr.End)

	if role != "" {
		q = q.Where("users.role = ?", role)
	}

	q.Group("users.id, users.real_name, users.username, users.role, departments.name").
		Order("lead_count DESC").
		Limit(limit).
		Scan(&rows)

	result := make([]staffItem, 0, len(rows))
	for _, r := range rows {
		result = append(result, staffItem{
			UserID:    r.UserID,
			RealName:  r.RealName,
			Username:  r.Username,
			Role:      r.Role,
			DeptName:  r.DeptName,
			LeadCount: r.LeadCount,
		})
	}

	response.OK(c, result)
}

// ========== 8. B 员工业绩排行（认领数 / 维护玩家数）==========

type maintainerItem struct {
	UserID        uint   `json:"user_id"`
	RealName      string `json:"real_name"`
	Username      string `json:"username"`
	DeptName      string `json:"dept_name"`
	ActiveCount   int64  `json:"active_count"`   // 当前维护中
	TotalClaimed  int64  `json:"total_claimed"`  // 累计认领
}

// statsTopMaintainer B 部员工业绩排行
func (r *Router) statsTopMaintainer(c *gin.Context) {
	limitStr := c.DefaultQuery("limit", "10")
	limit, _ := strconv.Atoi(limitStr)
	if limit <= 0 || limit > 100 {
		limit = 10
	}

	type Row struct {
		UserID       uint
		RealName     string
		Username     string
		DeptName     string
		ActiveCount  int64
		TotalClaimed int64
	}
	var rows []Row

	r.DB.Model(&model.Lead{}).
		Select(`users.id AS user_id, users.real_name, users.username, departments.name AS dept_name,
			SUM(CASE WHEN leads.status = 'claimed' THEN 1 ELSE 0 END) AS active_count,
			COUNT(leads.id) AS total_claimed`).
		Joins("INNER JOIN users ON users.id = leads.claimed_by").
		Joins("LEFT JOIN departments ON departments.id = users.dept_id").
		Where("users.role IN ?", []model.Role{model.RoleBStaff, model.RoleBLead, model.RoleBManager}).
		Group("users.id, users.real_name, users.username, departments.name").
		Order("active_count DESC, total_claimed DESC").
		Limit(limit).
		Scan(&rows)

	result := make([]maintainerItem, 0, len(rows))
	for _, r := range rows {
		result = append(result, maintainerItem{
			UserID:       r.UserID,
			RealName:     r.RealName,
			Username:     r.Username,
			DeptName:     r.DeptName,
			ActiveCount:  r.ActiveCount,
			TotalClaimed: r.TotalClaimed,
		})
	}

	response.OK(c, result)
}

// ========== 9. 时段分布（按小时，看什么时候线索最多）==========

type hourlyItem struct {
	Hour  int   `json:"hour"`
	Count int64 `json:"count"`
}

// statsHourlyDistribution 拉新时段分布
func (r *Router) statsHourlyDistribution(c *gin.Context) {
	tr := parseTimeRange(c)

	type Row struct {
		Hour  int
		Count int64
	}
	var rows []Row

	r.DB.Model(&model.Lead{}).
		Select("EXTRACT(HOUR FROM created_at) AS hour, COUNT(*) AS count").
		Where("created_at BETWEEN ? AND ?", tr.Start, tr.End).
		Group("hour").
		Order("hour ASC").
		Scan(&rows)

	// 补全 0-23 小时
	hourMap := make(map[int]int64, 24)
	for _, r := range rows {
		hourMap[r.Hour] = r.Count
	}

	result := make([]hourlyItem, 0, 24)
	for h := 0; h < 24; h++ {
		result = append(result, hourlyItem{
			Hour:  h,
			Count: hourMap[h],
		})
	}

	response.OK(c, result)
}

// 占位：避免 unused 警告
var _ = http.StatusOK
var _ = gorm.ErrRecordNotFound