package api

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"crm-backend/internal/middleware"
	"crm-backend/internal/model"
	"crm-backend/pkg/response"
)

// statsTeamMembers A 部成员(主管/组长 下拉筛选用)
// 返回:
//   - admin              → 所有 A 部角色用户
//   - 主管 / 组长       → 自己 + 直属下属
//   - 员工              → 403
func (r *Router) statsTeamMembers(c *gin.Context) {
	role := middleware.GetCurrentRole(c)
	userID := middleware.GetCurrentUserID(c)
	if !canViewTeamDashboard(role) {
		response.Forbidden(c, "仅 admin / 主管 / 组长 可看 A 部成员")
		return
	}

	// 限定可见 user id 列表
	var visibleIDs []uint
	if role == model.RoleAdmin {
		// 全体 A 部
		if err := r.DB.Model(&model.User{}).
			Where("role IN ?", []model.Role{model.RoleAStaff, model.RoleALead, model.RoleAManager}).
			Where("status = ?", model.UserStatusEnabled).
			Pluck("id", &visibleIDs).Error; err != nil {
			response.ServerError(c, "查询失败: "+err.Error())
			return
		}
	} else {
		// 自己 + 直属下属
		ids, err := r.subordinateIDs(userID)
		if err != nil {
			response.ServerError(c, "查询下属失败: "+err.Error())
			return
		}
		visibleIDs = ids
	}
	if len(visibleIDs) == 0 {
		response.OK(c, []any{})
		return
	}

	type teamMember struct {
		UserID    uint   `json:"user_id"`
		Username  string `json:"username"`
		RealName  string `json:"real_name"`
		Role      string `json:"role"`
		LeadCount int64  `json:"lead_count"`
	}

	var members []teamMember
	err := r.DB.Model(&model.User{}).
		Select("users.id AS user_id, users.username, users.real_name, users.role, COUNT(leads.id) AS lead_count").
		Joins("LEFT JOIN leads ON leads.creator_id = users.id").
		Where("users.id IN ?", visibleIDs).
		Group("users.id, users.username, users.real_name, users.role").
		Order("lead_count DESC").
		Scan(&members).Error
	if err != nil {
		response.ServerError(c, "查询失败: "+err.Error())
		return
	}

	// 支持 days 参数过滤 lead_count
	daysStr := c.DefaultQuery("days", "0")
	days, _ := strconv.Atoi(daysStr)
	if days > 0 && days <= 365 {
		// 重新按时间段聚合
		type row struct {
			UserID    uint
			LeadCount int64
		}
		var rows []row
		r.DB.Model(&model.Lead{}).
			Select("creator_id AS user_id, COUNT(*) AS lead_count").
			Where("created_at >= NOW() - make_interval(days => ?)", days).
			Group("creator_id").
			Scan(&rows)
		m := make(map[uint]int64, len(rows))
		for _, v := range rows {
			m[v.UserID] = v.LeadCount
		}
		for i := range members {
			if v, ok := m[members[i].UserID]; ok {
				members[i].LeadCount = v
			} else {
				members[i].LeadCount = 0
			}
		}
	}

	response.OK(c, members)
}
