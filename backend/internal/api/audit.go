package api

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"crm-backend/internal/model"
	"crm-backend/pkg/response"
)

// auditList 操作日志列表(仅 admin)
func (r *Router) auditList(c *gin.Context) {
	var filter struct {
		UserID       uint   `form:"user_id"`
		Username     string `form:"username"`
		Action       string `form:"action"`
		TargetType   string `form:"target_type"`
		TargetID     uint   `form:"target_id"`
		Status       string `form:"status"`
		StartDate    string `form:"start_date"`
		EndDate      string `form:"end_date"`
		Page         int    `form:"page"`
		PageSize     int    `form:"page_size"`
	}
	if err := c.ShouldBindQuery(&filter); err != nil {
		response.FailWithStatus(c, http.StatusBadRequest, 400, "参数错误: "+err.Error())
		return
	}
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.PageSize < 1 || filter.PageSize > 200 {
		filter.PageSize = 20
	}

	q := r.DB.Model(&model.AuditLog{}).Order("id DESC")
	if filter.UserID > 0 {
		q = q.Where("user_id = ?", filter.UserID)
	}
	if filter.Username != "" {
		q = q.Where("username = ?", filter.Username)
	}
	if filter.Action != "" {
		q = q.Where("action = ?", filter.Action)
	}
	if filter.TargetType != "" {
		q = q.Where("target_type = ?", filter.TargetType)
	}
	if filter.TargetID > 0 {
		q = q.Where("target_id = ?", filter.TargetID)
	}
	if filter.Status != "" {
		q = q.Where("status = ?", filter.Status)
	}
	if filter.StartDate != "" {
		q = q.Where("created_at >= ?", filter.StartDate)
	}
	if filter.EndDate != "" {
		q = q.Where("created_at < ?", filter.EndDate+" 23:59:59")
	}

	var total int64
	q.Count(&total)

	var logs []model.AuditLog
	if err := q.Offset((filter.Page - 1) * filter.PageSize).Limit(filter.PageSize).Find(&logs).Error; err != nil {
		response.ServerError(c, "查询失败")
		return
	}

	response.OK(c, gin.H{
		"list":      logs,
		"total":     total,
		"page":      filter.Page,
		"page_size": filter.PageSize,
	})
}

// auditGet 操作日志详情
func (r *Router) auditGet(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.FailWithStatus(c, http.StatusBadRequest, 400, "ID 格式错误")
		return
	}
	var log model.AuditLog
	if err := r.DB.First(&log, uint(id)).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.NotFound(c, "日志不存在")
			return
		}
		response.ServerError(c, "查询失败")
		return
	}
	response.OK(c, log)
}

// auditStats 操作日志统计(近 7 天动作分布)
func (r *Router) auditStats(c *gin.Context) {
	type ActionCount struct {
		Action string `gorm:"column:action" json:"action"`
		Count  int64  `gorm:"column:count" json:"count"`
	}
	var rows []ActionCount
	r.DB.Model(&model.AuditLog{}).
		Select("action, COUNT(*) AS count").
		Where("created_at >= NOW() - INTERVAL '7 days'").
		Group("action").
		Order("count DESC").
		Scan(&rows)

	type UserCount struct {
		Username string `gorm:"column:username" json:"username"`
		Count    int64  `gorm:"column:count" json:"count"`
	}
	var rowsByUser []UserCount
	r.DB.Model(&model.AuditLog{}).
		Select("username, COUNT(*) AS count").
		Where("created_at >= NOW() - INTERVAL '7 days'").
		Group("username").
		Order("count DESC").
		Limit(10).
		Scan(&rowsByUser)

	response.OK(c, gin.H{
		"by_action": rows,
		"by_user":   rowsByUser,
	})
}

// 防止 strings 包未引用
var _ = strings.TrimSpace