package api

import (
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"crm-backend/internal/config"
	"crm-backend/internal/middleware"
	"crm-backend/internal/model"
	"crm-backend/internal/service"
	"crm-backend/internal/storage"
	"crm-backend/internal/ws"
	"crm-backend/pkg/jwt"
	"crm-backend/pkg/turnstile"
)

// Router 路由器
type Router struct {
	*gin.Engine
	DB        *gorm.DB
	RDB       *redis.Client
	Cfg       *config.Config
	JWT       *jwt.Manager
	Turnstile *turnstile.Client
	Uploader  *storage.Uploader
	WSHub     *ws.Hub
	Alerter   *service.AlertNotifier
}

// NewRouter 创建路由
func NewRouter(cfg *config.Config, db *gorm.DB, rdb *redis.Client, hub *ws.Hub, uploader *storage.Uploader) *Router {
	gin.SetMode(cfg.Server.Mode)

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.CORS())
	r.Use(middleware.Logger())

	jwtMgr := jwt.NewManager(cfg.JWT.Secret, cfg.JWT.AccessTokenTTL, cfg.JWT.RefreshTokenTTL)
	tsClient := turnstile.NewClient(cfg.Turnstile.SecretKey, cfg.Turnstile.VerifyURL)

	router := &Router{
		Engine:    r,
		DB:        db,
		RDB:       rdb,
		Cfg:       cfg,
		JWT:       jwtMgr,
		Turnstile: tsClient,
		Uploader:  uploader,
		WSHub:     hub,
		Alerter:   service.NewAlertNotifier(db, hub),
	}

	router.setupRoutes()
	return router
}

func (r *Router) setupRoutes() {
	// 健康检查
	r.GET("/health", r.healthCheck)

	// 静态文件服务（uploads）
	if r.Uploader != nil {
		r.Static("/uploads", "./uploads")
	}

	// API v1 路由组
	v1 := r.Group("/api/v1")
	{
		// ========== 公开路由（无需鉴权）==========
		auth := v1.Group("/auth")
		{
			auth.POST("/login", r.login)
			auth.POST("/refresh", r.refresh)
		}

		// ========== 需要鉴权的路由 ==========
		authorized := v1.Group("")
		authorized.Use(middleware.JWTAuth(r.JWT))
		{
			// 当前用户
			authorized.GET("/auth/me", r.getCurrentUser)
			authorized.POST("/auth/logout", r.logout)

			// 个人设置(任意登录用户可读/改自己的)
			authorized.GET("/users/me/settings", r.userMeSettingsGet)
			authorized.PUT("/users/me/settings", r.userMeSettingsUpdate)

			// P11 跟进提醒
			authorized.GET("/leads/reminders/today", r.remindersToday)
			// P11 写跟进(写在 leads/:id 组的 POST 上,这里单独走)
			authorized.POST("/leads/:id/follow", r.leadCreateFollow)
			authorized.POST("/leads/:id/mark-lost", r.leadMarkLostManual)
			authorized.POST("/leads/:id/mark-active", r.leadMarkActiveManual)
			// P13 全字段强制编辑(admin only,玩家流转 / 数据调整)
			authorized.PUT("/leads/:id/force-edit", middleware.RequireAdmin(), r.forceEditLead)
			// P11 系统配置(follow)
			authorized.GET("/system/configs/follow", r.systemConfigFollowGet)
			authorized.PUT("/system/configs/follow", r.systemConfigFollowUpdate)
			// P12 实时通知配置
			authorized.GET("/system/configs/realtime", r.systemConfigRealtimeGet)
			authorized.PUT("/system/configs/realtime", r.systemConfigRealtimeUpdate)
			// P11 手动触发扫描(admin)
			authorized.POST("/system/follow/scan", r.followDailyScanTrigger)

			// 用户管理（admin）
			users := authorized.Group("/users")
			users.Use(middleware.RequireAdmin())
			{
				users.GET("", r.listUsers)
				users.POST("", r.createUser)
				users.GET("/:id", r.getUser)
				users.PUT("/:id", r.updateUser)
				users.DELETE("/:id", r.deleteUser)
				users.POST("/:id/reset-password", r.resetUserPassword)
				users.GET("/options", r.usersOptions)
			}

			// 部门管理（admin）
			depts := authorized.Group("/departments")
			depts.Use(middleware.RequireAdmin())
			{
				depts.GET("", r.listDepartments)
				depts.POST("", r.createDepartment)
				depts.GET("/:id", r.getDepartment)
				depts.PUT("/:id", r.updateDepartment)
				depts.DELETE("/:id", r.deleteDepartment)
			}

			// ============ P2：基础数据管理 ============

			// 游戏管理（GET 列表对所有登录用户开放,CRUD 仅 admin）
			authorized.GET("/games", r.gameList)
			authorized.GET("/games/:id", r.gameGet)
			games := authorized.Group("/games")
			games.Use(middleware.RequireAdmin())
			{
				games.POST("", r.gameCreate)
				games.PUT("/:id", r.gameUpdate)
				games.DELETE("/:id", r.gameDelete)
			}

			// 字典管理（GET 列表对所有登录用户开放,CRUD 仅 admin）
			authorized.GET("/dicts/categories", r.dictCategories)
			authorized.GET("/dicts", r.dictList)
			dicts := authorized.Group("/dicts")
			dicts.Use(middleware.RequireAdmin())
			{
				dicts.POST("", r.dictCreate)
				dicts.GET("/:id", r.dictGet)
				dicts.PUT("/:id", r.dictUpdate)
				dicts.DELETE("/:id", r.dictDelete)
			}

			// 广播模板（admin）
			tpls := authorized.Group("/broadcast-templates")
			tpls.Use(middleware.RequireAdmin())
			{
				tpls.GET("", r.broadcastTemplateList)
				tpls.POST("", r.broadcastTemplateCreate)
				tpls.GET("/:id", r.broadcastTemplateGet)
				tpls.PUT("/:id", r.broadcastTemplateUpdate)
				tpls.DELETE("/:id", r.broadcastTemplateDelete)
				tpls.POST("/render", r.renderTemplate)
			}

			// ============ P3：线索核心 ============

			// 文件上传（所有登录用户）
			authorized.POST("/upload", r.uploadFiles)
			authorized.POST("/upload/single", r.uploadSingleFile)

			// 线索管理
			leads := authorized.Group("/leads")
			{
				// 公开池（B 部全员 + admin）
				leads.GET("/pool", middleware.RequireRole(
					model.RoleAdmin,
					model.RoleBManager, model.RoleBLead, model.RoleBStaff,
				), r.poolList)

				// 线索列表 / 创建 / 详情
				leads.GET("", r.leadList)
				leads.POST("", r.leadCreate)
				leads.GET("/:id", r.leadGet)
				leads.PUT("/:id", r.leadUpdate)
				leads.DELETE("/:id", r.leadDelete)

				// 业务动作
				leads.POST("/:id/claim", r.leadClaim)
				leads.POST("/:id/release", r.leadRelease)
				leads.POST("/:id/archive", r.leadArchive)

				// 付费标记（admin / 主管 / 组长 可调）
				leads.POST("/:id/mark-paid", r.leadMarkPaid)
				leads.POST("/:id/unmark-paid", r.leadUnmarkPaid)

				// P9:跟进记录(挂在 lead 下,聚合到玩家页时间线)
				leads.POST("/:id/follow-records", r.followRecordCreate)
				leads.GET("/:id/follow-records", r.followRecordList)
			}

			// P9:跟进记录删除(独立路由,因为 :id 是 follow-record id,不是 lead id)
			authorized.DELETE("/follow-records/:id", r.followRecordDelete)

			// P9:玩家列表(按 player_id+game_id 聚合)
			players := authorized.Group("/players")
			{
				players.GET("", r.playerList)
				players.GET("/kpi", r.playerKPI)
				players.GET("/:player_id/:game_id", r.playerDetail)
				// P12 玩家活跃时间更新(手动 / 合作方 / webhook 携带)
				players.POST("/:player_id/active", r.playerActiveUpdate)
			}

			// ============ P6：报表 / 看板 ============
			// 所有已登录用户都可看（含员工，主管看本部门，admin 看全公司）
			stats := authorized.Group("/stats")
			{
				stats.GET("/overview", r.statsOverview)               // 4 卡总览
				stats.GET("/trend", r.statsTrend)                   // 拉新趋势
				stats.GET("/by-game", r.statsByGame)                 // 按游戏
				stats.GET("/by-channel", r.statsByChannel)           // 按渠道
				stats.GET("/by-country", r.statsByCountry)           // 按国家
				stats.GET("/funnel", r.statsFunnel)                 // 转化漏斗
				stats.GET("/top-staff", r.statsTopStaff)             // 拉新排行
				stats.GET("/top-maintainer", r.statsTopMaintainer)   // 维护排行
				stats.GET("/hourly", r.statsHourlyDistribution)      // 时段分布
				stats.GET("/mine", r.statsMine)                    // A 员工个人拉新看板(只看自己)
				stats.GET("/team", r.statsTeam)                    // 主管/组长看板:可选全体/某个员工
				stats.GET("/team/members", r.statsTeamMembers)     // A 部成员 + 各自拉新数(做下拉选项)
				stats.GET("/follow/mine", r.statsFollowMine)     // follow 端我的维护看板(只统计自己认领的 lead)
			}

			// ============ P8：操作审计 ============
			audit := authorized.Group("/audit")
			audit.Use(middleware.RequireAdmin())
			{
				audit.GET("/logs", r.auditList)
				audit.GET("/logs/:id", r.auditGet)
				audit.GET("/stats", r.auditStats)
				audit.GET("/write-stats", r.auditWriteStats)
			}

			// 注册扩展字段(所有登录用户可读,管理操作仅 admin) 
			authorized.GET("/lead-extension-fields", r.extensionFieldList)
			leadExt := authorized.Group("/lead-extension-fields")
			leadExt.Use(middleware.RequireAdmin())
			{
				leadExt.GET("/admin", r.extensionFieldAdminList)
				leadExt.POST("", r.extensionFieldCreate)
				leadExt.PUT("/:id", r.extensionFieldUpdate)
				leadExt.DELETE("/:id", r.extensionFieldDelete)
			}

			// ============ P10：列表视图（玩家 / 订单字段可配置）============
			// GET 列表任何登录用户可用(列表页要拉默认视图);CRUD 仅 admin
			authorized.GET("/list-views", r.listViewsList)
			authorized.GET("/list-views/available-fields", r.listViewsAvailableFields)
			listViews := authorized.Group("/list-views")
			listViews.Use(middleware.RequireAdmin())
			{
				listViews.GET("/:id", r.listViewsGet)
				listViews.POST("", r.listViewsCreate)
				listViews.PUT("/:id", r.listViewsUpdate)
				listViews.DELETE("/:id", r.listViewsDelete)
				listViews.POST("/:id/duplicate", r.listViewsDuplicate)
				listViews.POST("/:id/set-default", r.listViewsSetDefault)
			}

			// ============ P7：订单数据接入 ============

			// 上传模板（admin）
			tpls2 := authorized.Group("/upload-templates")
			tpls2.Use(middleware.RequireAdmin())
			{
				tpls2.GET("", r.uploadTemplateList)
				tpls2.POST("", r.uploadTemplateCreate)
				tpls2.GET("/:id", r.uploadTemplateGet)
				tpls2.PUT("/:id", r.uploadTemplateUpdate)
				tpls2.DELETE("/:id", r.uploadTemplateDelete)
			}

			// 合作方（admin）
			partners := authorized.Group("/partners")
			partners.Use(middleware.RequireAdmin())
			{
				partners.GET("", r.partnerList)
				partners.POST("", r.partnerCreate)
				partners.GET("/:id", r.partnerGet)
				partners.PUT("/:id", r.partnerUpdate)
				partners.DELETE("/:id", r.partnerDelete)
				partners.POST("/:id/pull", r.partnerPull)
				partners.POST("/:id/regenerate-secret", r.partnerRegenerateSecret)
			}

			// 订单（admin / A B 主管 / B 部员工按权限）
			orders := authorized.Group("/orders")
			{
				orders.GET("", r.orderList)
				orders.GET("/stats", r.orderStats)
				orders.GET("/:id", r.orderGet)
				orders.PATCH("/:id/bind", r.orderBind)
				orders.DELETE("/:id", r.orderDelete)

				// 手动上传（admin）
				orders.POST("/import/preview", middleware.RequireAdmin(), r.orderImportPreview)
				orders.POST("/import", middleware.RequireAdmin(), r.orderImport)
			}
		}
	}

	// WebSocket 端点（不需要 JWT 中间件，内部校验 token）
	r.GET("/ws", r.wsConnect)

	// ============ P7：Webhook 公开端点（无需 JWT，HMAC 验签） ============
	// URL: POST /api/v1/orders/webhook/:partner_code
	v1Public := r.Group("/api/v1")
	v1Public.POST("/orders/webhook/:partner_code", r.webhookReceive)
}

// ============= 占位实现（实际 handler 在后续文件） =============

// 列表视图（route 绑定 forwarder；实现在 list_view.go）
func (r *Router) listViewsList(c *gin.Context)             { r.listViewList(c) }
func (r *Router) listViewsGet(c *gin.Context)              { r.listViewGet(c) }
func (r *Router) listViewsCreate(c *gin.Context)           { r.listViewCreate(c) }
func (r *Router) listViewsUpdate(c *gin.Context)           { r.listViewUpdate(c) }
func (r *Router) listViewsDelete(c *gin.Context)           { r.listViewDelete(c) }
func (r *Router) listViewsDuplicate(c *gin.Context)        { r.listViewDuplicate(c) }
func (r *Router) listViewsSetDefault(c *gin.Context)       { r.listViewSetDefault(c) }
func (r *Router) listViewsAvailableFields(c *gin.Context)  { r.listViewAvailableFields(c) }

// 用户管理
func (r *Router) listUsers(c *gin.Context)         { r.userList(c) }
func (r *Router) createUser(c *gin.Context)        { r.userCreate(c) }
func (r *Router) getUser(c *gin.Context)           { r.userGet(c) }
func (r *Router) updateUser(c *gin.Context)        { r.userUpdate(c) }
func (r *Router) deleteUser(c *gin.Context)        { r.userDelete(c) }
func (r *Router) resetUserPassword(c *gin.Context) { r.userResetPassword(c) }
func (r *Router) usersOptions(c *gin.Context)        { r.userOptions(c) }

// 部门管理
func (r *Router) listDepartments(c *gin.Context)   { r.deptList(c) }
func (r *Router) createDepartment(c *gin.Context)  { r.deptCreate(c) }
func (r *Router) getDepartment(c *gin.Context)     { r.deptGet(c) }
func (r *Router) updateDepartment(c *gin.Context)  { r.deptUpdate(c) }
func (r *Router) deleteDepartment(c *gin.Context)  { r.deptDelete(c) }