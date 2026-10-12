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

// CreateUploadTemplateReq 创建模板请求
type CreateUploadTemplateReq struct {
	Name           string                 `json:"name" binding:"required,min=1,max=100"`
	GameID         uint                   `json:"game_id" binding:"required"`
	ColumnMappings map[string]interface{} `json:"column_mappings" binding:"required"`
	TimeFormat     string                 `json:"time_format"`
	StatusMapping  map[string]string      `json:"status_mapping"`
	Currency       string                 `json:"currency"`
	SampleFileURL  string                 `json:"sample_file_url"`
	Remark         string                 `json:"remark"`
}

// UpdateUploadTemplateReq 更新模板
type UpdateUploadTemplateReq struct {
	Name           *string                 `json:"name"`
	ColumnMappings map[string]interface{}  `json:"column_mappings"`
	TimeFormat     *string                 `json:"time_format"`
	StatusMapping  *map[string]string      `json:"status_mapping"`
	Currency       *string                 `json:"currency"`
	SampleFileURL  *string                 `json:"sample_file_url"`
	Remark         *string                 `json:"remark"`
}

// uploadTemplateList 模板列表
func (r *Router) uploadTemplateList(c *gin.Context) {
	gameIDStr := c.Query("game_id")
	q := r.DB.Model(&model.UploadTemplate{})
	if gameIDStr != "" {
		if gid, err := strconv.ParseUint(gameIDStr, 10, 64); err == nil {
			q = q.Where("game_id = ?", uint(gid))
		}
	}

	var tpls []model.UploadTemplate
	if err := q.Preload("Game").Preload("Creator").Order("id DESC").Find(&tpls).Error; err != nil {
		response.ServerError(c, "查询失败")
		return
	}
	response.OK(c, tpls)
}

// uploadTemplateCreate 创建模板
func (r *Router) uploadTemplateCreate(c *gin.Context) {
	var req CreateUploadTemplateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithStatus(c, http.StatusBadRequest, 400, "参数错误: "+err.Error())
		return
	}

	// 校验游戏
	var game model.Game
	if err := r.DB.First(&game, req.GameID).Error; err != nil {
		response.Fail(c, 1, "游戏不存在")
		return
	}

	// 校验 column_mappings 必含字段
	required := []string{"order_no", "player_id", "amount", "order_time"}
	for _, f := range required {
		if _, ok := req.ColumnMappings[f]; !ok {
			response.Fail(c, 1, "column_mappings 缺少必填字段: "+f)
			return
		}
	}

	timeFormat := req.TimeFormat
	if timeFormat == "" {
		timeFormat = "2006-01-02 15:04:05"
	}
	currency := req.Currency
	if currency == "" {
		currency = "CNY"
	}
	statusMapping := model.JSONB{}
	for k, v := range req.StatusMapping {
		statusMapping[k] = v
	}

	userID, _ := c.Get("user_id")
	uid, _ := userID.(uint)

	tpl := model.UploadTemplate{
		Name:           req.Name,
		GameID:         req.GameID,
		ColumnMappings: model.JSONB(req.ColumnMappings),
		TimeFormat:     timeFormat,
		StatusMapping:  statusMapping,
		Currency:       currency,
		SampleFileURL:  req.SampleFileURL,
		Remark:         req.Remark,
		CreatedBy:      &uid,
	}
	if err := r.DB.Create(&tpl).Error; err != nil {
		response.ServerError(c, "创建失败")
		return
	}

	r.DB.Preload("Game").Preload("Creator").First(&tpl, tpl.ID)
	response.OK(c, tpl)
}

// uploadTemplateGet 模板详情
func (r *Router) uploadTemplateGet(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.FailWithStatus(c, http.StatusBadRequest, 400, "ID 格式错误")
		return
	}
	var tpl model.UploadTemplate
	if err := r.DB.Preload("Game").Preload("Creator").First(&tpl, uint(id)).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.NotFound(c, "模板不存在")
			return
		}
		response.ServerError(c, "查询失败")
		return
	}
	response.OK(c, tpl)
}

// uploadTemplateUpdate 更新模板
func (r *Router) uploadTemplateUpdate(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.FailWithStatus(c, http.StatusBadRequest, 400, "ID 格式错误")
		return
	}
	var req UpdateUploadTemplateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithStatus(c, http.StatusBadRequest, 400, "参数错误: "+err.Error())
		return
	}

	var tpl model.UploadTemplate
	if err := r.DB.First(&tpl, uint(id)).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.NotFound(c, "模板不存在")
			return
		}
		response.ServerError(c, "查询失败")
		return
	}

	updates := map[string]interface{}{}
	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.ColumnMappings != nil {
		required := []string{"order_no", "player_id", "amount", "order_time"}
		for _, f := range required {
			if _, ok := req.ColumnMappings[f]; !ok {
				response.Fail(c, 1, "column_mappings 缺少必填字段: "+f)
				return
			}
		}
		updates["column_mappings"] = model.JSONB(req.ColumnMappings)
	}
	if req.TimeFormat != nil {
		updates["time_format"] = *req.TimeFormat
	}
	if req.StatusMapping != nil {
		updates["status_mapping"] = model.JSONB{}
		for k, v := range *req.StatusMapping {
			updates["status_mapping"].(model.JSONB)[k] = v
		}
	}
	if req.Currency != nil {
		updates["currency"] = *req.Currency
	}
	if req.SampleFileURL != nil {
		updates["sample_file_url"] = *req.SampleFileURL
	}
	if req.Remark != nil {
		updates["remark"] = *req.Remark
	}

	if len(updates) == 0 {
		response.Fail(c, 1, "无更新字段")
		return
	}
	if err := r.DB.Model(&tpl).Updates(updates).Error; err != nil {
		response.ServerError(c, "更新失败")
		return
	}
	r.DB.Preload("Game").Preload("Creator").First(&tpl, tpl.ID)
	response.OK(c, tpl)
}

// uploadTemplateDelete 删除模板
func (r *Router) uploadTemplateDelete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.FailWithStatus(c, http.StatusBadRequest, 400, "ID 格式错误")
		return
	}
	if err := r.DB.Delete(&model.UploadTemplate{}, uint(id)).Error; err != nil {
		response.ServerError(c, "删除失败")
		return
	}
	response.OK(c, nil)
}