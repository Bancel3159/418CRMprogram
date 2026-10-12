package main

import (
	"log"

	"crm-backend/internal/config"
	"crm-backend/internal/model"
	"crm-backend/pkg/database"
)

// migrate 数据库迁移工具
// 用法: go run cmd/migrate/main.go
func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("配置加载失败: %v", err)
	}

	db, err := database.NewPostgres(cfg.Database)
	if err != nil {
		log.Fatalf("数据库连接失败: %v", err)
	}
	defer func() {
		sqlDB, _ := db.DB()
		if sqlDB != nil {
			sqlDB.Close()
		}
	}()

	log.Println("🚀 开始数据库迁移...")

	if err := db.AutoMigrate(model.GetAllModels()...); err != nil {
		log.Fatalf("❌ 迁移失败: %v", err)
	}

	log.Println("✅ 迁移完成")
	for _, m := range model.GetAllModels() {
		log.Printf("   - %T", m)
	}
}