package main

import (
	"context"
	"log"

	"crm-backend/internal/api"
	"crm-backend/internal/config"
	"crm-backend/internal/model"
	"crm-backend/internal/service"
	"crm-backend/internal/storage"
	"crm-backend/internal/ws"
	"crm-backend/pkg/database"
	"crm-backend/pkg/logger"
)

func main() {
	// 1. 加载配置
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("配置加载失败: %v", err)
	}

	// 2. 初始化日志
	logger.Init(cfg.Log.Level)

	// 3. 初始化数据库连接
	db, err := database.NewPostgres(cfg.Database)
	if err != nil {
		log.Fatalf("PostgreSQL 连接失败: %v", err)
	}
	defer func() {
		sqlDB, _ := db.DB()
		if sqlDB != nil {
			sqlDB.Close()
		}
	}()

	rdb, err := database.NewRedis(cfg.Redis)
	if err != nil {
		log.Fatalf("Redis 连接失败: %v", err)
	}
	defer rdb.Close()

	// 4. 启动 WebSocket Hub
	hub := ws.NewHub()
	go hub.Run()

	// 5. 初始化文件上传器
	uploader := storage.NewUploader("./uploads", "/uploads")

	// 6. 设置路由
	router := api.NewRouter(cfg, db, rdb, hub, uploader)

	// 7. 启动 cron 调度器(P7)
	cronRunner := service.NewCronRunner(db)
	// 注入拉取回调(cron 触发时调用)
	service.SetPullFunc(func(ctx context.Context, partnerID uint) error {
		var partner model.Partner
		if err := db.WithContext(ctx).First(&partner, partnerID).Error; err != nil {
			return err
		}
		// 调用 api 包内私有方法(通过 router 暴露的 PullPartner 入口)
		return router.PullPartnerByID(ctx, &partner)
	})
	cronRunner.ReloadAll(context.Background())

	// P11 跟进规则引擎:每日扫描(从 system_configs 读 cron expr)
	service.EnsureDefaultConfigs(db)
	// P12 实时通知默认值
	service.EnsureRealtimeDefaults(db)
	followEngine := service.NewFollowReminderEngine(db)
	followCronExpr := followEngine.ReloadCronExpr()
	if err := cronRunner.AddGlobalJob("follow_daily_scan", followCronExpr, func() error {
		_, err := followEngine.DailyScan(context.Background())
		return err
	}); err != nil {
		log.Printf("⚠️ 注册 follow_daily_scan 失败: %v(可后续用 API 触发)", err)
	}

	cronRunner.Start()
	defer cronRunner.Stop(context.Background())

	// 8. 启动服务
	addr := ":" + cfg.Server.Port
	log.Printf("🚀 服务启动: http://localhost%s (WS: /ws)", addr)
	if err := router.Run(addr); err != nil {
		log.Fatalf("服务启动失败: %v", err)
	}
}