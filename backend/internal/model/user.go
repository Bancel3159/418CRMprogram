package model

import (
	"time"

	"gorm.io/gorm"
)

// Role 角色枚举
type Role string

const (
	RoleAdmin     Role = "admin"     // 超级管理员
	RoleAManager  Role = "a_manager" // 前端主管
	RoleALead     Role = "a_lead"    // 前端组长
	RoleAStaff    Role = "a_staff"   // 前端员工
	RoleBManager  Role = "b_manager" // 后端主管
	RoleBLead     Role = "b_lead"    // 后端组长
	RoleBStaff    Role = "b_staff"   // 后端员工
)

// IsValid 校验角色是否合法
func (r Role) IsValid() bool {
	switch r {
	case RoleAdmin, RoleAManager, RoleALead, RoleAStaff,
		RoleBManager, RoleBLead, RoleBStaff:
		return true
	}
	return false
}

// IsManager 是否主管
func (r Role) IsManager() bool {
	return r == RoleAdmin || r == RoleAManager || r == RoleBManager
}

// IsLead 是否组长（含主管）
func (r Role) IsLead() bool {
	return r.IsManager() || r == RoleALead || r == RoleBLead
}

// DeptType 部门类型（A/B/组）
func (r Role) DeptType() DeptType {
	switch r {
	case RoleAManager, RoleALead, RoleAStaff:
		return DeptTypeA
	case RoleBManager, RoleBLead, RoleBStaff:
		return DeptTypeB
	default:
		return DeptTypeCompany
	}
}

// UserStatus 用户状态
type UserStatus string

const (
	UserStatusEnabled  UserStatus = "enabled"
	UserStatusDisabled UserStatus = "disabled"
)

// User 用户/员工
type User struct {
	ID           uint           `gorm:"primaryKey" json:"id"`
	Username     string         `gorm:"size:50;uniqueIndex;not null" json:"username"`
	PasswordHash string         `gorm:"size:255;not null" json:"-"`
	RealName     string         `gorm:"size:50" json:"real_name"`
	Role         Role           `gorm:"size:20;not null;index" json:"role"`
	DeptID       *uint          `gorm:"index" json:"dept_id"`     // 部门 ID（保留字段,JWT/WS 仍在用）
	GroupID      *uint          `gorm:"index" json:"group_id"`     // 组 ID（可选）
	SuperiorID   *uint          `gorm:"index" json="superior_id"` // 归属 / 上级（指向另一个用户的 ID）
	Status       UserStatus     `gorm:"size:20;not null;default:'enabled'" json:"status"`
	AcceptOrders bool           `gorm:"not null;default:true" json:"accept_orders"` // 是否接单（follow 端公开池广播接收开关）
	LastLoginAt  *time.Time     `json:"last_login_at"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`

	// 关联
	Dept     *Department `gorm:"foreignKey:DeptID" json:"dept,omitempty"`
	Group    *Department `gorm:"foreignKey:GroupID" json:"group,omitempty"`
	Superior *User       `gorm:"foreignKey:SuperiorID" json:"superior,omitempty"`
}

// TableName 表名
func (User) TableName() string {
	return "users"
}