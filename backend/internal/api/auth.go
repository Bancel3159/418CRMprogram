package api

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"crm-backend/internal/model"
	"crm-backend/internal/service"
	"crm-backend/pkg/password"
	"crm-backend/pkg/response"
	"crm-backend/pkg/turnstile"
)

// LoginReq 登录请求
type LoginReq struct {
	Username        string `json:"username" binding:"required"`
	Password        string `json:"password" binding:"required"`
	TurnstileToken  string `json:"turnstile_token"`
}

// LoginResp 登录响应
type LoginResp struct {
	AccessToken  string      `json:"access_token"`
	RefreshToken string      `json:"refresh_token"`
	ExpiresAt    time.Time   `json:"expires_at"`
	User         *model.User `json:"user"`
}

// login 登录
func (r *Router) login(c *gin.Context) {
	var req LoginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithStatus(c, http.StatusBadRequest, 400, "请求参数错误: "+err.Error())
		return
	}

	audit := service.NewAuditRecorder(r.DB)

	// 1. Turnstile 验证（开发环境 secret 未配置则跳过）
	if !r.Turnstile.IsSkipped() {
		result, err := r.Turnstile.Verify(req.TurnstileToken, c.ClientIP())
		if err != nil {
			audit.RecordUnauthLogin(c, false, req.Username, "turnstile: "+err.Error())
			response.Fail(c, 1, "Turnstile 验证失败: "+err.Error())
			return
		}
		if !result.Success {
			audit.RecordUnauthLogin(c, false, req.Username, fmt.Sprintf("turnstile: %v", result.ErrorCodes))
			response.Fail(c, 1, "Turnstile 验证未通过: "+fmt.Sprintf("%v", result.ErrorCodes))
			return
		}
	}

	// 2. 查询用户
	var user model.User
	if err := r.DB.Where("username = ?", req.Username).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			audit.RecordUnauthLogin(c, false, req.Username, "用户不存在")
			response.Fail(c, 1, "用户名或密码错误")
			return
		}
		response.ServerError(c, "查询失败")
		return
	}

	// 3. 校验密码
	if !password.Verify(user.PasswordHash, req.Password) {
		audit.RecordUnauthLogin(c, false, req.Username, "密码错误")
		response.Fail(c, 1, "用户名或密码错误")
		return
	}

	// 4. 检查状态
	if user.Status != model.UserStatusEnabled {
		audit.RecordUnauthLogin(c, false, req.Username, "账号已停用")
		response.Fail(c, 1, "账号已停用")
		return
	}

	// 5. 签发 JWT
	accessToken, expiresAt, err := r.JWT.GenerateAccessToken(user.ID, user.Username, string(user.Role), user.DeptID)
	if err != nil {
		response.ServerError(c, "token 签发失败")
		return
	}
	refreshToken, _, err := r.JWT.GenerateRefreshToken(user.ID, user.Username, string(user.Role), user.DeptID)
	if err != nil {
		response.ServerError(c, "refresh token 签发失败")
		return
	}

	// 6. 更新最后登录时间
	now := time.Now()
	r.DB.Model(&user).Update("last_login_at", &now)

	// 7. 隐藏敏感字段
	user.PasswordHash = ""

	// 8. 记录登录成功(注入用户信息后手动写,因为登录成功时 gin ctx 还没有 userID)
	loginLog := &model.AuditLog{
		UserID:      &user.ID,
		Username:    user.Username,
		Role:        string(user.Role),
		Action:      model.AuditLogin,
		TargetType:  model.AuditTargetSystem,
		TargetLabel: user.Username,
		IP:          c.ClientIP(),
		UserAgent:   c.GetHeader("User-Agent"),
		Status:      model.AuditStatusSuccess,
	}
	audit.RecordSync(c.Request.Context(), loginLog)

	response.OK(c, LoginResp{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    expiresAt,
		User:         &user,
	})
}

// RefreshReq 刷新请求
type RefreshReq struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// refresh 刷新 access token
func (r *Router) refresh(c *gin.Context) {
	var req RefreshReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithStatus(c, http.StatusBadRequest, 400, "请求参数错误")
		return
	}

	claims, err := r.JWT.ParseRefreshToken(req.RefreshToken)
	if err != nil {
		response.Unauthorized(c, "refresh token 无效或已过期")
		return
	}

	// 重新查询用户（确保账号仍可用）
	var user model.User
	if err := r.DB.First(&user, claims.UserID).Error; err != nil {
		response.Unauthorized(c, "用户不存在")
		return
	}
	if user.Status != model.UserStatusEnabled {
		response.Unauthorized(c, "账号已暂停")
		return
	}

	accessToken, expiresAt, err := r.JWT.GenerateAccessToken(user.ID, user.Username, string(user.Role), user.DeptID)
	if err != nil {
		response.ServerError(c, "token 签发失败")
		return
	}

	response.OK(c, gin.H{
		"access_token": accessToken,
		"expires_at":   expiresAt,
	})
}

// logout 登出（前后端均可清掉 token；如需 token 黑名单可加 Redis）
func (r *Router) logout(c *gin.Context) {
	// TODO: 可选 - 将 token 加入 Redis 黑名单（带 TTL）
	// 当前实现：前端清掉 token 即可
	response.OKWithMessage(c, "已登出", nil)
}

// GetCurrentUser 获取当前登录用户信息
func (r *Router) getCurrentUser(c *gin.Context) {
	userID := getUserIDFromContext(c)

	var user model.User
	if err := r.DB.Preload("Dept").Preload("Group").First(&user, userID).Error; err != nil {
		response.NotFound(c, "用户不存在")
		return
	}

	user.PasswordHash = ""
	response.OK(c, user)
}

// getUserIDFromContext 从 gin context 获取用户 ID
func getUserIDFromContext(c *gin.Context) uint {
	if v, ok := c.Get("userID"); ok {
		if id, ok := v.(uint); ok {
			return id
		}
	}
	return 0
}

// 占位实现，确保 import 不报错
var _ turnstile.VerifyResponse