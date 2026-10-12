package api

import (
	"errors"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"crm-backend/internal/middleware"
	"crm-backend/internal/model"
	"crm-backend/internal/service"
	"crm-backend/internal/ws"
	"crm-backend/pkg/response"
)

// CreateLeadReq 创建线索请求（13 字段 + extension + remark）
type CreateLeadReq struct {
	GameID         uint              `json:"game_id" binding:"required"`
	Channel        string            `json:"channel" binding:"required"`
	PlayerUsername string            `json:"player_username" binding:"required"`
	PlayerGameName string            `json:"player_game_name"`
	PlayerID       string            `json:"player_id" binding:"required"`
	Region         string            `json:"region"`
	PlayerCountry  string            `json:"player_country"`
	PlayerAge      int               `json:"player_age"`
	IsTarget       model.IsTarget    `json:"is_target" binding:"required"`
	Community      string            `json:"community"`
	TokenCode      string            `json:"token_code"`
	Screenshots    []string          `json:"screenshots"`
	Extension      map[string]any    `json:"extension"`
	Remark         string            `json:"remark"`
}

// UpdateLeadReq 更新线索请求（部分更新）
type UpdateLeadReq struct {
	PlayerGameName *string         `json:"player_game_name"`
	Region         *string         `json:"region"`
	PlayerCountry  *string         `json:"player_country"`
	PlayerAge      *int            `json:"player_age"`
	IsTarget       *model.IsTarget `json:"is_target"`
	Community      *string         `json:"community"`
	Extension      map[string]any  `json:"extension"`
	Remark         *string         `json:"remark"`
}

// leadList 列表（带权限过滤）
func (r *Router) leadList(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	status := c.Query("status")
	gameID := c.Query("game_id")
	creatorID := c.Query("creator_id")
	claimedBy := c.Query("claimed_by")
	keyword := strings.TrimSpace(c.Query("keyword"))

	q := r.DB.Model(&model.Lead{})
	if status != "" {
		q = q.Where("status = ?", status)
	}
	if gameID != "" {
		if id, err := strconv.ParseUint(gameID, 10, 64); err == nil {
			q = q.Where("game_id = ?", id)
		}
	}
	if creatorID != "" {
		if id, err := strconv.ParseUint(creatorID, 10, 64); err == nil {
			q = q.Where("creator_id = ?", id)
		}
	}
	if claimedBy != "" {
		if id, err := strconv.ParseUint(claimedBy, 10, 64); err == nil {
			q = q.Where("claimed_by = ?", id)
		}
	}
	if keyword != "" {
		// 玩家 ID / 玩家用户名 / 玩家游戏名 模糊
		like := "%" + keyword + "%"
		q = q.Where("player_id ILIKE ? OR player_username ILIKE ? OR player_game_name ILIKE ?", like, like, like)
	}

	// 权限过滤：员工只看自己
	role := middleware.GetCurrentRole(c)
	userID := middleware.GetCurrentUserID(c)
	switch role {
	case model.RoleAStaff, model.RoleBStaff:
		q = q.Where("creator_id = ? OR claimed_by = ?", userID, userID)
	case model.RoleALead, model.RoleBLead:
		// 组长：本组所有数据（暂按本人 + 自己下属；P5+ 完善）
		q = q.Where("creator_id = ? OR claimed_by = ?", userID, userID)
	case model.RoleAManager, model.RoleBManager:
		// 主管：本部门所有（暂按本人 + 自己下属）
		q = q.Where("creator_id = ? OR claimed_by = ?", userID, userID)
		// admin：全部可见（不限制）
	}

	var total int64
	q.Count(&total)

	var leads []model.Lead
	if err := q.Preload("Game").Preload("Creator").Preload("ClaimedByUser").
		Order("id DESC").
		Offset((page - 1) * pageSize).Limit(pageSize).
		Find(&leads).Error; err != nil {
		response.ServerError(c, "查询失败")
		return
	}

	response.OK(c, gin.H{
		"list":      leads,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// poolList 公开池（status = pending）
func (r *Router) poolList(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "50"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 50
	}

	gameID := c.Query("game_id")
	q := r.DB.Model(&model.Lead{}).Where("status = ?", model.LeadStatusPending)
	if gameID != "" {
		if id, err := strconv.ParseUint(gameID, 10, 64); err == nil {
			q = q.Where("game_id = ?", id)
		}
	}

	var total int64
	q.Count(&total)

	var leads []model.Lead
	if err := q.Preload("Game").Preload("Creator").
		Order("id DESC").
		Offset((page - 1) * pageSize).Limit(pageSize).
		Find(&leads).Error; err != nil {
		response.ServerError(c, "查询失败")
		return
	}

	response.OK(c, gin.H{
		"list":      leads,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// leadGet 详情
func (r *Router) leadGet(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.FailWithStatus(c, http.StatusBadRequest, 400, "ID 格式错误")
		return
	}

	var lead model.Lead
	if err := r.DB.Preload("Game").Preload("Creator").Preload("ClaimedByUser").Preload("ReleasedByUser").
		First(&lead, uint(id)).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.NotFound(c, "线索不存在")
			return
		}
		response.ServerError(c, "查询失败")
		return
	}

	// 权限检查
	if !canAccessLead(c, &lead) {
		response.Forbidden(c, "无权限查看该线索")
		return
	}

	// 计算 paid_auto / is_paid
	r.fillPaidFlags(&lead)

	// P9:填充跟进统计(数量 + 最后跟进时间)
	r.DB.Model(&model.LeadFollowRecord{}).
		Where("lead_id = ?", lead.ID).
		Count(&lead.FollowCount)
	r.DB.Model(&model.LeadFollowRecord{}).
		Where("lead_id = ?", lead.ID).
		Order("created_at DESC").Limit(1).
		Pluck("created_at", &lead.LastFollowAt)

	response.OK(c, lead)
}

// leadCreate 提交线索
func (r *Router) leadCreate(c *gin.Context) {
	var req CreateLeadReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithStatus(c, http.StatusBadRequest, 400, "参数错误: "+err.Error())
		return
	}

	// 校验必填
	if !model.IsTarget(req.IsTarget).IsValid() && req.IsTarget != "" {
		// 允许空值（默认 not_target）
	}
	if req.IsTarget == "" {
		req.IsTarget = model.IsTargetFalse
	}

	// 校验游戏存在
	var game model.Game
	if err := r.DB.First(&game, req.GameID).Error; err != nil {
		response.Fail(c, 1, "游戏不存在")
		return
	}

	creatorID := middleware.GetCurrentUserID(c)

	lead := model.Lead{
		GameID:         req.GameID,
		CreatorID:      creatorID,
		Channel:        req.Channel,
		PlayerUsername: req.PlayerUsername,
		PlayerGameName: req.PlayerGameName,
		PlayerID:       req.PlayerID,
		Region:         req.Region,
		PlayerCountry:  req.PlayerCountry,
		PlayerAge:      req.PlayerAge,
		IsTarget:       req.IsTarget,
		Community:      req.Community,
		TokenCode:      req.TokenCode,
		Screenshots:    model.Strings(req.Screenshots),
		Extension:      model.JSONB(req.Extension),
		Remark:         req.Remark,
		Status:         model.LeadStatusPending,
	}
	if lead.Screenshots == nil {
		lead.Screenshots = model.Strings{}
	}
	if lead.Extension == nil {
		lead.Extension = model.JSONB{}
	}

	if err := r.DB.Create(&lead).Error; err != nil {
		response.ServerError(c, "创建失败: "+err.Error())
		return
	}

	// 渲染广播模板
	renderedMsg := r.renderBroadcast(lead.ID)

	// WebSocket 广播
	if r.WSHub != nil {
		var creator model.User
		r.DB.First(&creator, creatorID)
		r.WSHub.BroadcastToFilteredChannel("pool:broadcast", ws.MessageTypeBroadcast, "lead.created", gin.H{
			"id":              lead.ID,
			"game_code":       game.Code,
			"game_name":       game.Name,
			"player_country":  lead.PlayerCountry,
			"player_age":      lead.PlayerAge,
			"region":          lead.Region,
			"channel":         lead.Channel,
			"is_target":       string(lead.IsTarget),
			"creator_name":    creator.RealName,
			"rendered_message": renderedMsg,
		}, shouldSkipPoolBroadcast)
	}

	r.DB.Preload("Game").Preload("Creator").First(&lead, lead.ID)
	response.OK(c, gin.H{
		"lead":            lead,
		"rendered_message": renderedMsg,
	})
}

// renderBroadcast 渲染广播模板
func (r *Router) renderBroadcast(leadID uint) string {
	var lead model.Lead
	if err := r.DB.Preload("Game").First(&lead, leadID).Error; err != nil {
		return ""
	}

	// 抽取一条启用的模板
	var tpls []model.BroadcastTemplate
	r.DB.Where("enabled = ?", true).Find(&tpls)
	if len(tpls) == 0 {
		return fmt.Sprintf("新线索: %s", lead.PlayerUsername)
	}

	tpl := pickRandomTemplate(tpls)

	// 渲染
	data := &RenderReq{
		Game:       stringPtr(lead.Game.Name),
		GameCode:   stringPtr(lead.Game.Code),
		Country:    stringPtr(lead.PlayerCountry),
		Age:        stringPtr(strconv.Itoa(lead.PlayerAge)),
		Region:     stringPtr(lead.Region),
		Channel:    stringPtr(lead.Channel),
		Community:  stringPtr(lead.Community),
		IsTarget:   stringPtr(string(lead.IsTarget)),
		PlayerName: stringPtr(lead.PlayerGameName),
		Username:   stringPtr(lead.PlayerUsername),
		TokenCode:  stringPtr(lead.TokenCode),
	}
	return renderTemplateContent(tpl.Template, data)
}

func stringPtr(s string) *string {
	return &s
}

// leadUpdate 更新线索（仅 pending 状态可更新部分字段）
func (r *Router) leadUpdate(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.FailWithStatus(c, http.StatusBadRequest, 400, "ID 格式错误")
		return
	}

	var lead model.Lead
	if err := r.DB.First(&lead, uint(id)).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.NotFound(c, "线索不存在")
			return
		}
		response.ServerError(c, "查询失败")
		return
	}

	// 权限：仅创建者或主管/admin
	userID := middleware.GetCurrentUserID(c)
	role := middleware.GetCurrentRole(c)
	if lead.CreatorID != userID && !role.IsManager() {
		response.Forbidden(c, "无权修改该线索")
		return
	}

	var req UpdateLeadReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithStatus(c, http.StatusBadRequest, 400, "参数错误: "+err.Error())
		return
	}

	updates := map[string]interface{}{}
	if req.PlayerGameName != nil {
		updates["player_game_name"] = *req.PlayerGameName
	}
	if req.Region != nil {
		updates["region"] = *req.Region
	}
	if req.PlayerCountry != nil {
		updates["player_country"] = *req.PlayerCountry
	}
	if req.PlayerAge != nil {
		updates["player_age"] = *req.PlayerAge
	}
	if req.IsTarget != nil {
		updates["is_target"] = *req.IsTarget
	}
	if req.Community != nil {
		updates["community"] = *req.Community
	}
	if req.Remark != nil {
		updates["remark"] = *req.Remark
	}
	if req.Extension != nil {
		updates["extension"] = model.JSONB(req.Extension)
	}

	if len(updates) == 0 {
		response.Fail(c, 1, "无更新字段")
		return
	}

	if err := r.DB.Model(&lead).Updates(updates).Error; err != nil {
		response.ServerError(c, "更新失败")
		return
	}

	r.DB.Preload("Game").Preload("Creator").Preload("ClaimedByUser").First(&lead, lead.ID)
	response.OK(c, lead)
}

// leadClaim 认领线索（仅 B 部员工/组长/主管/admin）
func (r *Router) leadClaim(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.FailWithStatus(c, http.StatusBadRequest, 400, "ID 格式错误")
		return
	}

	role := middleware.GetCurrentRole(c)
	if !canClaim(role) {
		response.Forbidden(c, "仅 B 部员工可以认领")
		return
	}

	userID := middleware.GetCurrentUserID(c)
	now := time.Now()

	// 事务 + 行锁（FOR UPDATE）
	tx := r.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	var lead model.Lead
	if err := tx.Set("gorm:query_option", "FOR UPDATE").First(&lead, uint(id)).Error; err != nil {
		tx.Rollback()
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.NotFound(c, "线索不存在")
			return
		}
		response.ServerError(c, "查询失败")
		return
	}

	if lead.Status != model.LeadStatusPending {
		tx.Rollback()
		response.Fail(c, 1, "该线索已被认领或归档")
		return
	}

	if err := tx.Model(&lead).Updates(map[string]interface{}{
		"status":     model.LeadStatusClaimed,
		"claimed_by": userID,
		"claimed_at": &now,
	}).Error; err != nil {
		tx.Rollback()
		response.ServerError(c, "认领失败")
		return
	}

	if err := tx.Commit().Error; err != nil {
		response.ServerError(c, "提交事务失败")
		return
	}

	// 审计
	service.NewAuditRecorder(r.DB).RecordAction(c, model.AuditLeadClaim,
		model.AuditTargetLead, &lead.ID,
		fmt.Sprintf("lead#%d %s", lead.ID, lead.PlayerUsername),
		nil, nil,
		model.JSONB{"claimed_by": userID})

	// WebSocket 广播：从公开池移除
	if r.WSHub != nil {
		r.WSHub.BroadcastToChannel("pool:remove", ws.MessageTypeRemove, "lead.claimed", gin.H{
			"id": lead.ID,
		})
	}

	// 返回完整 lead（含 token_code 触发前端自动复制）
	r.DB.Preload("Game").Preload("Creator").Preload("ClaimedByUser").First(&lead, lead.ID)
	response.OK(c, gin.H{
		"lead":            lead,
		"token_to_copy":   lead.TokenCode, // 前端拿到后自动复制
		"clipboard_hint":  "请妥善保管此 token 码",
	})
}

// leadRelease 释放线索（仅主管/admin）
//   - B 部主管：释放 B 部门员工认领的
//   - A 部主管：释放 A 部员工认领的（A 部不直接认领，但可强制归档）
//   - admin：全部可释放
func (r *Router) leadRelease(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.FailWithStatus(c, http.StatusBadRequest, 400, "ID 格式错误")
		return
	}

	role := middleware.GetCurrentRole(c)
	if !canRelease(role) {
		response.Forbidden(c, "仅主管可释放")
		return
	}

	userID := middleware.GetCurrentUserID(c)
	now := time.Now()

	var lead model.Lead
	if err := r.DB.First(&lead, uint(id)).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.NotFound(c, "线索不存在")
			return
		}
		response.ServerError(c, "查询失败")
		return
	}

	if lead.Status != model.LeadStatusClaimed {
		response.Fail(c, 1, "只能释放已认领的线索")
		return
	}

	if err := r.DB.Model(&lead).Updates(map[string]interface{}{
		"status":      model.LeadStatusPending,
		"claimed_by":  nil,
		"claimed_at":  nil,
		"released_by": userID,
		"released_at": &now,
	}).Error; err != nil {
		response.ServerError(c, "释放失败")
		return
	}

	// WebSocket 广播：新线索进入公开池
	if r.WSHub != nil {
		var game model.Game
		r.DB.First(&game, lead.GameID)
		r.WSHub.BroadcastToFilteredChannel("pool:broadcast", ws.MessageTypeBroadcast, "lead.released", gin.H{
			"id":            lead.ID,
			"game_code":     game.Code,
			"rendered_message": r.renderBroadcast(lead.ID),
		}, shouldSkipPoolBroadcast)
	}

	// 审计
	service.NewAuditRecorder(r.DB).RecordAction(c, model.AuditLeadRelease,
		model.AuditTargetLead, &lead.ID,
		fmt.Sprintf("lead#%d", lead.ID),
		nil, nil,
		model.JSONB{"released_by": userID})

	r.DB.Preload("Game").Preload("Creator").Preload("ClaimedByUser").First(&lead, lead.ID)
	response.OK(c, lead)
}

// leadArchive 归档线索（流失/无效）
func (r *Router) leadArchive(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.FailWithStatus(c, http.StatusBadRequest, 400, "ID 格式错误")
		return
	}

	role := middleware.GetCurrentRole(c)
	userID := middleware.GetCurrentUserID(c)

	var lead model.Lead
	if err := r.DB.First(&lead, uint(id)).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.NotFound(c, "线索不存在")
			return
		}
		response.ServerError(c, "查询失败")
		return
	}

	// 权限：创建者、认领人、主管、admin
	canArchive := lead.CreatorID == userID ||
		(lead.ClaimedBy != nil && *lead.ClaimedBy == userID) ||
		role.IsManager()
	if !canArchive {
		response.Forbidden(c, "无权归档")
		return
	}

	now := time.Now()
	if err := r.DB.Model(&lead).Updates(map[string]interface{}{
		"status":      model.LeadStatusArchived,
		"released_by": userID,
		"released_at": &now,
	}).Error; err != nil {
		response.ServerError(c, "归档失败")
		return
	}

	// 广播：从公开池移除（如果还在 pending 状态被归档）
	if lead.Status == model.LeadStatusPending && r.WSHub != nil {
		r.WSHub.BroadcastToChannel("pool:remove", ws.MessageTypeRemove, "lead.archived", gin.H{
			"id": lead.ID,
		})
	}

	// 审计
	service.NewAuditRecorder(r.DB).RecordAction(c, model.AuditLeadArchive,
		model.AuditTargetLead, &lead.ID,
		fmt.Sprintf("lead#%d", lead.ID),
		nil, nil,
		model.JSONB{"archived_by": userID})

	r.DB.Preload("Game").First(&lead, lead.ID)
	response.OK(c, lead)
}

// ========== 权限辅助函数 ==========

// canClaim 是否可认领线索
func canClaim(role model.Role) bool {
	switch role {
	case model.RoleBStaff, model.RoleBLead, model.RoleBManager, model.RoleAdmin:
		return true
	}
	return false
}

// canRelease 是否可释放线索（仅主管）
func canRelease(role model.Role) bool {
	return role.IsManager()
}

// canAccessLead 是否可访问线索详情
func canAccessLead(c *gin.Context, lead *model.Lead) bool {
	role := middleware.GetCurrentRole(c)
	userID := middleware.GetCurrentUserID(c)

	if role == model.RoleAdmin {
		return true
	}
	// 创建者 / 认领人
	if lead.CreatorID == userID {
		return true
	}
	if lead.ClaimedBy != nil && *lead.ClaimedBy == userID {
		return true
	}
	// 主管/组长：暂按"全部允许"（后续按部门数据权限细化）
	if role.IsLead() {
		return true
	}
	return false
}

// leadDelete 删除线索（仅 admin，软删除）
func (r *Router) leadDelete(c *gin.Context) {
	role := middleware.GetCurrentRole(c)
	if role != model.RoleAdmin {
		response.Forbidden(c, "仅 admin 可删除")
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.FailWithStatus(c, http.StatusBadRequest, 400, "ID 格式错误")
		return
	}

	if err := r.DB.Delete(&model.Lead{}, uint(id)).Error; err != nil {
		response.ServerError(c, "删除失败")
		return
	}

	response.OK(c, nil)
}

// ========== WebSocket 连接端点 ==========

// wsConnect WebSocket 连接
func (r *Router) wsConnect(c *gin.Context) {
	tokenStr := c.Query("token")
	if tokenStr == "" {
		response.Unauthorized(c, "缺少 token")
		return
	}

	claims, err := r.JWT.ParseAccessToken(tokenStr)
	if err != nil {
		response.Unauthorized(c, "token 无效")
		return
	}

	// 升级连接
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("[WS] upgrade error: %v", err)
		return
	}

	// 默认订阅频道
	channels := make(map[string]bool)
	channels["pool:broadcast"] = true
	channels["pool:remove"] = true

	// P12 follow 端订阅实时通知频道(订单 / 高额 / 玩家激活)
	if isFollowRole(claims.Role) {
		channels["follow:realtime"] = true
	}

	// admin 自动订阅告警频道(P8)
	if model.Role(claims.Role) == model.RoleAdmin {
		channels["alert:admin"] = true
	}

	// 个人频道
	channels[fmt.Sprintf("user:%d", claims.UserID)] = true

	client := ws.NewClient(
		fmt.Sprintf("u%d_%d", claims.UserID, time.Now().UnixNano()),
		claims.UserID,
		claims.Username,
		claims.Role,
		claims.DeptID,
		channels,
		conn,
		r.WSHub,
		// 从 DB 读最新的接单状态(可能是另一台设备刚改的)
		getAcceptOrders(r.DB, claims.UserID),
	)

	r.WSHub.RegisterClient(client)

	// 启动读写 goroutine
	client.Start()

	// 发送欢迎消息
	client.SendJSON(ws.Message{
		Type:    "system",
		Channel: "system",
		Event:   "connected",
		Data: gin.H{
			"user_id":  claims.UserID,
			"username": claims.Username,
			"channels": channelsToSlice(channels),
		},
		Time: time.Now(),
	})
}

func channelsToSlice(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

// getAcceptOrders 从 DB 读用户的接单状态(默认 true,允许不存在的用户 fallback)
// 用于 WS 客户端注册时把接单状态写到 Client.AcceptOrders
func getAcceptOrders(db *gorm.DB, userID uint) bool {
	var u model.User
	if err := db.Select("accept_orders").First(&u, userID).Error; err != nil {
		return true
	}
	return u.AcceptOrders
}

// isFollowRole 是否为 follow 端角色(用于 pool:broadcast 过滤)
// admin 不视作 follow 端(他看池子是为了管理,不受接单开关影响)
func isFollowRole(role string) bool {
	switch model.Role(role) {
	case model.RoleBStaff, model.RoleBLead, model.RoleBManager:
		return true
	}
	return false
}

// shouldSkipPoolBroadcast pool:broadcast 过滤函数:
//   follow 端用户 + 关了接单 → 跳过
//   其他角色 / 接单开 → 不跳过
func shouldSkipPoolBroadcast(c *ws.Client) bool {
	if !isFollowRole(c.Role) {
		return false
	}
	return !c.AcceptOrders
}

// 占位：避免 import 报错
var _ = strings.Builder{}
var _ = rand.Intn
// ============ 付费标记(管理员/主管/组长 可手标) ============

// canMarkPaid 是否可手动标记已付费
//   允许:admin / A manager / A lead / B manager / B lead
//   禁止:staff(any)
func canMarkPaid(role model.Role) bool {
	switch role {
	case model.RoleAdmin, model.RoleAManager, model.RoleALead, model.RoleBManager, model.RoleBLead:
		return true
	}
	return false
}

// leadMarkPaid 标记为已付费(手动覆盖 / 兜底)
func (r *Router) leadMarkPaid(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.FailWithStatus(c, http.StatusBadRequest, 400, "ID 格式错误")
		return
	}

	role := middleware.GetCurrentRole(c)
	if !canMarkPaid(role) {
		response.Forbidden(c, "仅 admin / 主管 / 组长 可标记付费")
		return
	}

	var lead model.Lead
	if err := r.DB.First(&lead, uint(id)).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.NotFound(c, "线索不存在")
			return
		}
		response.ServerError(c, "查询失败")
		return
	}

	userID := middleware.GetCurrentUserID(c)
	now := time.Now()

	if err := r.DB.Model(&lead).Updates(map[string]interface{}{
		"paid_mark":     true,
		"paid_marked_by": userID,
		"paid_marked_at": &now,
	}).Error; err != nil {
		response.ServerError(c, "标记失败")
		return
	}

	// 审计
	service.NewAuditRecorder(r.DB).RecordAction(c, model.AuditLeadPaid,
		model.AuditTargetLead, &lead.ID,
		fmt.Sprintf("lead#%d %s", lead.ID, lead.PlayerUsername),
		nil, nil,
		model.JSONB{"paid_marked_by": userID, "paid_marked_at": now})

	// 重新加载并填充 paid_auto / is_paid
	r.fillPaidFlags(&lead)
	r.DB.Preload("Game").Preload("Creator").Preload("ClaimedByUser").First(&lead, lead.ID)
	response.OK(c, lead)
}

// leadUnmarkPaid 取消已付费标记
func (r *Router) leadUnmarkPaid(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.FailWithStatus(c, http.StatusBadRequest, 400, "ID 格式错误")
		return
	}

	role := middleware.GetCurrentRole(c)
	if !canMarkPaid(role) {
		response.Forbidden(c, "仅 admin / 主管 / 组长 可取消付费标记")
		return
	}

	var lead model.Lead
	if err := r.DB.First(&lead, uint(id)).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.NotFound(c, "线索不存在")
			return
		}
		response.ServerError(c, "查询失败")
		return
	}

	userID := middleware.GetCurrentUserID(c)
	now := time.Now()

	if err := r.DB.Model(&lead).Updates(map[string]interface{}{
		"paid_mark":     false,
		"paid_marked_by": nil,
		"paid_marked_at": nil,
		"updated_at":     &now,
	}).Error; err != nil {
		response.ServerError(c, "取消标记失败")
		return
	}

	// 审计
	service.NewAuditRecorder(r.DB).RecordAction(c, model.AuditLeadUnpaid,
		model.AuditTargetLead, &lead.ID,
		fmt.Sprintf("lead#%d %s", lead.ID, lead.PlayerUsername),
		nil, nil,
		model.JSONB{"unmarked_by": userID, "unmarked_at": now})

	r.fillPaidFlags(&lead)
	r.DB.Preload("Game").Preload("Creator").Preload("ClaimedByUser").First(&lead, lead.ID)
	response.OK(c, lead)
}

// fillPaidFlags 填充 paid_auto / is_paid:
//   paid_auto = orders 表中存在 player_id = leads.player_id 且 status = 'paid' 的订单
//   is_paid    = paid_mark || paid_auto
func (r *Router) fillPaidFlags(lead *model.Lead) {
	var n int64
	r.DB.Model(&model.Order{}).
		Where("player_id = ? AND status = ?", lead.PlayerID, model.OrderStatusPaid).
		Count(&n)
	lead.PaidAuto = n > 0
	lead.IsPaid = lead.PaidMark || lead.PaidAuto
}

// fillPaidFlagsSlice 批量填充
func (r *Router) fillPaidFlagsSlice(leads *[]model.Lead) {
	if len(*leads) == 0 {
		return
	}
	// 收集 player_ids
	playerIDs := make([]string, 0, len(*leads))
	for _, l := range *leads {
		if l.PlayerID != "" {
			playerIDs = append(playerIDs, l.PlayerID)
		}
	}
	if len(playerIDs) == 0 {
		return
	}
	type row struct {
		PlayerID string
		Cnt      int64
	}
	var rows []row
	r.DB.Model(&model.Order{}).
		Select("player_id, COUNT(*) AS cnt").
		Where("player_id IN ? AND status = ?", playerIDs, model.OrderStatusPaid).
		Group("player_id").
		Scan(&rows)
	paidMap := make(map[string]bool, len(rows))
	for _, r := range rows {
		paidMap[r.PlayerID] = true
	}
	for i := range *leads {
		(*leads)[i].PaidAuto = paidMap[(*leads)[i].PlayerID]
		(*leads)[i].IsPaid = (*leads)[i].PaidMark || (*leads)[i].PaidAuto
	}
}