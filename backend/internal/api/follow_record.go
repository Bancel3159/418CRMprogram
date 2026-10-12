package api

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"crm-backend/internal/middleware"
	"crm-backend/internal/model"
	"crm-backend/pkg/response"
)

// CreateFollowRecordReq 新增跟进记录
type CreateFollowRecordReq struct {
	FollowType string `json:"follow_type"` // 自由文本,默认 "text"
	Content    string `json:"content" binding:"required,min=1"`
}

// followRecordCreate 创建跟进记录(POST /api/v1/leads/:id/follow-records)
//
// 权限:
//   - 必须能访问该 lead(canAccessLead)
//   - 通常是认领人 / 创建者 / 主管 / admin 操作
func (r *Router) followRecordCreate(c *gin.Context) {
	leadID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.FailWithStatus(c, http.StatusBadRequest, 400, "lead ID 格式错误")
		return
	}

	var lead model.Lead
	if err := r.DB.First(&lead, uint(leadID)).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.NotFound(c, "线索不存在")
			return
		}
		response.ServerError(c, "查询失败")
		return
	}
	if !canAccessLead(c, &lead) {
		response.Forbidden(c, "无权限对该线索新增跟进")
		return
	}

	var req CreateFollowRecordReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithStatus(c, http.StatusBadRequest, 400, "参数错误: "+err.Error())
		return
	}
	if req.FollowType == "" {
		req.FollowType = "text"
	}

	followerID := middleware.GetCurrentUserID(c)

	rec := model.LeadFollowRecord{
		LeadID:     uint(leadID),
		FollowerID: &followerID,
		FollowType: req.FollowType,
		Content:    req.Content,
	}
	if err := r.DB.Create(&rec).Error; err != nil {
		response.ServerError(c, "创建失败: "+err.Error())
		return
	}

	r.DB.Preload("Lead").Preload("Follower").First(&rec, rec.ID)
	response.OK(c, rec)
}

// followRecordList 列出某条 lead 的所有跟进记录(GET /api/v1/leads/:id/follow-records)
//
// 默认按时间倒序。权限复用 canAccessLead。
func (r *Router) followRecordList(c *gin.Context) {
	leadID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.FailWithStatus(c, http.StatusBadRequest, 400, "lead ID 格式错误")
		return
	}

	var lead model.Lead
	if err := r.DB.First(&lead, uint(leadID)).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.NotFound(c, "线索不存在")
			return
		}
		response.ServerError(c, "查询失败")
		return
	}
	if !canAccessLead(c, &lead) {
		response.Forbidden(c, "无权限查看跟进记录")
		return
	}

	var records []model.LeadFollowRecord
	if err := r.DB.Where("lead_id = ?", uint(leadID)).
		Preload("Follower").
		Order("created_at DESC").
		Find(&records).Error; err != nil {
		response.ServerError(c, "查询失败")
		return
	}

	response.OK(c, gin.H{
		"list":  records,
		"total": len(records),
	})
}

// followRecordDelete 删除跟进记录(DELETE /api/v1/follow-records/:id)
//
// 权限:
//   - 仅 admin / 该记录的 follower / 主管可删
func (r *Router) followRecordDelete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.FailWithStatus(c, http.StatusBadRequest, 400, "ID 格式错误")
		return
	}

	var rec model.LeadFollowRecord
	if err := r.DB.First(&rec, uint(id)).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.NotFound(c, "跟进记录不存在")
			return
		}
		response.ServerError(c, "查询失败")
		return
	}

	role := middleware.GetCurrentRole(c)
	userID := middleware.GetCurrentUserID(c)

	canDel := role == model.RoleAdmin ||
		role.IsManager() ||
		(rec.FollowerID != nil && *rec.FollowerID == userID)
	if !canDel {
		response.Forbidden(c, "无权删除该跟进记录")
		return
	}

	if err := r.DB.Delete(&rec).Error; err != nil {
		response.ServerError(c, "删除失败")
		return
	}
	response.OK(c, nil)
}