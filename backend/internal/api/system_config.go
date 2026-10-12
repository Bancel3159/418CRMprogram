package api

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"crm-backend/internal/middleware"
	"crm-backend/internal/model"
	"crm-backend/pkg/response"
)

// ============ P11 系统配置 API(全局 follow 配置) ============

// UpdateFollowConfigReq 批量更新 follow 配置
type UpdateFollowConfigReq struct {
	NewRegisterDays    *int    `json:"new_register_days,omitempty"`    // ptr 区分未传
	PaidOfflineDays    *int    `json:"paid_offline_days,omitempty"`
	PaidOfflineMaxDays *int    `json:"paid_offline_max_days,omitempty"`
	DailyCron          *string `json:"daily_cron,omitempty"`
}

// systemConfigFollowGet 读 follow 配置(admin 写 / 任意登录用户读)
func (r *Router) systemConfigFollowGet(c *gin.Context) {
	var rows []model.SystemConfig
	if err := r.DB.Where("key IN ?", []string{
		model.FollowConfigNewRegisterDays,
		model.FollowConfigPaidOfflineDays,
		model.FollowConfigPaidOfflineMaxDays,
		model.FollowConfigDailyCron,
	}).Find(&rows).Error; err != nil {
		response.ServerError(c, "查询失败")
		return
	}
	cfg := make(map[string]string, len(rows))
	for _, row := range rows {
		cfg[row.Key] = row.Value
	}
	response.OK(c, cfg)
}

// systemConfigFollowUpdate 批量更新 follow 配置(admin only)
func (r *Router) systemConfigFollowUpdate(c *gin.Context) {
	if middleware.GetCurrentRole(c) != model.RoleAdmin {
		response.Forbidden(c, "仅 admin 可修改配置")
		return
	}
	var req UpdateFollowConfigReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithStatus(c, 400, 400, "参数错误: "+err.Error())
		return
	}
	userID := middleware.GetCurrentUserID(c)

	updates := []struct {
		key   string
		value *string
		desc  string
	}{
		{model.FollowConfigNewRegisterDays, intPtrToStringPtr(req.NewRegisterDays), "新注册跟进天数"},
		{model.FollowConfigPaidOfflineDays, intPtrToStringPtr(req.PaidOfflineDays), "付费玩家离线升级天数"},
		{model.FollowConfigPaidOfflineMaxDays, intPtrToStringPtr(req.PaidOfflineMaxDays), "付费玩家离线流失天数"},
		{model.FollowConfigDailyCron, req.DailyCron, "每日扫描 cron 表达式"},
	}

	for _, u := range updates {
		if u.value == nil {
			continue
		}
		if err := r.DB.Model(&model.SystemConfig{}).
			Where("key = ?", u.key).
			Updates(map[string]interface{}{
				"value":      *u.value,
				"updated_by": userID,
			}).Error; err != nil {
			response.ServerError(c, "更新失败: "+u.key)
			return
		}
	}
	response.OKWithMessage(c, "已更新", nil)
}

func intPtrToStringPtr(p *int) *string {
	if p == nil {
		return nil
	}
	s := strconv.Itoa(*p)
	return &s
}

// ============ P12 实时通知配置 API ============

// UpdateRealtimeConfigReq 批量更新实时通知配置
type UpdateRealtimeConfigReq struct {
	HighValueAmount *float64 `json:"high_value_amount,omitempty"`
	Enabled         *bool    `json:"enabled,omitempty"`
}

// systemConfigRealtimeGet 读实时通知配置(admin 写 / 任意登录用户读)
func (r *Router) systemConfigRealtimeGet(c *gin.Context) {
	var rows []model.SystemConfig
	if err := r.DB.Where("key IN ?", []string{
		model.RealtimeHighValueAmount,
		model.RealtimeEnabled,
	}).Find(&rows).Error; err != nil {
		response.ServerError(c, "查询失败")
		return
	}
	cfg := make(map[string]string, len(rows))
	for _, row := range rows {
		cfg[row.Key] = row.Value
	}
	response.OK(c, cfg)
}

// systemConfigRealtimeUpdate 批量更新实时通知配置(admin only)
func (r *Router) systemConfigRealtimeUpdate(c *gin.Context) {
	if middleware.GetCurrentRole(c) != model.RoleAdmin {
		response.Forbidden(c, "仅 admin 可修改配置")
		return
	}
	var req UpdateRealtimeConfigReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithStatus(c, 400, 400, "参数错误: "+err.Error())
		return
	}
	userID := middleware.GetCurrentUserID(c)

	type kv struct {
		key   string
		value *string
	}
	var updates []kv

	if req.HighValueAmount != nil {
		s := strconv.FormatFloat(*req.HighValueAmount, 'f', -1, 64)
		updates = append(updates, kv{model.RealtimeHighValueAmount, &s})
	}
	if req.Enabled != nil {
		s := "false"
		if *req.Enabled {
			s = "true"
		}
		updates = append(updates, kv{model.RealtimeEnabled, &s})
	}

	for _, u := range updates {
		if u.value == nil {
			continue
		}
		if err := r.DB.Model(&model.SystemConfig{}).
			Where("key = ?", u.key).
			Updates(map[string]interface{}{
				"value":      *u.value,
				"updated_by": userID,
			}).Error; err != nil {
			response.ServerError(c, "更新失败: "+u.key)
			return
		}
	}
	response.OKWithMessage(c, "已更新", nil)
}
