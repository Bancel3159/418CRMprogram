package api

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"crm-backend/internal/model"
	"crm-backend/pkg/response"
)

// CreateDepartmentReq 创建部门请求
type CreateDepartmentReq struct {
	Name     string          `json:"name" binding:"required,min=1,max=50"`
	ParentID *uint           `json:"parent_id"`
	Type     model.DeptType  `json:"type" binding:"required"`
}

// UpdateDepartmentReq 更新部门请求
type UpdateDepartmentReq struct {
	Name     *string         `json:"name"`
	ParentID *uint           `json:"parent_id"`
	Type     *model.DeptType `json:"type"`
}

// deptList 部门列表（树形）
func (r *Router) deptList(c *gin.Context) {
	var depts []model.Department
	if err := r.DB.Order("id ASC").Find(&depts).Error; err != nil {
		response.ServerError(c, "查询失败")
		return
	}
	response.OK(c, depts)
}

// deptCreate 创建部门
func (r *Router) deptCreate(c *gin.Context) {
	var req CreateDepartmentReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithStatus(c, http.StatusBadRequest, 400, "参数错误: "+err.Error())
		return
	}

	// 校验 type
	switch req.Type {
	case model.DeptTypeCompany, model.DeptTypeA, model.DeptTypeB, model.DeptTypeGroup:
	default:
		response.Fail(c, 1, "部门类型非法")
		return
	}

	dept := model.Department{
		Name:     req.Name,
		ParentID: req.ParentID,
		Type:     req.Type,
	}

	if err := r.DB.Create(&dept).Error; err != nil {
		response.ServerError(c, "创建失败: "+err.Error())
		return
	}

	response.OK(c, dept)
}

// deptGet 部门详情
func (r *Router) deptGet(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.FailWithStatus(c, http.StatusBadRequest, 400, "ID 格式错误")
		return
	}

	var dept model.Department
	if err := r.DB.First(&dept, uint(id)).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.NotFound(c, "部门不存在")
			return
		}
		response.ServerError(c, "查询失败")
		return
	}

	response.OK(c, dept)
}

// deptUpdate 更新部门
func (r *Router) deptUpdate(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.FailWithStatus(c, http.StatusBadRequest, 400, "ID 格式错误")
		return
	}

	var req UpdateDepartmentReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithStatus(c, http.StatusBadRequest, 400, "参数错误: "+err.Error())
		return
	}

	var dept model.Department
	if err := r.DB.First(&dept, uint(id)).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.NotFound(c, "部门不存在")
			return
		}
		response.ServerError(c, "查询失败")
		return
	}

	updates := map[string]interface{}{}
	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.ParentID != nil {
		updates["parent_id"] = *req.ParentID
	}
	if req.Type != nil {
		updates["type"] = *req.Type
	}

	if len(updates) == 0 {
		response.Fail(c, 1, "无更新字段")
		return
	}

	if err := r.DB.Model(&dept).Updates(updates).Error; err != nil {
		response.ServerError(c, "更新失败")
		return
	}

	r.DB.First(&dept, dept.ID)
	response.OK(c, dept)
}

// deptDelete 删除部门（软删除）
func (r *Router) deptDelete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.FailWithStatus(c, http.StatusBadRequest, 400, "ID 格式错误")
		return
	}

	// 检查是否有子部门
	var childCount int64
	r.DB.Model(&model.Department{}).Where("parent_id = ?", uint(id)).Count(&childCount)
	if childCount > 0 {
		response.Fail(c, 1, "该部门下还有子部门，无法删除")
		return
	}

	// 检查是否有关联用户
	var userCount int64
	r.DB.Model(&model.User{}).Where("dept_id = ? OR group_id = ?", uint(id), uint(id)).Count(&userCount)
	if userCount > 0 {
		response.Fail(c, 1, "该部门下还有员工，无法删除")
		return
	}

	if err := r.DB.Delete(&model.Department{}, uint(id)).Error; err != nil {
		response.ServerError(c, "删除失败")
		return
	}

	response.OK(c, nil)
}