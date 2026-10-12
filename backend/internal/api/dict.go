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

// CreateDictReq 创建字典
type CreateDictReq struct {
	Category model.DictCategory `json:"category" binding:"required"`
	Code     string             `json:"code" binding:"required,min=1,max=50"`
	Label    string             `json:"label" binding:"required,min=1,max=100"`
	Sort     int                `json:"sort"`
	Status   model.DictStatus   `json:"status"`
	Remark   string             `json:"remark"`
}

// UpdateDictReq 更新字典
type UpdateDictReq struct {
	Label  *string            `json:"label"`
	Sort   *int               `json:"sort"`
	Status *model.DictStatus  `json:"status"`
	Remark *string            `json:"remark"`
}

// dictList 字典列表（按 category 过滤）
func (r *Router) dictList(c *gin.Context) {
	category := c.Query("category")

	q := r.DB.Model(&model.Dict{})
	if category != "" {
		q = q.Where("category = ?", category)
	}

	var dicts []model.Dict
	if err := q.Order("category ASC, sort ASC, id ASC").Find(&dicts).Error; err != nil {
		response.ServerError(c, "查询失败")
		return
	}

	response.OK(c, dicts)
}

// dictCreate 创建字典
func (r *Router) dictCreate(c *gin.Context) {
	var req CreateDictReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithStatus(c, http.StatusBadRequest, 400, "参数错误: "+err.Error())
		return
	}

	// 校验 category
	switch req.Category {
	case model.DictCategoryChannel:
	default:
		response.Fail(c, 1, "字典分类非法")
		return
	}

	// 检查 (category, code) 唯一
	var existing model.Dict
	if err := r.DB.Where("category = ? AND code = ?", req.Category, req.Code).First(&existing).Error; err == nil {
		response.Fail(c, 1, "该分类下 code 已存在")
		return
	}

	status := req.Status
	if status == "" {
		status = model.DictStatusEnabled
	}

	dict := model.Dict{
		Category: req.Category,
		Code:     req.Code,
		Label:    req.Label,
		Sort:     req.Sort,
		Status:   status,
		Remark:   req.Remark,
	}
	if err := r.DB.Create(&dict).Error; err != nil {
		response.ServerError(c, "创建失败")
		return
	}

	response.OK(c, dict)
}

// dictGet 字典详情
func (r *Router) dictGet(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.FailWithStatus(c, http.StatusBadRequest, 400, "ID 格式错误")
		return
	}

	var dict model.Dict
	if err := r.DB.First(&dict, uint(id)).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.NotFound(c, "字典不存在")
			return
		}
		response.ServerError(c, "查询失败")
		return
	}

	response.OK(c, dict)
}

// dictUpdate 更新字典
func (r *Router) dictUpdate(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.FailWithStatus(c, http.StatusBadRequest, 400, "ID 格式错误")
		return
	}

	var req UpdateDictReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithStatus(c, http.StatusBadRequest, 400, "参数错误: "+err.Error())
		return
	}

	var dict model.Dict
	if err := r.DB.First(&dict, uint(id)).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.NotFound(c, "字典不存在")
			return
		}
		response.ServerError(c, "查询失败")
		return
	}

	updates := map[string]interface{}{}
	if req.Label != nil {
		updates["label"] = *req.Label
	}
	if req.Sort != nil {
		updates["sort"] = *req.Sort
	}
	if req.Status != nil {
		updates["status"] = *req.Status
	}
	if req.Remark != nil {
		updates["remark"] = *req.Remark
	}

	if len(updates) == 0 {
		response.Fail(c, 1, "无更新字段")
		return
	}

	if err := r.DB.Model(&dict).Updates(updates).Error; err != nil {
		response.ServerError(c, "更新失败")
		return
	}

	r.DB.First(&dict, dict.ID)
	response.OK(c, dict)
}

// dictDelete 删除字典
func (r *Router) dictDelete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.FailWithStatus(c, http.StatusBadRequest, 400, "ID 格式错误")
		return
	}

	if err := r.DB.Delete(&model.Dict{}, uint(id)).Error; err != nil {
		response.ServerError(c, "删除失败")
		return
	}

	response.OK(c, nil)
}

// dictCategories 获取所有字典分类（前端下拉用）
func (r *Router) dictCategories(c *gin.Context) {
	categories := []gin.H{
		{"value": "channel", "label": "来源渠道"},
	}
	response.OK(c, categories)
}