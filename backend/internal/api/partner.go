package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"crm-backend/internal/model"
	"crm-backend/internal/service"
	"crm-backend/pkg/response"
)

// CreatePartnerReq 创建合作方
type CreatePartnerReq struct {
	Code          string                 `json:"code" binding:"required,min=1,max=50"`
	Name          string                 `json:"name" binding:"required,min=1,max=100"`
	Type          string                 `json:"type" binding:"required,oneof=api webhook"`
	Enabled       *bool                  `json:"enabled"`
	Remark        string                 `json:"remark"`
	BaseURL       string                 `json:"base_url"`
	AuthConfig    map[string]interface{} `json:"auth_config"`
	CronExpr      string                 `json:"cron_expr"`
	FieldMappings map[string]interface{} `json:"field_mappings"`
	HMACSecret    string                 `json:"hmac_secret"`
}

// UpdatePartnerReq 更新
type UpdatePartnerReq struct {
	Name          *string                 `json:"name"`
	Enabled       *bool                   `json:"enabled"`
	Remark        *string                 `json:"remark"`
	BaseURL       *string                 `json:"base_url"`
	AuthConfig    *map[string]interface{} `json:"auth_config"`
	CronExpr      *string                 `json:"cron_expr"`
	FieldMappings *map[string]interface{} `json:"field_mappings"`
	HMACSecret    *string                 `json:"hmac_secret"`
}

// partnerList 合作方列表
func (r *Router) partnerList(c *gin.Context) {
	var partners []model.Partner
	q := r.DB.Model(&model.Partner{}).Order("id ASC")
	typeFilter := c.Query("type")
	if typeFilter != "" {
		q = q.Where("type = ?", typeFilter)
	}
	if err := q.Find(&partners).Error; err != nil {
		response.ServerError(c, "查询失败")
		return
	}

	// 隐藏 HMACSecret 字段
	type partnerSafe struct {
		model.Partner
		HMACSecret string `json:"hmac_secret,omitempty"`
	}
	out := make([]model.Partner, 0, len(partners))
	for _, p := range partners {
		// 仅清空 secret 字段(其他保留)
		cp := p
		cp.HMACSecret = ""
		out = append(out, cp)
	}
	response.OK(c, out)
	_ = partnerSafe{}
}

// partnerCreate 创建合作方
func (r *Router) partnerCreate(c *gin.Context) {
	var req CreatePartnerReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithStatus(c, http.StatusBadRequest, 400, "参数错误: "+err.Error())
		return
	}

	if !model.PartnerType(req.Type).IsValid() {
		response.Fail(c, 1, "type 必须为 api 或 webhook")
		return
	}

	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	partner := model.Partner{
		Code:          req.Code,
		Name:          req.Name,
		Type:          model.PartnerType(req.Type),
		Enabled:       enabled,
		Remark:        req.Remark,
		BaseURL:       req.BaseURL,
		AuthConfig:    model.JSONB(req.AuthConfig),
		CronExpr:      req.CronExpr,
		FieldMappings: model.JSONB(req.FieldMappings),
		HMACSecret:    req.HMACSecret,
	}
	if err := r.DB.Create(&partner).Error; err != nil {
		response.Fail(c, 1, "创建失败(可能 code 重复): "+err.Error())
		return
	}
	partner.HMACSecret = ""
	response.OK(c, partner)
}

// partnerGet 详情
func (r *Router) partnerGet(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.FailWithStatus(c, http.StatusBadRequest, 400, "ID 格式错误")
		return
	}
	var partner model.Partner
	if err := r.DB.First(&partner, uint(id)).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.NotFound(c, "合作方不存在")
			return
		}
		response.ServerError(c, "查询失败")
		return
	}
	partner.HMACSecret = ""
	if partner.Type == model.PartnerTypeWebhook {
		partner.WebhookURL = partner.ComputeWebhookURL(r.baseURLForClient())
	}
	response.OK(c, partner)
}

// partnerUpdate 更新
func (r *Router) partnerUpdate(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.FailWithStatus(c, http.StatusBadRequest, 400, "ID 格式错误")
		return
	}
	var req UpdatePartnerReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithStatus(c, http.StatusBadRequest, 400, "参数错误: "+err.Error())
		return
	}
	var partner model.Partner
	if err := r.DB.First(&partner, uint(id)).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.NotFound(c, "合作方不存在")
			return
		}
		response.ServerError(c, "查询失败")
		return
	}

	updates := map[string]interface{}{}
	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.Enabled != nil {
		updates["enabled"] = *req.Enabled
	}
	if req.Remark != nil {
		updates["remark"] = *req.Remark
	}
	if req.BaseURL != nil {
		updates["base_url"] = *req.BaseURL
	}
	if req.AuthConfig != nil {
		updates["auth_config"] = model.JSONB(*req.AuthConfig)
	}
	if req.CronExpr != nil {
		updates["cron_expr"] = *req.CronExpr
	}
	if req.FieldMappings != nil {
		updates["field_mappings"] = model.JSONB(*req.FieldMappings)
	}
	if req.HMACSecret != nil {
		updates["hmac_secret"] = *req.HMACSecret
	}

	if len(updates) == 0 {
		response.Fail(c, 1, "无更新字段")
		return
	}
	if err := r.DB.Model(&partner).Updates(updates).Error; err != nil {
		response.ServerError(c, "更新失败")
		return
	}
	r.DB.First(&partner, partner.ID)
	partner.HMACSecret = ""
	if partner.Type == model.PartnerTypeWebhook {
		partner.WebhookURL = partner.ComputeWebhookURL(r.baseURLForClient())
	}
	response.OK(c, partner)
}

// partnerDelete 删除
func (r *Router) partnerDelete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.FailWithStatus(c, http.StatusBadRequest, 400, "ID 格式错误")
		return
	}
	if err := r.DB.Delete(&model.Partner{}, uint(id)).Error; err != nil {
		response.ServerError(c, "删除失败")
		return
	}
	response.OK(c, nil)
}

// partnerPull 手动触发 API 拉取
func (r *Router) partnerPull(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.FailWithStatus(c, http.StatusBadRequest, 400, "ID 格式错误")
		return
	}
	var partner model.Partner
	if err := r.DB.First(&partner, uint(id)).Error; err != nil {
		response.NotFound(c, "合作方不存在")
		return
	}
	if partner.Type != model.PartnerTypeAPI {
		response.Fail(c, 1, "该合作方不是 API 类型")
		return
	}
	if !partner.Enabled {
		response.Fail(c, 1, "合作方已禁用")
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 60*time.Second)
	defer cancel()
	summary, err := r.pullPartnerOrders(ctx, &partner)
	if err != nil {
		response.Fail(c, 1, "拉取失败: "+err.Error())
		return
	}

	// 更新 partner 状态
	now := time.Now()
	partner.LastPulledAt = &now
	partner.LastPullStatus = fmt.Sprintf("成功,入库 %d 笔,跳过 %d 笔", summary.Inserted, summary.Skipped)
	partner.LastPullCount = summary.Inserted
	r.DB.Save(&partner)

	response.OK(c, gin.H{
		"summary": summary,
		"partner": gin.H{
			"id":               partner.ID,
			"name":             partner.Name,
			"last_pulled_at":   partner.LastPulledAt,
			"last_pull_status": partner.LastPullStatus,
		},
	})
}

// PullPartnerByID 供 cron 调用的导出方法
func (r *Router) PullPartnerByID(ctx context.Context, partner *model.Partner) error {
	summary, err := r.pullPartnerOrders(ctx, partner)
	if err != nil {
		return err
	}
	now := time.Now()
	partner.LastPulledAt = &now
	partner.LastPullStatus = fmt.Sprintf("cron 自动: 入库 %d 笔, 跳过 %d 笔", summary.Inserted, summary.Skipped)
	partner.LastPullCount = summary.Inserted
	r.DB.Save(partner)
	return nil
}

// pullPartnerOrders 实际拉取合作方订单(单次)
// 默认约定: GET {base_url}/orders?since={last_pulled_at or 24h ago}
// 返回 {"data": [...]}; 字段映射由 partner.FieldMappings 决定
func (r *Router) pullPartnerOrders(ctx context.Context, partner *model.Partner) (*service.ImportSummary, error) {
	if partner.BaseURL == "" {
		return nil, errors.New("partner.base_url 为空")
	}

	// 构造请求 URL(简化版:直接拼 orders 路径)
	url := partner.BaseURL
	if len(url) > 0 && url[len(url)-1] == '/' {
		url = url[:len(url)-1]
	}
	since := time.Now().Add(-24 * time.Hour)
	if partner.LastPulledAt != nil {
		since = *partner.LastPulledAt
	}
	url = fmt.Sprintf("%s/orders?since=%s", url, since.UTC().Format(time.RFC3339))

	httpReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	// 应用 auth
	if partner.AuthConfig != nil {
		if authType, _ := partner.AuthConfig["type"].(string); authType == "bearer" {
			if token, _ := partner.AuthConfig["token"].(string); token != "" {
				httpReq.Header.Set("Authorization", "Bearer "+token)
			}
		} else if authType == "apikey" {
			if key, _ := partner.AuthConfig["key"].(string); key != "" {
				headerName, _ := partner.AuthConfig["header"].(string)
				if headerName == "" {
					headerName = "X-API-Key"
				}
				httpReq.Header.Set(headerName, key)
			}
		}
	}
	httpReq.Header.Set("Accept", "application/json")

	httpClient := &http.Client{Timeout: 50 * time.Second}
	resp, err := httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("HTTP 请求失败: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}
	body, _ := io.ReadAll(resp.Body)

	// 解析响应
	var rawResp map[string]interface{}
	if err := json.Unmarshal(body, &rawResp); err != nil {
		return nil, fmt.Errorf("响应非合法 JSON: %w", err)
	}

	// 提取订单数组
	ordersRaw := extractOrders(rawResp, partner.FieldMappings)
	if len(ordersRaw) == 0 {
		return &service.ImportSummary{}, nil
	}

	// 转换为 ParsedRow(取游戏ID和partnerID)
	pid := partner.ID
	rows := make([]service.ParsedRow, 0, len(ordersRaw))
	for _, raw := range ordersRaw {
		pr, ok := raw.(map[string]interface{})
		if !ok {
			continue
		}
		row := service.ParsedRow{}
		if v, ok := pr["order_no"].(string); ok {
			row.OrderNo = v
		} else {
			continue // 必须有订单号
		}
		if v, ok := pr["player_id"].(string); ok {
			row.PlayerID = v
		} else {
			continue
		}
		row.Amount, _ = toFloat(pr["amount"])
		if v, ok := pr["currency"].(string); ok {
			row.Currency = v
		} else {
			row.Currency = "CNY"
		}
		if v, ok := pr["status"].(string); ok {
			row.Status = model.OrderStatus(v)
		} else {
			row.Status = model.OrderStatusPaid
		}
		if v, ok := pr["order_time"].(string); ok {
			if t, err := time.Parse(time.RFC3339, v); err == nil {
				row.OrderTime = t
			} else if t, err := time.Parse("2006-01-02 15:04:05", v); err == nil {
				row.OrderTime = t
			}
		}
		if row.OrderTime.IsZero() {
			row.OrderTime = time.Now()
		}
		if v, ok := pr["paid_at"].(string); ok {
			if t, err := time.Parse(time.RFC3339, v); err == nil {
				row.PaidAt = &t
			}
		}
		if v, ok := pr["remark"].(string); ok {
			row.Remark = v
		}
		row.RawExtras = pr
		rows = append(rows, row)
	}

	// game_id: 默认取 partner 关联的 game(此处需 partner 有一个 game_id 字段,目前没有)
	// 简化策略: 假定上游返回的订单中包含 game_id
	gameID := uint(0)
	for _, r := range ordersRaw {
		if m, ok := r.(map[string]interface{}); ok {
			if v, ok := m["game_id"].(float64); ok {
				gameID = uint(v)
				break
			}
		}
	}
	if gameID == 0 {
		return nil, errors.New("未能从上游响应中提取 game_id(需在字段映射中配置)")
	}

	importer := service.NewOrderImporter(r.DB)
	batch := fmt.Sprintf("api-%d-%d", partner.ID, time.Now().Unix())
	return importer.ImportRows(ctx, rows, service.ImportOptions{
		GameID:      gameID,
		Source:      model.OrderSourceAPI,
		PartnerID:   &pid,
		ImportBatch: batch,
		RawPayload:  map[string]interface{}{"partner_code": partner.Code},
	})
}

// extractOrders 从上游响应 JSON 中抽取订单数组
func extractOrders(raw map[string]interface{}, mappings model.JSONB) []interface{} {
	// 1. 通过 mappings["orders"] 配置路径定位
	if path, ok := mappings["orders"].(string); ok && path != "" {
		v := digPath(raw, path)
		if arr, ok := v.([]interface{}); ok {
			return arr
		}
	}
	// 2. 默认尝试 data / orders / list 三个常用路径
	for _, k := range []string{"data", "orders", "list"} {
		if v, ok := raw[k]; ok {
			if arr, ok := v.([]interface{}); ok {
				return arr
			}
		}
	}
	return nil
}

// digPath 简易 JSONPath(支持 data.list / data.items 这种 2 段)
func digPath(m map[string]interface{}, path string) interface{} {
	parts := splitDots(path)
	cur := interface{}(m)
	for _, p := range parts {
		if mp, ok := cur.(map[string]interface{}); ok {
			cur = mp[p]
		} else {
			return nil
		}
	}
	return cur
}

func splitDots(s string) []string {
	out := []string{}
	cur := ""
	for _, c := range s {
		if c == '.' {
			out = append(out, cur)
			cur = ""
		} else {
			cur += string(c)
		}
	}
	if cur != "" {
		out = append(out, cur)
	}
	return out
}

func toFloat(v interface{}) (float64, bool) {
	switch x := v.(type) {
	case float64:
		return x, true
	case float32:
		return float64(x), true
	case int:
		return float64(x), true
	case int64:
		return float64(x), true
	case string:
		f, err := strconv.ParseFloat(x, 64)
		return f, err == nil
	}
	return 0, false
}

// baseURLForClient 返回拼接 webhook URL 的基础 URL(简单取 cfg.Server.BaseURL)
func (r *Router) baseURLForClient() string {
	if r.Cfg != nil && r.Cfg.Server.BaseURL != "" {
		return r.Cfg.Server.BaseURL
	}
	return "http://localhost:8080"
}

// partnerRegenerateSecret 重新生成 HMAC 密钥(用于安全轮换)
func (r *Router) partnerRegenerateSecret(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.FailWithStatus(c, http.StatusBadRequest, 400, "ID 格式错误")
		return
	}
	var partner model.Partner
	if err := r.DB.First(&partner, uint(id)).Error; err != nil {
		response.NotFound(c, "合作方不存在")
		return
	}
	if partner.Type != model.PartnerTypeWebhook {
		response.Fail(c, 1, "仅 webhook 类型支持密钥重置")
		return
	}
	newSecret := generateRandomSecret(32)
	r.DB.Model(&partner).Update("hmac_secret", newSecret)
	partner.HMACSecret = newSecret
	partner.WebhookURL = partner.ComputeWebhookURL(r.baseURLForClient())
	response.OK(c, partner)
}

// generateRandomSecret 生成随机密钥
func generateRandomSecret(n int) string {
	const alphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = alphabet[time.Now().UnixNano()%int64(len(alphabet))]
		time.Sleep(time.Nanosecond)
	}
	return string(b)
}

// _ = bytes.NewBuffer
var _ bytes.Buffer