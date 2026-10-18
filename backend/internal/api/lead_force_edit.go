package api

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"crm-backend/internal/middleware"
	"crm-backend/internal/model"
	"crm-backend/internal/service"
	"crm-backend/internal/ws"
	"crm-backend/pkg/response"
)

// ForceEditReq 全字段强制编辑请求(P13 admin only)
//
// 设计要点:
//   - 所有字段都是 *T 指针,nil = 本次不改(区分「不改」和「改成零值」)
//   - 系统生成字段(created_at / updated_at / deleted_at / paid_auto / is_paid)不开放
//   - *_at 时间字段用 *time.Time,传 null 显式清空
type ForceEditReq struct {
	// 归属(允许流转)
	GameID     *uint   `json:"game_id"`
	CreatorID  *uint   `json:"creator_id"`
	Channel    *string `json:"channel"`
	Status     *string `json:"status"` // pending / claimed / archived
	ClaimedBy  *uint   `json:"claimed_by"`
	ReleasedBy *uint   `json:"released_by"`

	// 13 核心业务字段
	PlayerUsername  *string `json:"player_username"`
	PlayerGameName  *string `json:"player_game_name"`
	PlayerID        *string `json:"player_id"`
	Region          *string `json:"region"`
	PlayerCountry   *string `json:"player_country"`
	PlayerAge       *int    `json:"player_age"`
	IsTarget        *string `json:"is_target"` // target / not_target
	Community       *string `json:"community"`
	TokenCode       *string `json:"token_code"`
	Screenshots     []string `json:"screenshots"`
	Remark          *string `json:"remark"`
	Extension       map[string]interface{} `json:"extension"`

	// 付费标记
	PaidMark *bool `json:"paid_mark"`

	// 流失 / 激活(P11 规则引擎状态,允许人工修正)
	IsLost             *bool      `json:"is_lost"`
	LostReason         *string    `json:"lost_reason"`
	ReactivatedAt      *time.Time `json:"reactivated_at"`
	LastPlayerActiveAt *time.Time `json:"last_player_active_at"`

	// 认领 / 释放时间
	ClaimedAt  *time.Time `json:"claimed_at"`
	ReleasedAt *time.Time `json:"released_at"`
	PaidMarkedAt *time.Time `json:"paid_marked_at"`
}

// forceEditLead 全字段强制编辑(P13)
//
// URL:  PUT /api/v1/leads/:id/force-edit
// 权限: 仅 admin
// 用途: 后期玩家流转 / 数据调整 —— 覆盖常规 leadUpdate 的业务校验限制
//
// 与常规 leadUpdate 的区别:
//   - 不校验 claimed_by 是否属于 B 部、不校验游戏归属
//   - 允许改 player_id / game_id(会触发玩家聚合身份变化)
//   - 允许直接改 is_lost / lost_reason / last_player_active_at 等规则引擎状态
//   - 全量写审计 before/after 差异
func (r *Router) forceEditLead(c *gin.Context) {
	leadID, err := strconv.Atoi(c.Param("id"))
	if err != nil || leadID <= 0 {
		response.FailWithStatus(c, http.StatusBadRequest, 400, "无效的 lead id")
		return
	}

	// 取改动前的完整快照(审计用)
	var before model.Lead
	if err := r.DB.First(&before, uint(leadID)).Error; err != nil {
		response.NotFound(c, "线索不存在")
		return
	}

	var req ForceEditReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithStatus(c, http.StatusBadRequest, 400, "参数错误: "+err.Error())
		return
	}

	// 组装 updates map(只放本次要改的字段)
	updates := map[string]interface{}{}

	if req.GameID != nil {
		if *req.GameID == 0 {
			response.FailWithStatus(c, http.StatusBadRequest, 400, "game_id 不能为 0")
			return
		}
		// 校验游戏存在
		var g model.Game
		if err := r.DB.First(&g, *req.GameID).Error; err != nil {
			response.FailWithStatus(c, http.StatusBadRequest, 400, fmt.Sprintf("游戏不存在 (id=%d)", *req.GameID))
			return
		}
		updates["game_id"] = *req.GameID
	}
	if req.CreatorID != nil {
		updates["creator_id"] = *req.CreatorID
	}
	if req.Channel != nil {
		updates["channel"] = *req.Channel
	}
	if req.Status != nil {
		if !model.LeadStatus(*req.Status).IsValid() {
			response.FailWithStatus(c, http.StatusBadRequest, 400, "status 非法(pending/claimed/archived)")
			return
		}
		updates["status"] = *req.Status
	}
	if req.ClaimedBy != nil {
		updates["claimed_by"] = *req.ClaimedBy
	}
	if req.ReleasedBy != nil {
		updates["released_by"] = *req.ReleasedBy
	}

	if req.PlayerUsername != nil {
		updates["player_username"] = *req.PlayerUsername
	}
	if req.PlayerGameName != nil {
		updates["player_game_name"] = *req.PlayerGameName
	}
	if req.PlayerID != nil {
		updates["player_id"] = *req.PlayerID
	}
	if req.Region != nil {
		updates["region"] = *req.Region
	}
	if req.PlayerCountry != nil {
		updates["player_country"] = *req.PlayerCountry
	}
	if req.PlayerAge != nil {
		if *req.PlayerAge < 0 || *req.PlayerAge > 150 {
			response.FailWithStatus(c, http.StatusBadRequest, 400, "player_age 超出合理范围 (0-150)")
			return
		}
		updates["player_age"] = *req.PlayerAge
	}
	if req.IsTarget != nil {
		if !model.IsTarget(*req.IsTarget).IsValid() {
			response.FailWithStatus(c, http.StatusBadRequest, 400, "is_target 非法(target/not_target)")
			return
		}
		updates["is_target"] = *req.IsTarget
	}
	if req.Community != nil {
		updates["community"] = *req.Community
	}
	if req.TokenCode != nil {
		updates["token_code"] = *req.TokenCode
	}
	if req.Screenshots != nil {
		updates["screenshots"] = model.Strings(req.Screenshots)
	}
	if req.Remark != nil {
		updates["remark"] = *req.Remark
	}
	if req.Extension != nil {
		updates["extension"] = model.JSONB(req.Extension)
	}

	if req.PaidMark != nil {
		updates["paid_mark"] = *req.PaidMark
		// 人工改 paid_mark 时同步标记人 / 标记时间
		uid := middleware.GetCurrentUserID(c)
		updates["paid_marked_by"] = uid
		updates["paid_marked_at"] = time.Now()
		if !*req.PaidMark {
			updates["paid_marked_by"] = nil
			updates["paid_marked_at"] = nil
		}
	}

	if req.IsLost != nil {
		updates["is_lost"] = *req.IsLost
		if !*req.IsLost {
			updates["lost_at"] = nil
			updates["lost_reason"] = ""
		} else if req.LostReason != nil {
			updates["lost_at"] = time.Now()
			updates["lost_reason"] = *req.LostReason
		}
	} else if req.LostReason != nil {
		updates["lost_reason"] = *req.LostReason
	}
	if req.ReactivatedAt != nil {
		updates["reactivated_at"] = req.ReactivatedAt
	}
	if req.LastPlayerActiveAt != nil {
		updates["last_player_active_at"] = req.LastPlayerActiveAt
	}

	if req.ClaimedAt != nil {
		updates["claimed_at"] = req.ClaimedAt
	}
	if req.ReleasedAt != nil {
		updates["released_at"] = req.ReleasedAt
	}
	if req.PaidMarkedAt != nil {
		updates["paid_marked_at"] = req.PaidMarkedAt
	}

	if len(updates) == 0 {
		response.FailWithStatus(c, http.StatusBadRequest, 400, "没有要修改的字段")
		return
	}

	// 落库
	tx := r.DB.Model(&model.Lead{}).Where("id = ?", leadID).Updates(updates)
	if tx.Error != nil {
		response.ServerError(c, "更新失败: "+tx.Error.Error())
		return
	}

	// 取改动后的快照
	var after model.Lead
	if err := r.DB.First(&after, uint(leadID)).Error; err != nil {
		response.ServerError(c, "回读失败: "+err.Error())
		return
	}

	// 审计:只记真正变化的字段(逐字段 diff,避免噪音)
	changed := diffLeadFields(&before, &after)
	service.NewAuditRecorder(r.DB).RecordAction(
		c,
		model.AuditUpdate,
		model.AuditTargetLead,
		&after.ID,
		after.PlayerID,
		toAuditJSON(before),
		toAuditJSON(after),
		model.JSONB{"changed_fields": changed, "force_edit": true},
	)

	// 广播池变更(如果这条 lead 在公开池,状态/游戏变了要同步刷新)
	if r.WSHub != nil && before.Status == model.LeadStatusPending {
		r.WSHub.BroadcastToFilteredChannel("pool:broadcast", ws.MessageTypeBroadcast, "lead.updated", gin.H{
			"id":    after.ID,
			"label": after.PlayerUsername,
		}, shouldSkipPoolBroadcast)
	}

	response.OK(c, gin.H{
		"lead":            after,
		"changed_fields":  changed,
	})
}

// diffLeadFields 对比 before/after,返回发生变化的字段名列表
func diffLeadFields(before, after *model.Lead) []string {
	changed := make([]string, 0)

	addIfDiff := func(name string, a, b interface{}) {
		if fmt.Sprintf("%v", a) != fmt.Sprintf("%v", b) {
			changed = append(changed, name)
		}
	}

	addIfDiff("game_id", before.GameID, after.GameID)
	addIfDiff("creator_id", before.CreatorID, after.CreatorID)
	addIfDiff("channel", before.Channel, after.Channel)
	addIfDiff("player_username", before.PlayerUsername, after.PlayerUsername)
	addIfDiff("player_game_name", before.PlayerGameName, after.PlayerGameName)
	addIfDiff("player_id", before.PlayerID, after.PlayerID)
	addIfDiff("region", before.Region, after.Region)
	addIfDiff("player_country", before.PlayerCountry, after.PlayerCountry)
	addIfDiff("player_age", before.PlayerAge, after.PlayerAge)
	addIfDiff("is_target", before.IsTarget, after.IsTarget)
	addIfDiff("community", before.Community, after.Community)
	addIfDiff("token_code", before.TokenCode, after.TokenCode)
	addIfDiff("screenshots", before.Screenshots, after.Screenshots)
	addIfDiff("status", before.Status, after.Status)
	addIfDiff("claimed_by", ptrVal(before.ClaimedBy), ptrVal(after.ClaimedBy))
	addIfDiff("claimed_at", ptrTime(before.ClaimedAt), ptrTime(after.ClaimedAt))
	addIfDiff("released_by", ptrVal(before.ReleasedBy), ptrVal(after.ReleasedBy))
	addIfDiff("released_at", ptrTime(before.ReleasedAt), ptrTime(after.ReleasedAt))
	addIfDiff("extension", before.Extension, after.Extension)
	addIfDiff("remark", before.Remark, after.Remark)
	addIfDiff("paid_mark", before.PaidMark, after.PaidMark)
	addIfDiff("paid_marked_by", ptrVal(before.PaidMarkedBy), ptrVal(after.PaidMarkedBy))
	addIfDiff("paid_marked_at", ptrTime(before.PaidMarkedAt), ptrTime(after.PaidMarkedAt))
	addIfDiff("is_lost", before.IsLost, after.IsLost)
	addIfDiff("lost_at", ptrTime(before.LostAt), ptrTime(after.LostAt))
	addIfDiff("lost_reason", before.LostReason, after.LostReason)
	addIfDiff("reactivated_at", ptrTime(before.ReactivatedAt), ptrTime(after.ReactivatedAt))
	addIfDiff("last_player_active_at", ptrTime(before.LastPlayerActiveAt), ptrTime(after.LastPlayerActiveAt))

	return changed
}

// ptrVal *uint → 可比较的值(nil → 0)
func ptrVal(p *uint) uint {
	if p == nil {
		return 0
	}
	return *p
}

// ptrTime *time.Time → RFC3339 字符串(nil → "")
func ptrTime(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}

// toAuditJSON 转成审计用的 map(排除 gorm:"-" 的计算字段和 User-Agent 等)
func toAuditJSON(l model.Lead) model.JSONB {
	return model.JSONB{
		"id":                   l.ID,
		"game_id":              l.GameID,
		"creator_id":           l.CreatorID,
		"channel":              l.Channel,
		"player_username":      l.PlayerUsername,
		"player_game_name":     l.PlayerGameName,
		"player_id":            l.PlayerID,
		"region":               l.Region,
		"player_country":       l.PlayerCountry,
		"player_age":           l.PlayerAge,
		"is_target":            string(l.IsTarget),
		"community":            l.Community,
		"token_code":           l.TokenCode,
		"status":               string(l.Status),
		"claimed_by":           ptrVal(l.ClaimedBy),
		"claimed_at":           ptrTime(l.ClaimedAt),
		"released_by":          ptrVal(l.ReleasedBy),
		"released_at":          ptrTime(l.ReleasedAt),
		"extension":            l.Extension,
		"remark":               l.Remark,
		"paid_mark":            l.PaidMark,
		// 派生字段也导出 —— diffLeadFields 会把它们计入 changed_fields,
		// 快照里没有值的话前端「对比」表会显示成「— → —」看不懂
		"paid_marked_by":        ptrVal(l.PaidMarkedBy),
		"paid_marked_at":        ptrTime(l.PaidMarkedAt),
		"is_lost":               l.IsLost,
		"lost_at":               ptrTime(l.LostAt),
		"lost_reason":           l.LostReason,
		"reactivated_at":        ptrTime(l.ReactivatedAt),
		"last_player_active_at": ptrTime(l.LastPlayerActiveAt),
	}
}