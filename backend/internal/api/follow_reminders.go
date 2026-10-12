package api

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"crm-backend/internal/middleware"
	"crm-backend/internal/model"
	"crm-backend/internal/service"
	"crm-backend/pkg/response"
)

// ============ P11 follow 端 API ============

// ReminderItem 提醒项(联合 lead 信息)
type ReminderItem struct {
	ID         uint             `json:"id"`
	LeadID     uint             `json:"lead_id"`
	RemindType model.ReminderType `json:"remind_type"`
	Priority   model.ReminderPriority `json:"priority"`
	Status     model.ReminderStatus `json:"status"`
	Note       string           `json:"note"`
	RemindDate string           `json:"remind_date"`
	CreatedAt  string           `json:"created_at"`
	CompletedAt *string         `json:"completed_at,omitempty"`

	// 嵌套 lead 关键字段(避免前端再请求详情)
	Lead *LeadBrief `json:"lead,omitempty"`
}

// LeadBrief 提醒用 lead 精简信息
type LeadBrief struct {
	ID            uint   `json:"id"`
	GameCode      string `json:"game_code"`
	GameName      string `json:"game_name"`
	PlayerID      string `json:"player_id"`
	PlayerUsername string `json:"player_username"`
	PlayerGameName string `json:"player_game_name"`
	Region        string `json:"region"`
	Country       string `json:"player_country"`
	IsTarget      string `json:"is_target"`
	IsPaid        bool   `json:"is_paid"`
	IsLost        bool   `json:"is_lost"`
	LastFollowAt  *string `json:"last_follow_at,omitempty"`
	ClaimedBy     *uint   `json:"claimed_by,omitempty"`
	ClaimedByName string `json:"claimed_by_name,omitempty"`
}

// remindersToday 当前用户的「今日待办」(范围:claimed_by = me 或 creator_id = me)
// 排序:priority DESC, remind_date DESC, lead_id ASC
func (r *Router) remindersToday(c *gin.Context) {
	userID := middleware.GetCurrentUserID(c)
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "50"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 50
	}
	status := c.DefaultQuery("status", "pending") // pending / done / all
	statusQuery := r.DB.Model(&model.LeadReminder{}).
		Joins("JOIN leads ON leads.id = lead_reminders.lead_id").
		Where("leads.claimed_by = ? OR leads.creator_id = ?", userID, userID)
	if status == "pending" {
		statusQuery = statusQuery.Where("lead_reminders.status = ?", model.ReminderStatusPending)
	} else if status == "done" {
		statusQuery = statusQuery.Where("lead_reminders.status = ?", model.ReminderStatusDone)
	}

	var total int64
	if err := statusQuery.Count(&total).Error; err != nil {
		response.ServerError(c, "查询失败")
		return
	}

	var reminders []model.LeadReminder
	if err := statusQuery.
		Preload("Lead.Game").
		Preload("Lead.ClaimedByUser").
		Order("lead_reminders.priority DESC, lead_reminders.remind_date DESC, lead_reminders.lead_id ASC").
		Offset((page - 1) * pageSize).Limit(pageSize).
		Find(&reminders).Error; err != nil {
		response.ServerError(c, "查询失败")
		return
	}

	// 计算 is_paid
	paidMap := r.computePaidSet()

	items := make([]ReminderItem, 0, len(reminders))
	for _, rem := range reminders {
		items = append(items, reminderToItem(rem, paidMap))
	}

	response.OK(c, gin.H{
		"list":      items,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// reminderToItem model → API DTO
func reminderToItem(rem model.LeadReminder, paidMap map[string]bool) ReminderItem {
	item := ReminderItem{
		ID:         rem.ID,
		LeadID:     rem.LeadID,
		RemindType: rem.RemindType,
		Priority:   rem.Priority,
		Status:     rem.Status,
		Note:       rem.Note,
		RemindDate: rem.RemindDate.Format("2006-01-02"),
		CreatedAt:  rem.CreatedAt.Format(time.RFC3339),
	}
	if rem.CompletedAt != nil {
		s := rem.CompletedAt.Format(time.RFC3339)
		item.CompletedAt = &s
	}
	if rem.Lead != nil {
		brief := &LeadBrief{
			ID:             rem.Lead.ID,
			PlayerID:       rem.Lead.PlayerID,
			PlayerUsername: rem.Lead.PlayerUsername,
			PlayerGameName: rem.Lead.PlayerGameName,
			Region:         rem.Lead.Region,
			Country:        rem.Lead.PlayerCountry,
			IsTarget:       string(rem.Lead.IsTarget),
			IsPaid:         paidMap[rem.Lead.PlayerID] || rem.Lead.PaidMark,
			IsLost:         rem.Lead.IsLost,
			ClaimedBy:      rem.Lead.ClaimedBy,
		}
		if rem.Lead.Game != nil {
			brief.GameCode = rem.Lead.Game.Code
			brief.GameName = rem.Lead.Game.Name
		}
		if rem.Lead.ClaimedByUser != nil {
			brief.ClaimedByName = rem.Lead.ClaimedByUser.RealName
			if brief.ClaimedByName == "" {
				brief.ClaimedByName = rem.Lead.ClaimedByUser.Username
			}
		}
		if rem.Lead.LastFollowAt != nil {
			s := rem.Lead.LastFollowAt.Format(time.RFC3339)
			brief.LastFollowAt = &s
		}
		item.Lead = brief
	}
	return item
}

// computePaidSet 复用一下(后端其他地方也用到,简单内联实现)
func (r *Router) computePaidSet() map[string]bool {
	type row struct {
		PlayerID string
	}
	var rows []row
	r.DB.Model(&model.Order{}).
		Select("DISTINCT player_id").
		Where("status = ?", model.OrderStatusPaid).
		Scan(&rows)
	out := make(map[string]bool, len(rows))
	for _, r := range rows {
		out[r.PlayerID] = true
	}
	return out
}

// markLostReq 手动标记流失
type markLostReq struct {
	Reason string `json:"reason"` // 可选,默认 "manual"
}

// leadMarkLostManual 手动标记 lead 流失(主管/admin)
func (r *Router) leadMarkLostManual(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.FailWithStatus(c, 400, 400, "ID 格式错误")
		return
	}
	role := middleware.GetCurrentRole(c)
	if !role.IsManager() {
		response.Forbidden(c, "仅 admin / 主管可手动标记流失")
		return
	}
	engine := service.NewFollowReminderEngine(r.DB)
	if err := engine.ManualMarkLost(c.Request.Context(), uint(id)); err != nil {
		response.ServerError(c, "标记失败")
		return
	}
	response.OKWithMessage(c, "已标记流失", nil)
}

// leadMarkActiveManual 手动标记 lead 激活(用于"再次登录会激活"兜底)
func (r *Router) leadMarkActiveManual(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.FailWithStatus(c, 400, 400, "ID 格式错误")
		return
	}
	role := middleware.GetCurrentRole(c)
	if !role.IsManager() {
		response.Forbidden(c, "仅 admin / 主管可手动标记激活")
		return
	}
	engine := service.NewFollowReminderEngine(r.DB)
	if err := engine.ManualMarkActive(c.Request.Context(), uint(id)); err != nil {
		response.ServerError(c, "标记失败")
		return
	}
	response.OKWithMessage(c, "已标记激活", nil)
}

// followDailyScanTrigger 手动触发每日扫描(admin only,调试用)
func (r *Router) followDailyScanTrigger(c *gin.Context) {
	if middleware.GetCurrentRole(c) != model.RoleAdmin {
		response.Forbidden(c, "仅 admin 可触发")
		return
	}
	engine := service.NewFollowReminderEngine(r.DB)
	result, err := engine.DailyScan(c.Request.Context())
	if err != nil {
		response.ServerError(c, "扫描失败: "+err.Error())
		return
	}
	response.OKWithMessage(c, "扫描完成", result)
}
