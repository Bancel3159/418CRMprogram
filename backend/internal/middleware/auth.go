package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"

	"crm-backend/internal/model"
	"crm-backend/pkg/jwt"
	"crm-backend/pkg/response"
)

// ContextKey context key
const (
	CtxUserID   = "userID"
	CtxUsername = "username"
	CtxRole     = "role"
	CtxDeptID   = "deptID"
)

// JWTAuth JWT 认证中间件
func JWTAuth(jwtMgr *jwt.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			response.Unauthorized(c, "缺少 Authorization 头")
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			response.Unauthorized(c, "Authorization 格式错误")
			c.Abort()
			return
		}

		claims, err := jwtMgr.ParseAccessToken(parts[1])
		if err != nil {
			switch err {
			case jwt.ErrTokenExpired:
				response.Unauthorized(c, "token 已过期")
			default:
				response.Unauthorized(c, "token 无效")
			}
			c.Abort()
			return
		}

		// 注入用户信息到 context
		c.Set(CtxUserID, claims.UserID)
		c.Set(CtxUsername, claims.Username)
		c.Set(CtxRole, claims.Role)
		c.Set(CtxDeptID, claims.DeptID)

		c.Next()
	}
}

// RequireRole 要求指定角色（任一即可）
func RequireRole(roles ...model.Role) gin.HandlerFunc {
	allowed := make(map[model.Role]bool, len(roles))
	for _, r := range roles {
		allowed[r] = true
	}
	return func(c *gin.Context) {
		roleStr, _ := c.Get(CtxRole)
		role, ok := roleStr.(string)
		if !ok {
			response.Forbidden(c, "无法获取角色信息")
			c.Abort()
			return
		}

		if !allowed[model.Role(role)] {
			response.Forbidden(c, "权限不足")
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequireAdmin 仅 admin
func RequireAdmin() gin.HandlerFunc {
	return RequireRole(model.RoleAdmin)
}

// RequireManager 主管及以上（含 admin / A 主管 / B 主管）
func RequireManager() gin.HandlerFunc {
	return RequireRole(model.RoleAdmin, model.RoleAManager, model.RoleBManager)
}

// RequireLead 组长及以上（含主管）
func RequireLead() gin.HandlerFunc {
	return RequireRole(model.RoleAdmin, model.RoleAManager, model.RoleALead, model.RoleBManager, model.RoleBLead)
}

// RequireWebAccess 有 Web 端访问权限的角色（admin / 主管 / 组长）
func RequireWebAccess() gin.HandlerFunc {
	return RequireLead()
}

// GetCurrentUserID 从 context 获取当前用户 ID
func GetCurrentUserID(c *gin.Context) uint {
	if v, ok := c.Get(CtxUserID); ok {
		if id, ok := v.(uint); ok {
			return id
		}
	}
	return 0
}

// GetCurrentRole 从 context 获取当前角色
func GetCurrentRole(c *gin.Context) model.Role {
	if v, ok := c.Get(CtxRole); ok {
		if r, ok := v.(string); ok {
			return model.Role(r)
		}
	}
	return ""
}

// GetCurrentDeptID 从 context 获取当前部门 ID
func GetCurrentDeptID(c *gin.Context) *uint {
	if v, ok := c.Get(CtxDeptID); ok {
		if id, ok := v.(*uint); ok {
			return id
		}
	}
	return nil
}