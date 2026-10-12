package api

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"crm-backend/internal/middleware"
	"crm-backend/internal/model"
	"crm-backend/pkg/response"
)

// ============ P11 follow 端数据看板(只统计自己认领的 lead) ============
//
//   GET /api/v1/stats/follow/mine?days=30
//
//   范围:leads WHERE claimed_by = me
//   返回:4 卡 + 每日趋势

type followKPI struct {
	TotalClaimed   int64   `json:"total_claimed"`   // 总认领(非流失)
	TodayFollowUp  int64   `json:"today_follow_up"` // 今日跟进次数
	ActivePlayers  int64   `json:"active_players"`  // 待跟进(status=claimed 且今天没跟进)
	PaidPlayers    int64   `json:"paid_players"`    // 已付费(paid_mark=true)
	LostPlayers    int64   `json:"lost_players"`    // 流失(我跟进过且 status=archived)
	ConversionRate float64 `json:"conversion_rate"` // paid / total_claimed
}

type followTrendItem struct {
	Date     string `json:"date"`      // YYYY-MM-DD
	Claimed  int64  `json:"claimed"`   // 当日新增认领
	FollowUp int64  `json:"follow_up"` // 当日跟进次数
}

type followOverview struct {
	KPI   followKPI         `json:"kpi"`
	Trend []followTrendItem `json:"trend"`
	Days  int               `json:"days"`
}

// statsFollowMine 我的维护看板(只统计自己认领的 lead)
func (r *Router) statsFollowMine(c *gin.Context) {
	userID := middleware.GetCurrentUserID(c)

	days := 30
	if d := c.Query("days"); d != "" {
		if n, err := strconv.Atoi(d); err == nil && n > 0 && n <= 90 {
			days = n
		}
	}

	now := time.Now()
	startOfToday := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	// ============ KPI ============
	kpi := followKPI{}

	// 总认领(claimed_by = me 且非 archived)
	r.DB.Model(&model.Lead{}).
		Where("claimed_by = ? AND status <> ?", userID, model.LeadStatusArchived).
		Count(&kpi.TotalClaimed)

	// 今日跟进次数
	r.DB.Model(&model.LeadFollowRecord{}).
		Where("follower_id = ? AND created_at >= ?", userID, startOfToday).
		Count(&kpi.TodayFollowUp)

	// 待跟进(claimed_by = me && status = claimed && 今天没跟进)
	subQ := r.DB.Model(&model.LeadFollowRecord{}).
		Select("lead_id").
		Where("follower_id = ? AND created_at >= ?", userID, startOfToday)
	r.DB.Model(&model.Lead{}).
		Where("claimed_by = ? AND status = ? AND id NOT IN (?)", userID, model.LeadStatusClaimed, subQ).
		Count(&kpi.ActivePlayers)

	// 已付费(MVP 简化:仅 paid_mark=true。paid_auto 需要 join order 表,性能差,Phase 2 再做精确版)
	r.DB.Model(&model.Lead{}).
		Where("claimed_by = ? AND paid_mark = ?", userID, true).
		Count(&kpi.PaidPlayers)

	// 流失(我跟进过且 status=archived — 因为 archived 时 claimed_by 会被清掉)
	subQ2 := r.DB.Model(&model.LeadFollowRecord{}).
		Select("DISTINCT lead_id").
		Where("follower_id = ?", userID)
	r.DB.Model(&model.Lead{}).
		Where("status = ? AND id IN (?)", model.LeadStatusArchived, subQ2).
		Count(&kpi.LostPlayers)

	if kpi.TotalClaimed > 0 {
		kpi.ConversionRate = float64(kpi.PaidPlayers) / float64(kpi.TotalClaimed)
	}

	// ============ 趋势(每日认领 + 每日跟进) ============
	trend := make([]followTrendItem, 0, days)
	for i := days - 1; i >= 0; i-- {
		d := startOfToday.AddDate(0, 0, -i)
		next := d.AddDate(0, 0, 1)
		date := d.Format("2006-01-02")

		var claimedN, followN int64
		r.DB.Model(&model.Lead{}).
			Where("claimed_by = ? AND claimed_at >= ? AND claimed_at < ?", userID, d, next).
			Count(&claimedN)
		r.DB.Model(&model.LeadFollowRecord{}).
			Where("follower_id = ? AND created_at >= ? AND created_at < ?", userID, d, next).
			Count(&followN)

		trend = append(trend, followTrendItem{
			Date:     date,
			Claimed:  claimedN,
			FollowUp: followN,
		})
	}

	response.OK(c, followOverview{
		KPI:   kpi,
		Trend: trend,
		Days:  days,
	})
}
