package model

import (
	"time"

	"gorm.io/gorm"
)

// DeptType 部门类型
type DeptType string

const (
	DeptTypeCompany DeptType = "company" // 公司（顶级）
	DeptTypeA       DeptType = "dept_a"  // A 部门
	DeptTypeB       DeptType = "dept_b"  // B 部门
	DeptTypeGroup   DeptType = "group"   // 部门下的组
)

// Department 部门/组
type Department struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	Name      string         `gorm:"size:50;not null" json:"name"`
	ParentID  *uint          `gorm:"index" json:"parent_id"` // 上级（null = 顶级）
	Type      DeptType       `gorm:"size:20;not null" json:"type"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// 关联
	Parent   *Department  `gorm:"foreignKey:ParentID" json:"parent,omitempty"`
	Children []Department `gorm:"foreignKey:ParentID" json:"children,omitempty"`
}

// TableName 表名
func (Department) TableName() string {
	return "departments"
}