package service

import (
	"context"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"crm-backend/internal/middleware"
	"crm-backend/internal/model"
)

// AuditRecorder 操作日志记录服务
type AuditRecorder struct {
	DB *gorm.DB
}

// NewAuditRecorder 创建记录器
func NewAuditRecorder(db *gorm.DB) *AuditRecorder {
	return &AuditRecorder{DB: db}
}

// Record 异步写入一条日志(非阻塞,失败仅记录不抛错)
func (a *AuditRecorder) Record(ctx context.Context, log *model.AuditLog) {
	if log == nil {
		return
	}
	if log.Status == "" {
		log.Status = model.AuditStatusSuccess
	}
	go func() {
		c := context.Background()
		if err := a.DB.WithContext(c).Create(log).Error; err != nil {
			// 审计失败不影响业务
		}
	}()
}

// RecordSync 同步写入
func (a *AuditRecorder) RecordSync(ctx context.Context, log *model.AuditLog) error {
	if log == nil {
		return nil
	}
	if log.Status == "" {
		log.Status = model.AuditStatusSuccess
	}
	return a.DB.WithContext(ctx).Create(log).Error
}

// RecordAction 从 gin context 提取用户信息并异步记录一条审计
func (a *AuditRecorder) RecordAction(c *gin.Context, action model.AuditAction, targetType model.AuditTargetType, targetID *uint, targetLabel string, before, after, ext model.JSONB) {
	log := a.Build(c, action, targetType, targetID, targetLabel, before, after, ext)
	a.Record(c.Request.Context(), log)
}

// RecordFailure 记录失败(同步)
func (a *AuditRecorder) RecordFailure(c *gin.Context, action model.AuditAction, targetType model.AuditTargetType, targetLabel string, errMsg string) {
	log := a.Build(c, action, targetType, nil, targetLabel, nil, nil, nil)
	log.Status = model.AuditStatusFailure
	log.ErrorMessage = errMsg
	a.RecordSync(c.Request.Context(), log)
}

// Build 构造日志对象(不写入)
func (a *AuditRecorder) Build(c *gin.Context, action model.AuditAction, targetType model.AuditTargetType, targetID *uint, targetLabel string, before, after, ext model.JSONB) *model.AuditLog {
	log := &model.AuditLog{
		Action:      action,
		TargetType:  targetType,
		TargetID:    targetID,
		TargetLabel: targetLabel,
		Before:      before,
		After:       after,
		Extension:   ext,
		Status:      model.AuditStatusSuccess,
		IP:          c.ClientIP(),
		UserAgent:   c.GetHeader("User-Agent"),
	}

	if v, ok := c.Get(middleware.CtxUserID); ok {
		if uid, ok := v.(uint); ok {
			log.UserID = &uid
		}
	}
	if v, ok := c.Get(middleware.CtxUsername); ok {
		if s, ok := v.(string); ok {
			log.Username = s
		}
	}
	if v, ok := c.Get(middleware.CtxRole); ok {
		if r, ok := v.(model.Role); ok {
			log.Role = string(r)
		} else if s, ok := v.(string); ok {
			log.Role = s
		}
	}
	return log
}

// RecordUnauthLogin 记录未登录态事件(如登录失败,未注入用户信息)
func (a *AuditRecorder) RecordUnauthLogin(c *gin.Context, success bool, username string, errMsg string) {
	log := &model.AuditLog{
		Action:       model.AuditLogin,
		TargetType:   model.AuditTargetSystem,
		TargetLabel:  username,
		Username:     username,
		IP:           c.ClientIP(),
		UserAgent:    c.GetHeader("User-Agent"),
		Status:       model.AuditStatusSuccess,
	}
	if !success {
		log.Action = model.AuditLoginFailed
		log.Status = model.AuditStatusFailure
		log.ErrorMessage = errMsg
	}
	a.RecordSync(context.Background(), log)
}