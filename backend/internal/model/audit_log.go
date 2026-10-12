package model

import (
	"time"

	"gorm.io/gorm"
)

// AuditAction 操作类型枚举
type AuditAction string

const (
	// 认证类
	AuditLogin         AuditAction = "login"          // 登录成功
	AuditLoginFailed   AuditAction = "login_failed"   // 登录失败
	AuditLogout        AuditAction = "logout"         // 登出
	AuditTokenRefresh  AuditAction = "token_refresh"  // 刷新 token

	// 业务动作(无 before/after,只记录触发)
	AuditLeadClaim     AuditAction = "lead_claim"     // B 员工认领 lead
	AuditLeadRelease   AuditAction = "lead_release"   // 主管释放 lead
	AuditLeadArchive   AuditAction = "lead_archive"   // 归档 lead
	AuditLeadPaid      AuditAction = "lead_paid_mark" // 管理员标记 lead 已付费
	AuditLeadUnpaid    AuditAction = "lead_paid_unmark" // 管理员取消已付费标记
	AuditOrderBind     AuditAction = "order_bind"     // 人工绑定订单

	// 写操作
	AuditCreate        AuditAction = "create"         // 通用创建
	AuditUpdate        AuditAction = "update"         // 通用更新
	AuditDelete        AuditAction = "delete"         // 通用删除

	// 导入类
	AuditOrderImport   AuditAction = "order_import"   // CSV/Excel 导入
	AuditOrderWebhook  AuditAction = "order_webhook"  // Webhook 入库
	AuditPartnerPull   AuditAction = "partner_pull"   // API 拉取

	// 安全类
	AuditPasswordReset AuditAction = "password_reset" // 重置密码
)

// AuditTargetType 操作对象类型
type AuditTargetType string

const (
	AuditTargetUser           AuditTargetType = "user"
	AuditTargetDepartment     AuditTargetType = "department"
	AuditTargetGame           AuditTargetType = "game"
	AuditTargetDict           AuditTargetType = "dict"
	AuditTargetBroadcastTpl   AuditTargetType = "broadcast_template"
	AuditTargetLead           AuditTargetType = "lead"
	AuditTargetOrder          AuditTargetType = "order"
	AuditTargetPartner        AuditTargetType = "partner"
	AuditTargetUploadTemplate AuditTargetType = "upload_template"
	AuditTargetSystem         AuditTargetType = "system"
)

// AuditStatus 状态
type AuditStatus string

const (
	AuditStatusSuccess AuditStatus = "success"
	AuditStatusFailure AuditStatus = "failure"
)

// AuditLog 操作日志
type AuditLog struct {
	ID           uint           `gorm:"primaryKey" json:"id"`
	UserID       *uint          `gorm:"index" json:"user_id"`                  // 操作人(登录失败时可能为 NULL)
	Username     string         `gorm:"size:50;index" json:"username"`          // 冗余(用户改名/删除后仍可见)
	Role         string         `gorm:"size:20" json:"role"`                   // 冗余
	Action       AuditAction    `gorm:"size:30;not null;index" json:"action"`  // 操作类型
	TargetType   AuditTargetType `gorm:"size:30;index" json:"target_type"`     // 对象类型
	TargetID     *uint          `gorm:"index" json:"target_id"`                // 对象 ID
	TargetLabel  string         `gorm:"size:200" json:"target_label"`          // 对象展示名(冗余,便于查看)

	// 变更内容
	Before       JSONB          `gorm:"type:jsonb" json:"before"`              // 变更前
	After        JSONB          `gorm:"type:jsonb" json:"after"`               // 变更后

	// 扩展字段(各 action 自定义,如 import 的 batch / webhook 的 partner)
	Extension    JSONB          `gorm:"type:jsonb" json:"extension"`

	// 上下文
	IP           string         `gorm:"size:50;index" json:"ip"`
	UserAgent    string         `gorm:"size:500" json:"user_agent"`
	Status       AuditStatus    `gorm:"size:10;not null;default:'success';index" json:"status"`
	ErrorMessage string         `gorm:"type:text" json:"error_message"`

	CreatedAt time.Time      `gorm:"index" json:"created_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName 表名
func (AuditLog) TableName() string {
	return "audit_logs"
}