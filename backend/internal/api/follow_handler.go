package api

import (
	"errors"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"crm-backend/internal/middleware"
	"crm-backend/internal/model"
	"crm-backend/internal/service"
	"crm-backend/pkg/response"
)

// CreateFollowReq 写跟进记录请求
type CreateFollowReq struct {
	FollowType string `json:"follow_type"` // text / call / online / wechat / ...
	Content    string `json:"content" binding:"required,min=1"` // 内容必填
}

// leadCreateFollow 写跟进记录(P11 跟进规则引擎用)
//   - 内容必填
//   - 自动 mark 今日对应 reminders 为 done
//   - 更新 lead.last_follow_at(运行时计算字段)+ lead.updated_at
func (r *Router) leadCreateFollow(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.FailWithStatus(c, 400, 400, "ID 格式错误")
		return
	}
	var req CreateFollowReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithStatus(c, 400, 400, "参数错误: "+err.Error())
		return
	}
	if req.Content == "" {
		response.Fail(c, 1, "跟进内容不能为空")
		return
	}

	// 权限:只能跟进自己可见的 lead
	var lead model.Lead
	if err := r.DB.First(&lead, uint(id)).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.NotFound(c, "线索不存在")
			return
		}
		response.ServerError(c, "查询失败")
		return
	}
	if !canAccessLead(c, &lead) {
		response.Forbidden(c, "无权限跟进该线索")
		return
	}

	followType := req.FollowType
	if followType == "" {
		followType = "text"
	}

	userID := middleware.GetCurrentUserID(c)
	now := time.Now()

	// 写跟进记录
	rec := &model.LeadFollowRecord{
		LeadID:     lead.ID,
		FollowerID: &userID,
		FollowType: followType,
		Content:    req.Content,
	}
	if err := r.DB.Create(rec).Error; err != nil {
		response.ServerError(c, "写入跟进失败")
		return
	}

	// 标记今日 reminder 完成(P11 规则引擎消费)
	engine := service.NewFollowReminderEngine(r.DB)
	_ = engine.CompleteReminder(c.Request.Context(), lead.ID, userID, now)

	// 返回 lead 详情(前端不用再请求)
	r.DB.Preload("Game").Preload("Creator").Preload("ClaimedByUser").First(&lead, lead.ID)
	r.fillPaidFlags(&lead)

	response.OKWithMessage(c, "跟进已记录", gin.H{
		"follow_record": rec,
		"lead":          lead,
	})
}
