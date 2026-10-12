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

// CreateGameReq 创建游戏
type CreateGameReq struct {
	Code         string                  `json:"code" binding:"required,min=1,max=50"`
	Name         string                  `json:"name" binding:"required,min=1,max=100"`
	Company      string                  `json:"company"`
	Status       model.GameStatus        `json:"status"`
	Remark       string                  `json:"remark"`
	PartnerType  *model.PartnerType      `json:"partner_type"`   // null = 不启用数据回传
	PartnerConfig map[string]interface{} `json:"partner_config"` // 根据 type 存字段
}

// UpdateGameReq 更新游戏
type UpdateGameReq struct {
	Name          *string                  `json:"name"`
	Company       *string                  `json:"company"`
	Status        *model.GameStatus        `json:"status"`
	Remark        *string                  `json:"remark"`
	PartnerType   *model.PartnerType       `json:"partner_type"`   // 显式传 null 表示关闭
	PartnerConfig *map[string]interface{}  `json:"partner_config"`
}

// gameList 游戏列表
func (r *Router) gameList(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	status := c.Query("status")

	q := r.DB.Model(&model.Game{})
	if status != "" {
		q = q.Where("status = ?", status)
	}

	var total int64
	q.Count(&total)

	var games []model.Game
	if err := q.Order("id ASC").
		Offset((page - 1) * pageSize).Limit(pageSize).
		Find(&games).Error; err != nil {
		response.ServerError(c, "查询失败")
		return
	}

	// 填充运行时计算字段
	for i := range games {
		games[i].PartnerEnabled = games[i].ComputePartnerEnabled()
	}

	response.OK(c, gin.H{
		"list":      games,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// gameCreate 创建游戏
func (r *Router) gameCreate(c *gin.Context) {
	var req CreateGameReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithStatus(c, http.StatusBadRequest, 400, "参数错误: "+err.Error())
		return
	}

	var existing model.Game
	if err := r.DB.Where("code = ?", req.Code).First(&existing).Error; err == nil {
		response.Fail(c, 1, "游戏代码已存在")
		return
	}

	status := req.Status
	if status == "" {
		status = model.GameStatusEnabled
	}

	// 校验 partner_type
	if req.PartnerType != nil && *req.PartnerType != "" && !req.PartnerType.IsValid() {
		response.Fail(c, 1, "partner_type 必须为 api 或 webhook")
		return
	}

	game := model.Game{
		Code:          req.Code,
		Name:          req.Name,
		Company:       req.Company,
		Status:        status,
		Remark:        req.Remark,
		PartnerType:   req.PartnerType,
		PartnerConfig: model.JSONB(req.PartnerConfig),
	}
	if err := r.DB.Create(&game).Error; err != nil {
		response.ServerError(c, "创建失败")
		return
	}
	game.PartnerEnabled = game.ComputePartnerEnabled()

	response.OK(c, game)
}

// gameGet 游戏详情
func (r *Router) gameGet(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.FailWithStatus(c, http.StatusBadRequest, 400, "ID 格式错误")
		return
	}

	var game model.Game
	if err := r.DB.First(&game, uint(id)).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.NotFound(c, "游戏不存在")
			return
		}
		response.ServerError(c, "查询失败")
		return
	}
	game.PartnerEnabled = game.ComputePartnerEnabled()

	response.OK(c, game)
}

// gameUpdate 更新游戏
func (r *Router) gameUpdate(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.FailWithStatus(c, http.StatusBadRequest, 400, "ID 格式错误")
		return
	}

	var req UpdateGameReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithStatus(c, http.StatusBadRequest, 400, "参数错误: "+err.Error())
		return
	}

	var game model.Game
	if err := r.DB.First(&game, uint(id)).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.NotFound(c, "游戏不存在")
			return
		}
		response.ServerError(c, "查询失败")
		return
	}

	updates := map[string]interface{}{}
	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.Company != nil {
		updates["company"] = *req.Company
	}
	if req.Status != nil {
		updates["status"] = *req.Status
	}
	if req.Remark != nil {
		updates["remark"] = *req.Remark
	}
	if req.PartnerType != nil {
		// 校验
		if *req.PartnerType != "" && !req.PartnerType.IsValid() {
			response.Fail(c, 1, "partner_type 必须为 api 或 webhook")
			return
		}
		// 空字符串 = 关闭数据回传
		if *req.PartnerType == "" {
			updates["partner_type"] = nil
		} else {
			updates["partner_type"] = *req.PartnerType
		}
	}
	if req.PartnerConfig != nil {
		updates["partner_config"] = model.JSONB(*req.PartnerConfig)
	}

	if len(updates) == 0 {
		response.Fail(c, 1, "无更新字段")
		return
	}

	if err := r.DB.Model(&game).Updates(updates).Error; err != nil {
		response.ServerError(c, "更新失败")
		return
	}

	r.DB.First(&game, game.ID)
	game.PartnerEnabled = game.ComputePartnerEnabled()
	response.OK(c, game)
}

// gameDelete 删除游戏
func (r *Router) gameDelete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.FailWithStatus(c, http.StatusBadRequest, 400, "ID 格式错误")
		return
	}

	if err := r.DB.Delete(&model.Game{}, uint(id)).Error; err != nil {
		response.ServerError(c, "删除失败")
		return
	}

	response.OK(c, nil)
}