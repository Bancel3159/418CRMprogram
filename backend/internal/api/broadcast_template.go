package api

import (
	"errors"
	"math/rand"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"crm-backend/internal/model"
	"crm-backend/pkg/response"
)

// CreateBroadcastTemplateReq 创建广播模板
type CreateBroadcastTemplateReq struct {
	Name     string `json:"name" binding:"required,min=1,max=100"`
	Template string `json:"template" binding:"required,min=1"`
	Enabled  *bool  `json:"enabled"`
	Weight   int    `json:"weight"`
	Remark   string `json:"remark"`
}

// UpdateBroadcastTemplateReq 更新广播模板
type UpdateBroadcastTemplateReq struct {
	Name     *string `json:"name"`
	Template *string `json:"template"`
	Enabled  *bool   `json:"enabled"`
	Weight   *int    `json:"weight"`
	Remark   *string `json:"remark"`
}

// broadcastTemplateList 广播模板列表
func (r *Router) broadcastTemplateList(c *gin.Context) {
	var templates []model.BroadcastTemplate
	if err := r.DB.Order("id ASC").Find(&templates).Error; err != nil {
		response.ServerError(c, "查询失败")
		return
	}

	response.OK(c, templates)
}

// broadcastTemplateCreate 创建广播模板
func (r *Router) broadcastTemplateCreate(c *gin.Context) {
	var req CreateBroadcastTemplateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithStatus(c, http.StatusBadRequest, 400, "参数错误: "+err.Error())
		return
	}

	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	weight := req.Weight
	if weight < 1 {
		weight = 1
	}

	tpl := model.BroadcastTemplate{
		Name:     req.Name,
		Template: req.Template,
		Enabled:  enabled,
		Weight:   weight,
		Remark:   req.Remark,
	}
	if err := r.DB.Create(&tpl).Error; err != nil {
		response.ServerError(c, "创建失败")
		return
	}

	response.OK(c, tpl)
}

// broadcastTemplateGet 详情
func (r *Router) broadcastTemplateGet(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.FailWithStatus(c, http.StatusBadRequest, 400, "ID 格式错误")
		return
	}

	var tpl model.BroadcastTemplate
	if err := r.DB.First(&tpl, uint(id)).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.NotFound(c, "模板不存在")
			return
		}
		response.ServerError(c, "查询失败")
		return
	}

	response.OK(c, tpl)
}

// broadcastTemplateUpdate 更新
func (r *Router) broadcastTemplateUpdate(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.FailWithStatus(c, http.StatusBadRequest, 400, "ID 格式错误")
		return
	}

	var req UpdateBroadcastTemplateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithStatus(c, http.StatusBadRequest, 400, "参数错误: "+err.Error())
		return
	}

	var tpl model.BroadcastTemplate
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
	if req.Template != nil {
		updates["template"] = *req.Template
	}
	if req.Enabled != nil {
		updates["enabled"] = *req.Enabled
	}
	if req.Weight != nil {
		if *req.Weight < 1 {
			response.Fail(c, 1, "权重必须 >= 1")
			return
		}
		updates["weight"] = *req.Weight
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

	r.DB.First(&tpl, tpl.ID)
	response.OK(c, tpl)
}

// broadcastTemplateDelete 删除
func (r *Router) broadcastTemplateDelete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.FailWithStatus(c, http.StatusBadRequest, 400, "ID 格式错误")
		return
	}

	if err := r.DB.Delete(&model.BroadcastTemplate{}, uint(id)).Error; err != nil {
		response.ServerError(c, "删除失败")
		return
	}

	response.OK(c, nil)
}

// RenderReq 渲染测试请求
type RenderReq struct {
	TemplateID  uint            `json:"template_id"` // 指定模板（可选，不填则随机）
	Game        *string         `json:"game"`
	GameCode    *string         `json:"game_code"`
	Country     *string         `json:"country"`
	Age         *string         `json:"age"`
	Region      *string         `json:"region"`
	Channel     *string         `json:"channel"`
	Community   *string         `json:"community"`
	IsTarget    *string         `json:"is_target"`
	PlayerName  *string         `json:"player_game_name"`
	Username    *string         `json:"player_username"`
	TokenCode   *string         `json:"token_code"`
}

// renderTemplate 渲染模板（用于调试）
func (r *Router) renderTemplate(c *gin.Context) {
	var req RenderReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithStatus(c, http.StatusBadRequest, 400, "参数错误: "+err.Error())
		return
	}

	var tpl model.BroadcastTemplate
	if req.TemplateID > 0 {
		if err := r.DB.First(&tpl, req.TemplateID).Error; err != nil {
			response.NotFound(c, "模板不存在")
			return
		}
	} else {
		// 随机抽取
		var tpls []model.BroadcastTemplate
		r.DB.Where("enabled = ?", true).Find(&tpls)
		if len(tpls) == 0 {
			response.Fail(c, 1, "没有可用的模板")
			return
		}
		tpl = pickRandomTemplate(tpls)
	}

	rendered := renderTemplateContent(tpl.Template, &req)
	response.OK(c, gin.H{
		"template_id":   tpl.ID,
		"template_name": tpl.Name,
		"rendered":      rendered,
	})
}

// pickRandomTemplate 按权重随机抽取模板
func pickRandomTemplate(tpls []model.BroadcastTemplate) model.BroadcastTemplate {
	totalWeight := 0
	for _, t := range tpls {
		if t.Weight > 0 {
			totalWeight += t.Weight
		}
	}
	if totalWeight == 0 {
		return tpls[rand.Intn(len(tpls))]
	}
	r := rand.Intn(totalWeight)
	for _, t := range tpls {
		if t.Weight <= 0 {
			continue
		}
		if r < t.Weight {
			return t
		}
		r -= t.Weight
	}
	return tpls[0]
}

// renderTemplateContent 用字段值替换模板中的占位符
// 支持的占位符：{游戏} {游戏代码} {国家} {年龄} {区服} {渠道} {社群} {是否目标} {玩家游戏名} {玩家用户名} {token码}
func renderTemplateContent(tpl string, data *RenderReq) string {
	replacements := map[string]string{
		"游戏":           strVal(data.Game),
		"游戏代码":        strVal(data.GameCode),
		"国家":           strVal(data.Country),
		"年龄":           strVal(data.Age),
		"区服":           strVal(data.Region),
		"渠道":           strVal(data.Channel),
		"社群":           strVal(data.Community),
		"是否目标":        strVal(data.IsTarget),
		"玩家游戏名":      strVal(data.PlayerName),
		"玩家用户名":      strVal(data.Username),
		"token码":        strVal(data.TokenCode),
	}

	result := tpl
	for k, v := range replacements {
		result = strings.ReplaceAll(result, "{"+k+"}", v)
	}
	return result
}

func strVal(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}