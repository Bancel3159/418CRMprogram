package api

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"crm-backend/internal/middleware"
	"crm-backend/internal/model"
	"crm-backend/pkg/response"
)

// CreateListViewReq 创建列表视图
type CreateListViewReq struct {
	EntityType string                   `json:"entity_type" binding:"required"`
	Name       string                   `json:"name" binding:"required,min=1,max=50"`
	Fields     []map[string]interface{} `json:"fields"`
	IsDefault  bool                     `json:"is_default"`
}

// UpdateListViewReq 更新列表视图
type UpdateListViewReq struct {
	Name   *string                   `json:"name"`
	Fields *[]map[string]interface{} `json:"fields"`
}

// listViewList 视图列表（任何登录用户都可查）
//
//   GET /api/v1/list-views?entity_type=player[&is_default=true]
//
// 不传 entity_type → 返回所有 entity 的简略列表（供 admin 切换）
// is_default=true → 只返回该 entity 的默认视图完整 fields
func (r *Router) listViewList(c *gin.Context) {
	entityType := c.Query("entity_type")
	isDefault := c.Query("is_default") == "true"

	// 默认视图快捷查询
	if isDefault && entityType != "" {
		if !model.ListViewEntityType(entityType).IsValid() {
			response.Fail(c, 1, "entity_type 非法")
			return
		}
		var v model.ListView
		err := r.DB.Where("entity_type = ? AND is_default = ?", entityType, true).First(&v).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				// 该 entity 还没默认视图,尝试取 is_system 的
				err = r.DB.Where("entity_type = ? AND is_system = ?", entityType, true).First(&v).Error
			}
			if err != nil {
				response.NotFound(c, "该实体暂无默认视图")
				return
			}
		}
		response.OK(c, v)
		return
	}

	// 列表查询
	q := r.DB.Model(&model.ListView{})
	if entityType != "" {
		q = q.Where("entity_type = ?", entityType)
	}
	var views []model.ListView
	if err := q.Order("entity_type ASC, is_default DESC, id ASC").Find(&views).Error; err != nil {
		response.ServerError(c, "查询失败")
		return
	}
	response.OK(c, views)
}

// listViewGet 单个视图详情
func (r *Router) listViewGet(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.FailWithStatus(c, http.StatusBadRequest, 400, "ID 格式错误")
		return
	}
	var v model.ListView
	if err := r.DB.First(&v, uint(id)).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.NotFound(c, "视图不存在")
			return
		}
		response.ServerError(c, "查询失败")
		return
	}
	response.OK(c, v)
}

// listViewCreate admin 创建视图
func (r *Router) listViewCreate(c *gin.Context) {
	var req CreateListViewReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithStatus(c, http.StatusBadRequest, 400, "参数错误: "+err.Error())
		return
	}
	if !model.ListViewEntityType(req.EntityType).IsValid() {
		response.Fail(c, 1, "entity_type 必须是 player 或 order")
		return
	}
	if strings.TrimSpace(req.Name) == "" {
		response.Fail(c, 1, "name 不能为空")
		return
	}
	// name 在同 entity 内唯一
	var exist model.ListView
	if err := r.DB.Where("entity_type = ? AND name = ? AND deleted_at IS NULL", req.EntityType, req.Name).First(&exist).Error; err == nil {
		response.Fail(c, 1, "同实体下视图名重复")
		return
	}

	userID := middleware.GetCurrentUserID(c)

	// 若设置为默认，需清空其它 default
	if req.IsDefault {
		r.DB.Model(&model.ListView{}).
			Where("entity_type = ? AND is_default = ?", req.EntityType, true).
			Update("is_default", false)
	}

	v := model.ListView{
		EntityType: model.ListViewEntityType(req.EntityType),
		Name:       req.Name,
		IsDefault:  req.IsDefault,
		IsSystem:   false,
		Fields:     model.JSONList(req.Fields),
		CreatedBy:  &userID,
	}
	if err := r.DB.Create(&v).Error; err != nil {
		response.ServerError(c, "创建失败: "+err.Error())
		return
	}
	r.DB.First(&v, v.ID)
	response.OK(c, v)
}

// listViewUpdate admin 更新视图（name + fields）
//
// is_system=true 的视图也允许改 name/fields（否则配置没法用了），仅不允许删/复制/设为系统。
func (r *Router) listViewUpdate(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.FailWithStatus(c, http.StatusBadRequest, 400, "ID 格式错误")
		return
	}
	var req UpdateListViewReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithStatus(c, http.StatusBadRequest, 400, "参数错误: "+err.Error())
		return
	}
	var v model.ListView
	if err := r.DB.First(&v, uint(id)).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.NotFound(c, "视图不存在")
			return
		}
		response.ServerError(c, "查询失败")
		return
	}

	updates := map[string]interface{}{}
	if req.Name != nil {
		newName := strings.TrimSpace(*req.Name)
		if newName == "" {
			response.Fail(c, 1, "name 不能为空")
			return
		}
		// 重名检查
		var dup model.ListView
		if err := r.DB.Where("entity_type = ? AND name = ? AND id <> ? AND deleted_at IS NULL", v.EntityType, newName, v.ID).First(&dup).Error; err == nil {
			response.Fail(c, 1, "同实体下视图名重复")
			return
		}
		updates["name"] = newName
	}
	if req.Fields != nil {
		updates["fields"] = model.JSONList(*req.Fields)
	}
	if len(updates) == 0 {
		response.Fail(c, 1, "无更新字段")
		return
	}
	if err := r.DB.Model(&v).Updates(updates).Error; err != nil {
		response.ServerError(c, "更新失败")
		return
	}
	r.DB.First(&v, v.ID)
	response.OK(c, v)
}

// listViewDelete admin 删除视图
func (r *Router) listViewDelete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.FailWithStatus(c, http.StatusBadRequest, 400, "ID 格式错误")
		return
	}
	var v model.ListView
	if err := r.DB.First(&v, uint(id)).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.NotFound(c, "视图不存在")
			return
		}
		response.ServerError(c, "查询失败")
		return
	}
	if v.IsSystem {
		response.Fail(c, 1, "系统视图不能删除")
		return
	}
	if err := r.DB.Delete(&v).Error; err != nil {
		response.ServerError(c, "删除失败")
		return
	}

	// 若删的是默认视图，自动把同 entity 第一个非 system 视图升级为默认
	if v.IsDefault {
		var next model.ListView
		if err := r.DB.Where("entity_type = ? AND is_system = ?", v.EntityType, true).First(&next).Error; err == nil {
			r.DB.Model(&next).Update("is_default", true)
		}
	}
	return
}

// listViewDuplicate admin 复制视图（命名「XX - 副本」）
func (r *Router) listViewDuplicate(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.FailWithStatus(c, http.StatusBadRequest, 400, "ID 格式错误")
		return
	}
	var src model.ListView
	if err := r.DB.First(&src, uint(id)).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.NotFound(c, "视图不存在")
			return
		}
		response.ServerError(c, "查询失败")
		return
	}

	userID := middleware.GetCurrentUserID(c)

	// 找唯一新名字：默认视图 → 默认视图 - 副本；冲突追加 -2, -3 ...
	newName := src.Name + " - 副本"
	for i := 2; ; i++ {
		var exist model.ListView
		err := r.DB.Where("entity_type = ? AND name = ? AND deleted_at IS NULL", src.EntityType, newName).First(&exist).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			break
		}
		if err != nil {
			response.ServerError(c, "查询失败")
			return
		}
		newName = src.Name + " - 副本 (" + strconv.Itoa(i) + ")"
	}

	// 深拷贝 fields
	newFields := make([]map[string]interface{}, len(src.Fields))
	for i, f := range src.Fields {
		cp := make(map[string]interface{}, len(f))
		for k, v := range f {
			cp[k] = v
		}
		newFields[i] = cp
	}

	dup := model.ListView{
		EntityType: src.EntityType,
		Name:       newName,
		IsDefault:  false,
		IsSystem:   false,
		Fields:     model.JSONList(newFields),
		CreatedBy:  &userID,
	}
	if err := r.DB.Create(&dup).Error; err != nil {
		response.ServerError(c, "复制失败: "+err.Error())
		return
	}
	r.DB.First(&dup, dup.ID)
	response.OK(c, dup)
}

// listViewSetDefault admin 设为默认（取消同 entity 其它视图的 is_default）
func (r *Router) listViewSetDefault(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.FailWithStatus(c, http.StatusBadRequest, 400, "ID 格式错误")
		return
	}
	var v model.ListView
	if err := r.DB.First(&v, uint(id)).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.NotFound(c, "视图不存在")
			return
		}
		response.ServerError(c, "查询失败")
		return
	}

	tx := r.DB.Begin()
	if err := tx.Model(&model.ListView{}).
		Where("entity_type = ? AND is_default = ?", v.EntityType, true).
		Update("is_default", false).Error; err != nil {
		tx.Rollback()
		response.ServerError(c, "清除旧默认失败")
		return
	}
	if err := tx.Model(&v).Update("is_default", true).Error; err != nil {
		tx.Rollback()
		response.ServerError(c, "设置默认失败")
		return
	}
	tx.Commit()

	r.DB.First(&v, v.ID)
	response.OK(c, v)
}

// listViewAvailableFields admin 获取可用字段池（按 entity_type）
func (r *Router) listViewAvailableFields(c *gin.Context) {
	entityType := c.Query("entity_type")
	if !model.ListViewEntityType(entityType).IsValid() {
		response.Fail(c, 1, "entity_type 必须是 player 或 order")
		return
	}
	var fields []ListViewAvailableField
	switch model.ListViewEntityType(entityType) {
	case model.ListViewEntityPlayer:
		fields = r.getPlayerFieldPool()
	case model.ListViewEntityOrder:
		fields = r.getOrderFieldPool()
	}
	response.OK(c, fields)
}