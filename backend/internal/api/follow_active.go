package api

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"crm-backend/internal/model"
	"crm-backend/internal/service"
	"crm-backend/internal/ws"
	"crm-backend/pkg/response"
)

// playerActiveUpdate 标记玩家活跃时间(手动 / 合作方推送 / 系统内复用入口)
//
// URL:   POST /api/v1/players/:player_id/active
// 鉴权:  需要登录(JWT);任意角色可调(合作方推过来也用 JWT 配 partner 账号即可)
// Body: {"game_id": 1, "event_time": "2026-10-04T01:00:00Z", "source": "manual|webhook|api"}
//   - game_id    必填,定位 lead 用
//   - event_time 可选,默认 now
//   - source     仅审计标记;不影响行为(可选)
//
// 行为:
//   1. 调 service.UpdateLeadActiveTime 更新 lead.last_player_active_at
//   2. 若 lead.IsLost=true → 置回 is_lost=false + 写 reactivated_at(在 service 内已处理)
//   3. 推 WS 事件 player.react 到 follow 角色(让 follow 端「玩家重新激活」提示)
func (r *Router) playerActiveUpdate(c *gin.Context) {
	playerID := c.Param("player_id")
	if playerID == "" {
		response.FailWithStatus(c, http.StatusBadRequest, 400, "player_id 不能为空")
		return
	}

	var req struct {
		GameID    uint      `json:"game_id"`
		EventTime time.Time `json:"event_time"`
		Source    string    `json:"source"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithStatus(c, http.StatusBadRequest, 400, "请求体非法: "+err.Error())
		return
	}
	if req.GameID == 0 {
		response.FailWithStatus(c, http.StatusBadRequest, 400, "game_id 必填")
		return
	}
	if req.Source == "" {
		req.Source = "manual"
	}

	var eventTime *time.Time
	if !req.EventTime.IsZero() {
		eventTime = &req.EventTime
	}

	// 找 lead(用于决定是否已流失 + WS 推送附 claimer 信息)
	var lead model.Lead
	if err := r.DB.Where("player_id = ? AND game_id = ? AND status <> ?", playerID, req.GameID, model.LeadStatusArchived).
		Order("id DESC").First(&lead).Error; err != nil {
		response.NotFound(c, "找不到该玩家的活跃 lead")
		return
	}
	wasLost := lead.IsLost

	if err := service.UpdateLeadActiveTime(r.DB, playerID, req.GameID, eventTime); err != nil {
		response.ServerError(c, "更新活跃时间失败: "+err.Error())
		return
	}

	// WS 推送:follow 部门广播
	if r.WSHub != nil {
		var gameCode string
		var g model.Game
		if err := r.DB.Select("code").First(&g, lead.GameID).Error; err == nil {
			gameCode = g.Code
		}
		var claimedByName string
		if lead.ClaimedBy != nil {
			var u model.User
			if err := r.DB.Select("username").First(&u, *lead.ClaimedBy).Error; err == nil {
				claimedByName = u.Username
			}
		}

		evt := eventTimeOrNow(eventTime)
		r.WSHub.BroadcastToFollowRoles("follow:realtime", ws.MessageTypeAlert, "player.react", gin.H{
			"lead_id":         lead.ID,
			"player_id":       playerID,
			"game_id":         req.GameID,
			"game_code":       gameCode,
			"was_lost":        wasLost,
			"event_time":      evt,
			"source":          req.Source,
			"claimed_by":      lead.ClaimedBy,
			"claimed_by_name": claimedByName,
		})
	}

	response.OK(c, gin.H{
		"lead_id":    lead.ID,
		"was_lost":   wasLost,
		"player_id":  playerID,
		"game_id":    req.GameID,
		"event_time": eventTimeOrNow(eventTime),
	})
}

// eventTimeOrNow 安全返回指针所指值,空时 fallback 到 now
func eventTimeOrNow(t *time.Time) time.Time {
	if t == nil {
		return time.Now()
	}
	return *t
}