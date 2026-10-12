package api

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"crm-backend/internal/model"
	"crm-backend/internal/service"
	"crm-backend/pkg/response"
)

// ============================ 订单管理 ============================

// OrderListFilter 列表过滤
type OrderListFilter struct {
	GameID           uint   `form:"game_id"`
	Status           string `form:"status"`
	Source           string `form:"source"`
	AttributionStatus string `form:"attribution_status"`
	PlayerID         string `form:"player_id"`
	OrderNo          string `form:"order_no"`
	StartDate        string `form:"start_date"`
	EndDate          string `form:"end_date"`
	MinAmount        string `form:"min_amount"`
	MaxAmount        string `form:"max_amount"`
	Page             int    `form:"page"`
	PageSize         int    `form:"page_size"`
	LeadID           uint   `form:"lead_id"`
	Unbound          bool   `form:"unbound"` // 只看待人工绑定(lead_id IS NULL)
}

// orderList 订单列表
func (r *Router) orderList(c *gin.Context) {
	var filter OrderListFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		response.FailWithStatus(c, http.StatusBadRequest, 400, "参数错误: "+err.Error())
		return
	}
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.PageSize < 1 || filter.PageSize > 200 {
		filter.PageSize = 20
	}

	q := r.DB.Model(&model.Order{})

	// 权限:主管看本部门,员工看自己参与的 lead
	roleVal, _ := c.Get("role")
	role, _ := roleVal.(model.Role)
	userIDVal, _ := c.Get("userID")
	userID, _ := userIDVal.(uint)

	switch role {
	case model.RoleAdmin, model.RoleAManager, model.RoleBManager:
		// 全看
	case model.RoleAStaff, model.RoleALead:
		// A 部:只能看到自己创建的 lead 的订单
		q = q.Joins("LEFT JOIN leads ON leads.id = orders.lead_id").
			Where("leads.creator_id = ? OR orders.lead_id IS NULL", userID)
	case model.RoleBStaff, model.RoleBLead:
		// B 部:只能看到自己认领的 lead 的订单
		q = q.Joins("LEFT JOIN leads ON leads.id = orders.lead_id").
			Where("leads.claimed_by = ? OR orders.lead_id IS NULL", userID)
	}

	if filter.GameID > 0 {
		q = q.Where("orders.game_id = ?", filter.GameID)
	}
	if filter.Status != "" {
		q = q.Where("orders.status = ?", filter.Status)
	}
	if filter.Source != "" {
		q = q.Where("orders.source = ?", filter.Source)
	}
	if filter.AttributionStatus != "" {
		q = q.Where("orders.attribution_status = ?", filter.AttributionStatus)
	}
	if filter.PlayerID != "" {
		q = q.Where("orders.player_id = ?", filter.PlayerID)
	}
	if filter.OrderNo != "" {
		q = q.Where("orders.order_no LIKE ?", "%"+filter.OrderNo+"%")
	}
	if filter.LeadID > 0 {
		q = q.Where("orders.lead_id = ?", filter.LeadID)
	}
	if filter.Unbound {
		q = q.Where("orders.lead_id IS NULL")
	}
	if filter.StartDate != "" {
		if t, err := time.Parse("2006-01-02", filter.StartDate); err == nil {
			q = q.Where("orders.order_time >= ?", t)
		}
	}
	if filter.EndDate != "" {
		if t, err := time.Parse("2006-01-02", filter.EndDate); err == nil {
			q = q.Where("orders.order_time < ?", t.AddDate(0, 0, 1))
		}
	}
	if filter.MinAmount != "" {
		if v, err := strconv.ParseFloat(filter.MinAmount, 64); err == nil {
			q = q.Where("orders.amount >= ?", v)
		}
	}
	if filter.MaxAmount != "" {
		if v, err := strconv.ParseFloat(filter.MaxAmount, 64); err == nil {
			q = q.Where("orders.amount <= ?", v)
		}
	}

	var total int64
	q.Count(&total)

	var orders []model.Order
	if err := q.Preload("Game").Preload("Lead").Preload("Lead.Creator").Preload("Lead.ClaimedByUser").
		Preload("Partner").Preload("Importer").Preload("Attributor").
		Order("orders.order_time DESC, orders.id DESC").
		Offset((filter.Page - 1) * filter.PageSize).Limit(filter.PageSize).
		Find(&orders).Error; err != nil {
		response.ServerError(c, "查询失败")
		return
	}

	response.OK(c, gin.H{
		"list":      orders,
		"total":     total,
		"page":      filter.Page,
		"page_size": filter.PageSize,
	})
}

// orderGet 订单详情
func (r *Router) orderGet(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.FailWithStatus(c, http.StatusBadRequest, 400, "ID 格式错误")
		return
	}
	var order model.Order
	if err := r.DB.Preload("Game").Preload("Lead").Preload("Lead.Creator").Preload("Lead.ClaimedByUser").
		Preload("Partner").Preload("Importer").Preload("Attributor").
		First(&order, uint(id)).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.NotFound(c, "订单不存在")
			return
		}
		response.ServerError(c, "查询失败")
		return
	}

	// 候选 lead 列表(供人工绑定时使用)
	var candidates []model.Lead
	if order.LeadID == nil {
		r.DB.Where("player_id = ? AND game_id = ? AND status IN ?",
			order.PlayerID, order.GameID,
			[]model.LeadStatus{model.LeadStatusPending, model.LeadStatusClaimed},
		).Preload("Creator").Preload("ClaimedByUser").
			Order("created_at DESC").Limit(20).Find(&candidates)
	}

	response.OK(c, gin.H{
		"order":      order,
		"candidates": candidates,
	})
}

// BindOrderReq 人工绑定请求
type BindOrderReq struct {
	LeadID uint `json:"lead_id" binding:"required"`
}

// orderBind 人工绑定 lead
func (r *Router) orderBind(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.FailWithStatus(c, http.StatusBadRequest, 400, "ID 格式错误")
		return
	}
	var req BindOrderReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithStatus(c, http.StatusBadRequest, 400, "参数错误: "+err.Error())
		return
	}
	userID, _ := c.Get("userID")
	uid, _ := userID.(uint)

	attributor := service.NewAttributor(r.DB)
	if err := attributor.AttributeManual(uint(id), req.LeadID, uid); err != nil {
		response.Fail(c, 1, err.Error())
		return
	}

	// 重新加载(含关联)
	var order model.Order
	r.DB.Preload("Game").Preload("Lead").Preload("Lead.Creator").
		Preload("Lead.ClaimedByUser").Preload("Attributor").
		First(&order, uint(id))

	// 审计
	service.NewAuditRecorder(r.DB).RecordAction(c, model.AuditOrderBind,
		model.AuditTargetOrder, &order.ID,
		fmt.Sprintf("order#%d → lead#%d", order.ID, order.LeadID),
		nil, nil,
		model.JSONB{"bound_lead_id": order.LeadID})

	response.OK(c, order)
}

// orderImport 手动上传订单(CSV/Excel)
func (r *Router) orderImport(c *gin.Context) {
	gameIDStr := c.PostForm("game_id")
	templateIDStr := c.PostForm("template_id")

	gameID, err := strconv.ParseUint(gameIDStr, 10, 64)
	if err != nil || gameID == 0 {
		response.FailWithStatus(c, http.StatusBadRequest, 400, "game_id 必填")
		return
	}
	templateID, err := strconv.ParseUint(templateIDStr, 10, 64)
	if err != nil || templateID == 0 {
		response.FailWithStatus(c, http.StatusBadRequest, 400, "template_id 必填")
		return
	}

	// 校验游戏存在
	var game model.Game
	if err := r.DB.First(&game, uint(gameID)).Error; err != nil {
		response.Fail(c, 1, "游戏不存在")
		return
	}

	// 校验模板存在且属于该游戏
	var tpl model.UploadTemplate
	if err := r.DB.First(&tpl, uint(templateID)).Error; err != nil {
		response.Fail(c, 1, "模板不存在")
		return
	}
	if tpl.GameID != uint(gameID) {
		response.Fail(c, 1, "模板不属于该游戏")
		return
	}

	// 读取上传文件
	fileHeader, err := c.FormFile("file")
	if err != nil {
		response.FailWithStatus(c, http.StatusBadRequest, 400, "未上传文件: "+err.Error())
		return
	}
	src, err := fileHeader.Open()
	if err != nil {
		response.FailWithStatus(c, http.StatusBadRequest, 400, "打开文件失败")
		return
	}
	defer src.Close()
	fileBytes, err := io.ReadAll(src)
	if err != nil {
		response.FailWithStatus(c, http.StatusBadRequest, 400, "读取文件失败")
		return
	}

	// 解析
	parser := service.NewExcelParser()
	parseRes, err := parser.Parse(fileBytes, fileHeader.Filename, &tpl)
	if err != nil {
		response.Fail(c, 1, "文件解析失败: "+err.Error())
		return
	}

	if len(parseRes.Rows) == 0 {
		response.Fail(c, 1, "未解析到有效数据行")
		return
	}

	// 入库
	userID, _ := c.Get("userID")
	uid, _ := userID.(uint)
	batch := fmt.Sprintf("manual-%s", uuid.New().String()[:8])

	importer := service.NewOrderImporter(r.DB)
	summary, err := importer.ImportRows(c.Request.Context(), parseRes.Rows, service.ImportOptions{
		GameID:      uint(gameID),
		Source:      model.OrderSourceManual,
		ImportBatch: batch,
		ImportedBy:  &uid,
	})
	if err != nil {
		response.ServerError(c, "入库失败: "+err.Error())
		return
	}

	// 审计
	service.NewAuditRecorder(r.DB).RecordAction(c, model.AuditOrderImport,
		model.AuditTargetOrder, nil,
		fmt.Sprintf("批次 %s", batch),
		nil, nil,
		model.JSONB{
			"batch":          batch,
			"total":          summary.Total,
			"inserted":       summary.Inserted,
			"skipped":        summary.Skipped,
			"failed":         summary.Failed,
			"auto_attributed": summary.AutoAttr,
			"pending":        summary.PendingAttr,
		})

	// 告警:新订单入库(pending 状态的告警级别更高)
	if summary.Inserted > 0 {
		alertLevel := service.AlertLevelInfo
		if summary.PendingAttr > 0 {
			alertLevel = service.AlertLevelWarning
		}
		r.Alerter.Send(service.Alert{
			Level:   alertLevel,
			Title:   "订单导入完成",
			Message: fmt.Sprintf("批次 %s: 新增 %d, 跳过 %d, 待人工 %d", batch, summary.Inserted, summary.Skipped, summary.PendingAttr),
			Link:    "/orders",
		})
	}

	response.OK(c, gin.H{
		"summary":     summary,
		"batch":       batch,
		"row_errors":  parseRes.Errors,
		"total_rows":  summary.Total,
	})
}

// orderImportPreview 预览上传(不写入数据库)
func (r *Router) orderImportPreview(c *gin.Context) {
	gameIDStr := c.PostForm("game_id")
	templateIDStr := c.PostForm("template_id")
	gameID, err := strconv.ParseUint(gameIDStr, 10, 64)
	if err != nil || gameID == 0 {
		response.FailWithStatus(c, http.StatusBadRequest, 400, "game_id 必填")
		return
	}
	templateID, err := strconv.ParseUint(templateIDStr, 10, 64)
	if err != nil || templateID == 0 {
		response.FailWithStatus(c, http.StatusBadRequest, 400, "template_id 必填")
		return
	}

	var tpl model.UploadTemplate
	if err := r.DB.First(&tpl, uint(templateID)).Error; err != nil {
		response.Fail(c, 1, "模板不存在")
		return
	}
	if tpl.GameID != uint(gameID) {
		response.Fail(c, 1, "模板不属于该游戏")
		return
	}

	fileHeader, err := c.FormFile("file")
	if err != nil {
		response.FailWithStatus(c, http.StatusBadRequest, 400, "未上传文件: "+err.Error())
		return
	}
	src, _ := fileHeader.Open()
	defer src.Close()
	fileBytes, _ := io.ReadAll(src)

	parser := service.NewExcelParser()
	parseRes, err := parser.Parse(fileBytes, fileHeader.Filename, &tpl)
	if err != nil {
		response.Fail(c, 1, "文件解析失败: "+err.Error())
		return
	}

	// 仅返回前 5 行预览
	preview := parseRes.Rows
	if len(preview) > 5 {
		preview = preview[:5]
	}

	response.OK(c, gin.H{
		"total_rows": len(parseRes.Rows),
		"preview":    preview,
		"errors":     parseRes.Errors,
	})
}

// OrderStatsFilter 统计过滤
type OrderStatsFilter struct {
	GameID    uint   `form:"game_id"`
	StartDate string `form:"start_date"`
	EndDate   string `form:"end_date"`
	GroupBy   string `form:"group_by"` // day/game/team
}

// orderStats 订单统计(charge/revenue/ARPU/业绩分账)
func (r *Router) orderStats(c *gin.Context) {
	var filter OrderStatsFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		response.FailWithStatus(c, http.StatusBadRequest, 400, "参数错误: "+err.Error())
		return
	}
	if filter.GroupBy == "" {
		filter.GroupBy = "day"
	}

	q := r.DB.Model(&model.Order{}).Where("status = ?", model.OrderStatusPaid)

	if filter.GameID > 0 {
		q = q.Where("game_id = ?", filter.GameID)
	}
	if filter.StartDate != "" {
		if t, err := time.Parse("2006-01-02", filter.StartDate); err == nil {
			q = q.Where("order_time >= ?", t)
		}
	}
	if filter.EndDate != "" {
		if t, err := time.Parse("2006-01-02", filter.EndDate); err == nil {
			q = q.Where("order_time < ?", t.AddDate(0, 0, 1))
		}
	}

	// 总览
	var totalAmount float64
	var totalCount int64
	var pendingCount int64
	q.Session(&gorm.Session{}).Select("COALESCE(SUM(amount),0)").Scan(&totalAmount)
	q.Session(&gorm.Session{}).Count(&totalCount)
	r.DB.Model(&model.Order{}).Where("attribution_status = ?", model.AttributionPending).Count(&pendingCount)

	// 按玩家去重人数
	var paidPlayers int64
	r.DB.Model(&model.Order{}).
		Where("status = ?", model.OrderStatusPaid).
		Distinct("player_id").
		Count(&paidPlayers)

	arpu := 0.0
	if paidPlayers > 0 {
		arpu = totalAmount / float64(paidPlayers)
	}

	overview := gin.H{
		"total_revenue":   totalAmount,
		"total_orders":    totalCount,
		"paid_players":    paidPlayers,
		"arpu":            arpu,
		"pending_orders":  pendingCount,
	}

	// A/B 业绩分账(已归属订单)
	var typeStats []struct {
		IsFirstCharge bool
		TotalAmount   float64
		Count         int64
	}
	r.DB.Model(&model.Order{}).
		Select("is_first_charge, COALESCE(SUM(amount),0) as total_amount, COUNT(*) as count").
		Where("status = ? AND lead_id IS NOT NULL", model.OrderStatusPaid).
		Group("is_first_charge").
		Scan(&typeStats)

	aTeamRevenue := 0.0
	bTeamRevenue := 0.0
	aTeamCount := int64(0)
	bTeamCount := int64(0)
	for _, ts := range typeStats {
		if ts.IsFirstCharge {
			aTeamRevenue = ts.TotalAmount
			aTeamCount = ts.Count
		} else {
			bTeamRevenue = ts.TotalAmount
			bTeamCount = ts.Count
		}
	}
	overview["a_team_revenue"] = aTeamRevenue
	overview["b_team_revenue"] = bTeamRevenue
	overview["a_team_orders"] = aTeamCount
	overview["b_team_orders"] = bTeamCount

	// 趋势(按天/按游戏/按团队)
	var trends []gin.H
	switch filter.GroupBy {
	case "day":
		var rows []struct {
			Date        time.Time
			TotalAmount float64
			Count       int64
		}
		r.DB.Model(&model.Order{}).
			Select("DATE(order_time) as date, COALESCE(SUM(amount),0) as total_amount, COUNT(*) as count").
			Where("status = ?", model.OrderStatusPaid).
			Group("DATE(order_time)").
			Order("date ASC").
			Scan(&rows)
		for _, row := range rows {
			trends = append(trends, gin.H{
				"date":   row.Date.Format("2006-01-02"),
				"amount": row.TotalAmount,
				"count":  row.Count,
			})
		}
	case "game":
		var rows []struct {
			GameID      uint
			GameCode    string
			GameName    string
			TotalAmount float64
			Count       int64
		}
		r.DB.Model(&model.Order{}).
			Select("orders.game_id, games.code as game_code, games.name as game_name, COALESCE(SUM(orders.amount),0) as total_amount, COUNT(*) as count").
			Joins("LEFT JOIN games ON games.id = orders.game_id").
			Where("orders.status = ?", model.OrderStatusPaid).
			Group("orders.game_id, games.code, games.name").
			Order("total_amount DESC").
			Scan(&rows)
		for _, row := range rows {
			trends = append(trends, gin.H{
				"game_id":   row.GameID,
				"game_code": row.GameCode,
				"game_name": row.GameName,
				"amount":    row.TotalAmount,
				"count":     row.Count,
			})
		}
	}

	response.OK(c, gin.H{
		"overview": overview,
		"trends":   trends,
	})
}

// orderDelete 删除订单(仅 admin)
func (r *Router) orderDelete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.FailWithStatus(c, http.StatusBadRequest, 400, "ID 格式错误")
		return
	}
	if err := r.DB.Delete(&model.Order{}, uint(id)).Error; err != nil {
		response.ServerError(c, "删除失败")
		return
	}
	response.OK(c, nil)
}

// 辅助函数
func (r *Router) parseJSONBFromStringMap(m map[string]string) model.JSONB {
	res := model.JSONB{}
	for k, v := range m {
		res[k] = v
	}
	return res
}

// 防止 strings 包未引用
var _ = strings.TrimSpace