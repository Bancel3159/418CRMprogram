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

// CreateExtensionFieldReq 创建扩展字段
type CreateExtensionFieldReq struct {
	Code         string                 `json:"code" binding:"required,min=1,max=50"`
	Label        string                 `json:"label" binding:"required,min=1,max=100"`
	Type         string                 `json:"type" binding:"required"`
	Required     *bool                  `json:"required"`
	Enabled      *bool                  `json:"enabled"`
	Sort         int                    `json:"sort"`
	Options      map[string]interface{} `json:"options"`
	DictCategory string                 `json:"dict_category"` // select 类型时引用字典分类
	Placeholder  string                 `json:"placeholder"`
	Remark       string                 `json:"remark"`
}

// UpdateExtensionFieldReq 更新扩展字段
type UpdateExtensionFieldReq struct {
	Label        *string                 `json:"label"`
	Type         *string                 `json:"type"`
	Required     *bool                   `json:"required"`
	Enabled      *bool                   `json:"enabled"`
	Sort         *int                    `json:"sort"`
	Options      *map[string]interface{} `json:"options"`
	DictCategory *string                 `json:"dict_category"` // 空字符串 = 清除引用
	Placeholder  *string                 `json:"placeholder"`
	Remark       *string                 `json:"remark"`
}

// extensionFieldList 列表(任何登录用户都可查,用来渲染表单)
// 返回时:type=select 且设置了 dict_category 的字段,会附带 options 来自字典表
func (r *Router) extensionFieldList(c *gin.Context) {
	q := r.DB.Model(&model.LeadExtensionField{}).
		Where("enabled = ?", true).
		Order("sort ASC, id ASC")
	var fields []model.LeadExtensionField
	if err := q.Find(&fields).Error; err != nil {
		response.ServerError(c, "查询失败")
		return
	}

	// 收集所有引用了 dict_category 的分类,一次性拉字典数据
	type dictItem struct {
		Code  string `json:"code"`
		Label string `json:"label"`
	}
	dictCache := map[string][]dictItem{}
	for _, f := range fields {
		if f.Type != model.FieldTypeSelect || f.DictCategory == "" {
			continue
		}
		if _, ok := dictCache[f.DictCategory]; ok {
			continue
		}
		var rows []model.Dict
		if err := r.DB.Where("category = ? AND status = ?", f.DictCategory, model.DictStatusEnabled).
			Order("sort ASC, id ASC").
			Find(&rows).Error; err != nil {
			continue
		}
		opts := make([]dictItem, 0, len(rows))
		for _, d := range rows {
			opts = append(opts, dictItem{Code: d.Code, Label: d.Label})
		}
		dictCache[f.DictCategory] = opts
	}

	// 把字典选项灌回 fields 的 options(JSONB 格式:[{"label":"...","value":"..."}])
	type selectOption struct {
		Label string `json:"label"`
		Value string `json:"value"`
	}
	type fieldView struct {
		model.LeadExtensionField
		Options []selectOption `json:"options,omitempty"`
	}
	out := make([]fieldView, 0, len(fields))
	for _, f := range fields {
		v := fieldView{LeadExtensionField: f}
		if f.Type == model.FieldTypeSelect && f.DictCategory != "" {
			for _, di := range dictCache[f.DictCategory] {
				v.Options = append(v.Options, selectOption{Label: di.Label, Value: di.Code})
			}
		}
		out = append(out, v)
	}
	response.OK(c, out)
}

// extensionFieldAdminList admin 全量列表(含 disabled)
func (r *Router) extensionFieldAdminList(c *gin.Context) {
	var fields []model.LeadExtensionField
	if err := r.DB.Order("sort ASC, id ASC").Find(&fields).Error; err != nil {
		response.ServerError(c, "查询失败")
		return
	}
	response.OK(c, fields)
}

// extensionFieldCreate admin 创建
func (r *Router) extensionFieldCreate(c *gin.Context) {
	var req CreateExtensionFieldReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithStatus(c, http.StatusBadRequest, 400, "参数错误: "+err.Error())
		return
	}
	if !model.FieldType(req.Type).IsValid() {
		response.Fail(c, 1, "type 必须为 text/textarea/number/select/date")
		return
	}
	// 校验 dict_category(只有 select 才允许设置)
	if req.DictCategory != "" {
		if model.FieldType(req.Type) != model.FieldTypeSelect {
			response.Fail(c, 1, "只有 select 类型才能引用字典分类")
			return
		}
		switch model.DictCategory(req.DictCategory) {
		case model.DictCategoryChannel:
		default:
			response.Fail(c, 1, "dict_category 非法(目前只支持 channel)")
			return
		}
	}
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	required := false
	if req.Required != nil {
		required = *req.Required
	}

	// code 唯一
	var exist model.LeadExtensionField
	if err := r.DB.Where("code = ?", req.Code).First(&exist).Error; err == nil {
		response.Fail(c, 1, "code 已存在")
		return
	}

	userID, _ := c.Get("userID")
	uid, _ := userID.(uint)

	field := model.LeadExtensionField{
		Code:         req.Code,
		Label:        req.Label,
		Type:         model.FieldType(req.Type),
		Required:     required,
		Enabled:      enabled,
		Sort:         req.Sort,
		Options:      model.JSONB(req.Options),
		DictCategory: req.DictCategory,
		Placeholder:  req.Placeholder,
		Remark:       req.Remark,
		CreatedBy:    &uid,
	}
	if err := r.DB.Create(&field).Error; err != nil {
		response.ServerError(c, "创建失败")
		return
	}
	r.DB.First(&field, field.ID)
	response.OK(c, field)
}

// extensionFieldUpdate admin 更新
func (r *Router) extensionFieldUpdate(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.FailWithStatus(c, http.StatusBadRequest, 400, "ID 格式错误")
		return
	}
	var req UpdateExtensionFieldReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithStatus(c, http.StatusBadRequest, 400, "参数错误: "+err.Error())
		return
	}
	var field model.LeadExtensionField
	if err := r.DB.First(&field, uint(id)).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.NotFound(c, "字段不存在")
			return
		}
		response.ServerError(c, "查询失败")
		return
	}
	updates := map[string]interface{}{}
	if req.Label != nil {
		updates["label"] = *req.Label
	}
	if req.Type != nil {
		if !model.FieldType(*req.Type).IsValid() {
			response.Fail(c, 1, "type 非法")
			return
		}
		updates["type"] = *req.Type
	}
	if req.Required != nil {
		updates["required"] = *req.Required
	}
	if req.Enabled != nil {
		updates["enabled"] = *req.Enabled
	}
	if req.Sort != nil {
		updates["sort"] = *req.Sort
	}
	if req.Options != nil {
		updates["options"] = model.JSONB(*req.Options)
	}
	if req.DictCategory != nil {
		// 空字符串视为清除
		updates["dict_category"] = *req.DictCategory
	}
	if req.Placeholder != nil {
		updates["placeholder"] = *req.Placeholder
	}
	if req.Remark != nil {
		updates["remark"] = *req.Remark
	}
	if len(updates) == 0 {
		response.Fail(c, 1, "无更新字段")
		return
	}
	if err := r.DB.Model(&field).Updates(updates).Error; err != nil {
		response.ServerError(c, "更新失败")
		return
	}
	r.DB.First(&field, field.ID)
	response.OK(c, field)
}

// extensionFieldDelete admin 删除
func (r *Router) extensionFieldDelete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.FailWithStatus(c, http.StatusBadRequest, 400, "ID 格式错误")
		return
	}
	if err := r.DB.Delete(&model.LeadExtensionField{}, uint(id)).Error; err != nil {
		response.ServerError(c, "删除失败")
		return
	}
	response.OK(c, nil)
}