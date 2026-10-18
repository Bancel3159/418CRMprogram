package api

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"crm-backend/internal/model"
	"crm-backend/internal/service"
	"crm-backend/internal/ws"
	"crm-backend/pkg/response"
)

// webhookReceive 接收合作方推送的订单
//
// URL:    POST /api/v1/orders/webhook/:partner_code
//         (partner_code 即游戏的 code,要求该游戏已配置 partner_type=webhook)
// 鉴权:   HMAC-SHA256(secret, raw_body), 头部 X-Signature: sha256=<hex>
// 协议:   请求体为 JSON 数组或 { "orders": [...] }
//
// P12 实时通知:status=paid 订单入库后向 follow 角色推送 order.paid / order.high_value
// P12 可选字段:payload 里可带 player_last_active_at (RFC3339) — 携带则调用 UpdateLeadActiveTime
func (r *Router) webhookReceive(c *gin.Context) {
	code := c.Param("partner_code")
	if code == "" {
		response.FailWithStatus(c, http.StatusBadRequest, 400, "缺少 partner_code")
		return
	}

	// 读取原始 body(用于 HMAC 验签)
	rawBody, err := io.ReadAll(c.Request.Body)
	if err != nil {
		response.FailWithStatus(c, http.StatusBadRequest, 400, "读取请求体失败")
		return
	}

	// 查 game(partner_type=webhook 才接受回调)
	var game model.Game
	if err := r.DB.Where("code = ? AND partner_type = ?", code, model.PartnerTypeWebhook).First(&game).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.NotFound(c, "合作方(游戏)不存在或未启用 webhook")
			return
		}
		response.ServerError(c, "查询失败")
		return
	}
	if game.Status != model.GameStatusEnabled {
		response.FailWithStatus(c, http.StatusForbidden, 403, "游戏已禁用")
		return
	}

	// 从 partner_config 提取 hmac_secret + field_mappings
	var partnerCfg struct {
		HMACSecret    string                 `json:"hmac_secret"`
		FieldMappings map[string]interface{} `json:"field_mappings"`
	}
	if len(game.PartnerConfig) > 0 {
		if v, ok := game.PartnerConfig["hmac_secret"].(string); ok {
			partnerCfg.HMACSecret = v
		}
		if v, ok := game.PartnerConfig["field_mappings"].(map[string]interface{}); ok {
			partnerCfg.FieldMappings = v
		}
	}

	// HMAC 验签
	if partnerCfg.HMACSecret != "" {
		sig := c.GetHeader("X-Signature")
		if sig == "" {
			response.FailWithStatus(c, http.StatusUnauthorized, 401, "缺少签名 X-Signature")
			return
		}
		// 去掉前缀 sha256=
		sig = strings.TrimPrefix(sig, "sha256=")
		expected := computeHMAC(partnerCfg.HMACSecret, rawBody)
		if !hmac.Equal([]byte(sig), []byte(expected)) {
			response.FailWithStatus(c, http.StatusUnauthorized, 401, "签名校验失败")
			return
		}
	}

	// 解析 JSON: 支持两种格式
	//    1. {"orders": [...]} / {"data": [...]}
	//    2. 直接 [...]
	var payload map[string]interface{}
	if err := bindJSONBytes(rawBody, &payload); err != nil {
		// 尝试直接数组
		var arr []map[string]interface{}
		if err2 := bindJSONBytes(rawBody, &arr); err2 != nil {
			response.FailWithStatus(c, http.StatusBadRequest, 400, "请求体非合法 JSON")
			return
		}
		payload = map[string]interface{}{"orders": arr}
	}

	orderList := extractOrders(payload, model.JSONB(partnerCfg.FieldMappings))
	if len(orderList) == 0 {
		// 兜底:整个 body 视为单条订单
		orderList = []interface{}{payload}
	}

	// PartnerID 暂用 0(已无独立 Partner 表);后续审计追溯用 game_id
	importer := service.NewOrderImporter(r.DB)

	results := gin.H{
		"received": len(orderList),
		"inserted": 0,
		"skipped":  0,
		"failed":   0,
		"errors":   []string{},
	}
	var errorsList []string
	inserted := 0
	skipped := 0
	failed := 0

	for _, rawItem := range orderList {
		item, ok := rawItem.(map[string]interface{})
		if !ok {
			failed++
			errorsList = append(errorsList, "订单项非对象")
			continue
		}

		// 提取字段(支持 FieldMappings 字段重命名)
		mapped := applyMappings(item, model.JSONB(partnerCfg.FieldMappings))

		orderNo, _ := mapped["order_no"].(string)
		if orderNo == "" {
			failed++
			errorsList = append(errorsList, "order_no 缺失")
			continue
		}
		playerID, _ := mapped["player_id"].(string)
		if playerID == "" {
			failed++
			errorsList = append(errorsList, orderNo+": player_id 缺失")
			continue
		}
		amount, _ := toFloat(mapped["amount"])
		if amount == 0 {
			failed++
			errorsList = append(errorsList, orderNo+": amount 缺失或为 0")
			continue
		}

		// game_id(未指定时默认用此游戏的 id)
		var gameID uint = game.ID
		if v, ok := mapped["game_id"].(float64); ok && v > 0 {
			gameID = uint(v)
		} else if v, ok := mapped["game_id"].(string); ok && v != "" {
			if gid, err := strconv.ParseUint(v, 10, 64); err == nil && gid > 0 {
				gameID = uint(gid)
			}
		}
		// 校验游戏存在
		var targetGame model.Game
		if err := r.DB.First(&targetGame, gameID).Error; err != nil {
			failed++
			errorsList = append(errorsList, orderNo+": 游戏不存在")
			continue
		}

		statusStr, _ := mapped["status"].(string)
		status := model.OrderStatusPaid
		if model.OrderStatus(statusStr).IsValid() {
			status = model.OrderStatus(statusStr)
		}

		currency, _ := mapped["currency"].(string)
		if currency == "" {
			currency = "CNY"
		}

		orderTime := time.Now()
		if v, ok := mapped["order_time"].(string); ok && v != "" {
			for _, layout := range []string{time.RFC3339, "2006-01-02 15:04:05", "2006-01-02"} {
				if t, err := time.Parse(layout, v); err == nil {
					orderTime = t
					break
				}
			}
		}

		var paidAt *time.Time
		if v, ok := mapped["paid_at"].(string); ok && v != "" {
			if t, err := time.Parse(time.RFC3339, v); err == nil {
				paidAt = &t
			}
		}

		remark, _ := mapped["remark"].(string)

		order := model.Order{
			GameID:    gameID,
			PlayerID:  playerID,
			OrderNo:   orderNo,
			Amount:    amount,
			Currency:  currency,
			Status:    status,
			OrderTime: orderTime,
			PaidAt:    paidAt,
			Source:    model.OrderSourceWebhook,
			Remark:    remark,
			RawPayload: model.JSONB{
				"raw":          item,
				"game_code":    game.Code,
			},
			ImportBatch: "webhook-" + game.Code,
		}

		saved, err := importer.ImportOne(c.Request.Context(), order)
		if err != nil {
			failed++
			errorsList = append(errorsList, orderNo+": 入库失败 - "+err.Error())
			continue
		}
		// 幂等判断:saved 由 ImportOne 返回,
//   - 如果订单已存在,saved 指向数据库中的旧记录(saved.CreatedAt 早于 now)
//   - 如果是新插入,saved 指向 ImportOne 内的 order 副本(ID 由 GORM 自动填充)
// 用 saved.ID == 0 检测失败场景,真正的判断要靠 ID 是否 > 0
isNew := saved != nil && saved.ID > 0 && saved.CreatedAt.After(time.Now().Add(-5*time.Minute))
		if isNew {
			inserted++
			// 审计(异步)
			service.NewAuditRecorder(r.DB).Record(c.Request.Context(), &model.AuditLog{
				Action:      model.AuditOrderWebhook,
				TargetType:  model.AuditTargetOrder,
				TargetID:    &saved.ID,
				TargetLabel: saved.OrderNo,
				After: model.JSONB{
					"order_no":  saved.OrderNo,
					"amount":    saved.Amount,
					"player_id": saved.PlayerID,
					"status":    saved.Status,
				},
				Extension: model.JSONB{"partner_code": code},
				IP:        c.ClientIP(),
				UserAgent: c.GetHeader("User-Agent"),
				Status:    model.AuditStatusSuccess,
			})
			// 告警
			// P14:归属失败时说清楚原因 —— 之前只有 "pending_attribution" 三个字,
			// 排查时看不出是「玩家不在任何 lead 里」还是「命中多条活跃 lead」,
			// 也不知道是不是 game_id 传错了。补上 player / game / 命中数。
			level := service.AlertLevelInfo
			msg := saved.OrderNo + " / ¥" + service.FormatAmountFloat64(saved.Amount) + " / " + string(saved.AttributionStatus)
			if saved.AttributionStatus == model.AttributionPending {
				level = service.AlertLevelWarning
				msg = fmt.Sprintf("%s / ¥%s / 归属失败: player=%s game=%s 命中 lead=%d 条 → 需人工处理",
					saved.OrderNo, service.FormatAmountFloat64(saved.Amount),
					saved.PlayerID, service.UintToString(uint(saved.GameID)), saved.MatchedLeadCount)
			}
			r.Alerter.Send(service.Alert{
				Level:   level,
				Title:   "Webhook 新订单",
				Message: msg,
				Link:    "/orders/" + service.UintToString(uint(saved.ID)),
			})

			// P12 实时推送 + 活跃时间更新(仅 status=paid 触发)
			if saved.Status == model.OrderStatusPaid {
				// (a) 可选 player_last_active_at — 携带则更新 lead 活跃时间(并按需 mark-active)
				if v, ok := item["player_last_active_at"].(string); ok && v != "" {
					if t, err := time.Parse(time.RFC3339, v); err == nil {
						_ = service.UpdateLeadActiveTime(r.DB, saved.PlayerID, saved.GameID, &t)
					}
				}

				// (b) 实时推送:查 config → 若 enabled 推到 follow 角色
				rtCfg := service.LoadRealtimeConfig(r.DB)
				if rtCfg.Enabled && r.WSHub != nil {
					// 找 lead(用于附 lead_id / claimer 信息;找不到也照样推,前端只用作跳转依据)
					var leadID *uint
					var claimedBy *uint
					var claimedByName string
					var l model.Lead
					if err := r.DB.Select("id, claimed_by").Where("player_id = ? AND game_id = ? AND status <> ?", saved.PlayerID, saved.GameID, model.LeadStatusArchived).Order("id DESC").First(&l).Error; err == nil {
						leadID = &l.ID
						claimedBy = l.ClaimedBy
						if l.ClaimedBy != nil {
							var u model.User
							if err := r.DB.Select("username").First(&u, *l.ClaimedBy).Error; err == nil {
								claimedByName = u.Username
							}
						}
					}

					event := "order.paid"
					if saved.Amount >= rtCfg.HighValueAmount {
						event = "order.high_value"
					}

					r.WSHub.BroadcastToFollowRoles("follow:realtime", ws.MessageTypeAlert, event, gin.H{
						"order_id":         saved.ID,
						"order_no":         saved.OrderNo,
						"amount":           saved.Amount,
						"player_id":        saved.PlayerID,
						"game_id":          saved.GameID,
						"game_code":        game.Code,
						"paid_at":          saved.PaidAt,
						"attribution":      saved.AttributionStatus,
						"lead_id":          leadID,
						"claimed_by":       claimedBy,
						"claimed_by_name":  claimedByName,
						"high_value":       event == "order.high_value",
						"high_value_threshold": rtCfg.HighValueAmount,
					})
				}
			}
		} else {
			skipped++
		}
	}

	results["inserted"] = inserted
	results["skipped"] = skipped
	results["failed"] = failed
	results["errors"] = errorsList

	// 成功接收返回 200(部分失败也算成功接收,失败明细在 errors 里)
	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "ok",
		"data":    results,
	})
}

// computeHMAC 计算 HMAC-SHA256
func computeHMAC(secret string, body []byte) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write(body)
	return hex.EncodeToString(h.Sum(nil))
}

// bindJSONBytes 直接解析字节到对象
func bindJSONBytes(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}

// applyMappings 应用字段映射(将 source_key 重命名为 target_key)
func applyMappings(item map[string]interface{}, mappings model.JSONB) map[string]interface{} {
	out := make(map[string]interface{}, len(item))
	for k, v := range item {
		out[k] = v
	}
	// mappings: {"target_key": "source_key", ...}
	for target, sourceVal := range mappings {
		if target == "orders" || target == "data" {
			continue
		}
		sourceKey, ok := sourceVal.(string)
		if !ok || sourceKey == "" {
			continue
		}
		if v, exists := item[sourceKey]; exists {
			out[target] = v
		}
	}
	return out
}