package api

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"crm-backend/internal/middleware"
	"crm-backend/internal/model"
	"crm-backend/pkg/password"
	"crm-backend/pkg/response"
)

// CreateUserReq 创建用户请求
type CreateUserReq struct {
	Username   string           `json:"username" binding:"required,min=3,max=50"`
	Password   string           `json:"password" binding:"required,min=6"`
	RealName   string           `json:"real_name"`
	Role       model.Role       `json:"role" binding:"required"`
	DeptID     *uint            `json:"dept_id"`
	GroupID    *uint            `json:"group_id"`
	SuperiorID *uint            `json:"superior_id"` // 归属 / 上级
	Status     model.UserStatus `json:"status"`
}

// UpdateUserReq 更新用户请求
type UpdateUserReq struct {
	RealName   *string           `json:"real_name"`
	Role       *model.Role       `json:"role"`
	DeptID     *uint             `json:"dept_id"`
	GroupID    *uint             `json:"group_id"`
	SuperiorID *uint             `json:"superior_id"`
	Status     *model.UserStatus `json:"status"`
}

// ResetPasswordReq 重置密码请求
type ResetPasswordReq struct {
	NewPassword string `json:"new_password" binding:"required,min=6"`
}

// userList 用户列表
func (r *Router) userList(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	role := c.Query("role")
	deptID := c.Query("dept_id")
	status := c.Query("status")
	superiorID := c.Query("superior_id")
	keyword := strings.TrimSpace(c.Query("keyword"))

	// 排序(防 SQL 注入 — 白名单)
	sortBy := c.DefaultQuery("sort_by", "id")
	sortOrder := strings.ToLower(c.DefaultQuery("sort_order", "desc"))
	allowedSort := map[string]bool{
		"id": true, "username": true, "real_name": true,
		"role": true, "status": true, "last_login_at": true, "created_at": true,
	}
	if !allowedSort[sortBy] {
		sortBy = "id"
	}
	if sortOrder != "asc" && sortOrder != "desc" {
		sortOrder = "desc"
	}

	q := r.DB.Model(&model.User{})
	if role != "" {
		q = q.Where("role = ?", role)
	}
	if deptID != "" {
		if id, err := strconv.ParseUint(deptID, 10, 64); err == nil {
			q = q.Where("dept_id = ?", id)
		}
	}
	if status != "" {
		q = q.Where("status = ?", status)
	}
	if superiorID != "" {
		if id, err := strconv.ParseUint(superiorID, 10, 64); err == nil {
			q = q.Where("superior_id = ?", id)
		}
	}
	if keyword != "" {
		// 账号 / 真实姓名 模糊
		like := "%" + keyword + "%"
		q = q.Where("username ILIKE ? OR real_name ILIKE ?", like, like)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		response.ServerError(c, "查询失败")
		return
	}

	var users []model.User
	if err := q.Preload("Dept").Preload("Group").Preload("Superior").
		Order(sortBy + " " + sortOrder + ", id DESC").
		Offset((page - 1) * pageSize).Limit(pageSize).
		Find(&users).Error; err != nil {
		response.ServerError(c, "查询失败")
		return
	}

	// 隐藏密码
	for i := range users {
		users[i].PasswordHash = ""
	}

	response.OK(c, gin.H{
		"list":      users,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// userCreate 创建用户
func (r *Router) userCreate(c *gin.Context) {
	var req CreateUserReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithStatus(c, http.StatusBadRequest, 400, "参数错误: "+err.Error())
		return
	}

	if !req.Role.IsValid() {
		response.Fail(c, 1, "角色非法")
		return
	}

	// 检查用户名重复(用 Unscoped 包含已软删的用户,避免触发 PG unique 23505)
	var count int64
	if err := r.DB.Unscoped().Model(&model.User{}).Where("username = ?", req.Username).Count(&count).Error; err != nil {
		response.ServerError(c, "检查用户名失败: "+err.Error())
		return
	}
	if count > 0 {
		response.Fail(c, 1, "用户名已被占用(含历史已删除账号)")
		return
	}

	// 哈希密码
	hashed, err := password.Hash(req.Password)
	if err != nil {
		response.Fail(c, 1, "密码处理失败: "+err.Error())
		return
	}

	status := req.Status
	if status == "" {
		status = model.UserStatusEnabled
	}

	user := model.User{
		Username:     req.Username,
		PasswordHash: hashed,
		RealName:     req.RealName,
		Role:         req.Role,
		DeptID:       req.DeptID,
		GroupID:      req.GroupID,
		SuperiorID:   req.SuperiorID,
		Status:       status,
	}

	if err := r.DB.Create(&user).Error; err != nil {
		response.ServerError(c, "创建失败: "+err.Error())
		return
	}

	user.PasswordHash = ""
	response.OK(c, user)
}

// userGet 获取用户详情
func (r *Router) userGet(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.FailWithStatus(c, http.StatusBadRequest, 400, "ID 格式错误")
		return
	}

	var user model.User
	if err := r.DB.Preload("Dept").Preload("Group").Preload("Superior").First(&user, uint(id)).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.NotFound(c, "用户不存在")
			return
		}
		response.ServerError(c, "查询失败")
		return
	}

	user.PasswordHash = ""
	response.OK(c, user)
}

// userUpdate 更新用户
func (r *Router) userUpdate(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.FailWithStatus(c, http.StatusBadRequest, 400, "ID 格式错误")
		return
	}

	var req UpdateUserReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithStatus(c, http.StatusBadRequest, 400, "参数错误: "+err.Error())
		return
	}

	var user model.User
	if err := r.DB.First(&user, uint(id)).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.NotFound(c, "用户不存在")
			return
		}
		response.ServerError(c, "查询失败")
		return
	}

	updates := map[string]interface{}{}
	if req.RealName != nil {
		updates["real_name"] = *req.RealName
	}
	if req.Role != nil {
		if !req.Role.IsValid() {
			response.Fail(c, 1, "角色非法")
			return
		}
		updates["role"] = *req.Role
	}
	if req.DeptID != nil {
		updates["dept_id"] = *req.DeptID
	}
	if req.GroupID != nil {
		updates["group_id"] = *req.GroupID
	}
	if req.SuperiorID != nil {
		// 不允许把上级设为自己（避免自循环）
		if *req.SuperiorID == uint(id) {
			response.Fail(c, 1, "上级不能是自己")
			return
		}
		updates["superior_id"] = *req.SuperiorID
	}
	if req.Status != nil {
		updates["status"] = *req.Status
	}

	if len(updates) == 0 {
		response.Fail(c, 1, "无更新字段")
		return
	}

	if err := r.DB.Model(&user).Updates(updates).Error; err != nil {
		response.ServerError(c, "更新失败")
		return
	}

	r.DB.Preload("Dept").Preload("Group").Preload("Superior").First(&user, user.ID)
	user.PasswordHash = ""
	response.OK(c, user)
}

// userDelete 删除用户（软删除）
func (r *Router) userDelete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.FailWithStatus(c, http.StatusBadRequest, 400, "ID 格式错误")
		return
	}

	// 不允许删除自己
	if uint(id) == getUserIDFromContext(c) {
		response.Fail(c, 1, "不能删除自己")
		return
	}

	if err := r.DB.Delete(&model.User{}, uint(id)).Error; err != nil {
		response.ServerError(c, "删除失败")
		return
	}

	response.OK(c, nil)
}

// userOptions 拉取所有启用用户的精简列表(供上级筛选 / 新建编辑时下拉用)
// 与 userList 不同:不分页、不带敏感字段、只返回 enabled,保证选中值永远能在候选里找到 label
func (r *Router) userOptions(c *gin.Context) {
	type item struct {
		ID       uint            `json:"id"`
		Username string          `json:"username"`
		RealName string          `json:"real_name"`
		Role     model.Role      `json:"role"`
		Status   model.UserStatus `json:"status"`
	}
	var users []model.User
	if err := r.DB.
		Select("id, username, real_name, role, status").
		Where("status = ?", model.UserStatusEnabled).
		Order("real_name ASC, username ASC, id ASC").
		Find(&users).Error; err != nil {
		response.ServerError(c, "查询失败")
		return
	}
	items := make([]item, len(users))
	for i, u := range users {
		items[i] = item{ID: u.ID, Username: u.Username, RealName: u.RealName, Role: u.Role, Status: u.Status}
	}
	response.OK(c, items)
}

// userResetPassword 重置密码
func (r *Router) userResetPassword(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.FailWithStatus(c, http.StatusBadRequest, 400, "ID 格式错误")
		return
	}

	var req ResetPasswordReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithStatus(c, http.StatusBadRequest, 400, "参数错误: "+err.Error())
		return
	}

	var user model.User
	if err := r.DB.First(&user, uint(id)).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.NotFound(c, "用户不存在")
			return
		}
		response.ServerError(c, "查询失败")
		return
	}

	hashed, err := password.Hash(req.NewPassword)
	if err != nil {
		response.Fail(c, 1, "密码处理失败")
		return
	}

	if err := r.DB.Model(&user).Update("password_hash", hashed).Error; err != nil {
		response.ServerError(c, "重置失败")
		return
	}

	response.OKWithMessage(c, "密码已重置", nil)
}

// ============ P11 follow 端接单开关 / 个人设置 ============

// UserMeSettingsReq 个人设置请求(目前只支持接单开关)
type UserMeSettingsReq struct {
	AcceptOrders *bool `json:"accept_orders"` // 允许 null(不改)
}

// userMeSettingsGet 获取个人设置(任意登录用户可读自己的)
func (r *Router) userMeSettingsGet(c *gin.Context) {
	userID := middleware.GetCurrentUserID(c)
	var u model.User
	if err := r.DB.Select("accept_orders").First(&u, userID).Error; err != nil {
		response.ServerError(c, "查询失败")
		return
	}
	response.OK(c, gin.H{
		"accept_orders": u.AcceptOrders,
	})
}

// userMeSettingsUpdate 更新个人设置
//   - 任意登录用户可改自己的接单开关
//   - 改完后同步 Hub 缓存,让该用户所有在线客户端立刻生效(无需重连 WS)
func (r *Router) userMeSettingsUpdate(c *gin.Context) {
	userID := middleware.GetCurrentUserID(c)
	var req UserMeSettingsReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithStatus(c, http.StatusBadRequest, 400, "参数错误: "+err.Error())
		return
	}
	if req.AcceptOrders == nil {
		response.Fail(c, 1, "无更新字段")
		return
	}
	if err := r.DB.Model(&model.User{}).Where("id = ?", userID).
		Update("accept_orders", *req.AcceptOrders).Error; err != nil {
		response.ServerError(c, "更新失败")
		return
	}
	// 同步 WS Hub 缓存,让该用户所有在线客户端立刻生效
	if r.WSHub != nil {
		r.WSHub.UpdateClientAcceptOrders(userID, *req.AcceptOrders)
	}
	response.OKWithMessage(c, "已更新", gin.H{
		"accept_orders": *req.AcceptOrders,
	})
}