package main

import (
	"log"

	"crm-backend/internal/api"
	"crm-backend/internal/config"
	"crm-backend/internal/model"
	"crm-backend/pkg/database"
	"crm-backend/pkg/password"
)

// seed 数据库种子数据
// 用法: go run cmd/seed/main.go
//
// 创建：
//  - 公司（顶级部门）
//  - A 部门 / B 部门
//  - 默认 admin 账号（用户名 admin / 密码 admin123）
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

	log.Println("🌱 开始写入种子数据...")

	// 1. 创建公司
	var company model.Department
	if err := db.Where("type = ?", model.DeptTypeCompany).First(&company).Error; err != nil {
		if err := db.Create(&model.Department{
			Name: "本公司",
			Type: model.DeptTypeCompany,
		}).Error; err != nil {
			log.Fatalf("创建公司失败: %v", err)
		} else {
			log.Println("✓ 创建公司")
			db.Where("type = ?", model.DeptTypeCompany).First(&company)
		}
	} else {
		log.Println("• 公司已存在")
	}

	// 2. 创建 A 部门 / B 部门
	deptsToCreate := []model.Department{
		{Name: "前端", ParentID: &company.ID, Type: model.DeptTypeA},
		{Name: "后端", ParentID: &company.ID, Type: model.DeptTypeB},
	}
	for _, d := range deptsToCreate {
		var existing model.Department
		err := db.Where("type = ? AND parent_id = ?", d.Type, company.ID).First(&existing).Error
		if err != nil {
			d.ParentID = &company.ID
			if err := db.Create(&d).Error; err != nil {
				log.Printf("❌ 创建 %s 失败: %v", d.Name, err)
			} else {
				log.Printf("✓ 创建 %s (ID=%d)", d.Name, d.ID)
			}
		} else {
			log.Printf("• %s 已存在 (ID=%d)", d.Name, existing.ID)
		}
	}

	// 3. 创建 admin 用户
	var admin model.User
	if err := db.Where("username = ?", "admin").First(&admin).Error; err != nil {
		hashed, err := password.Hash("admin123")
		if err != nil {
			log.Fatalf("密码哈希失败: %v", err)
		}
		admin = model.User{
			Username:     "admin",
			PasswordHash: hashed,
			RealName:     "系统管理员",
			Role:         model.RoleAdmin,
			Status:       model.UserStatusEnabled,
		}
		if err := db.Create(&admin).Error; err != nil {
			log.Fatalf("创建 admin 失败: %v", err)
		}
		log.Println("✓ 创建 admin 账号 (用户名: admin / 密码: admin123)")
	} else {
		log.Println("• admin 账号已存在")
	}

	log.Println("✅ 种子数据写入完成")

	// 4. 初始化列表视图默认配置（玩家 / 订单各自的「默认视图」）
	log.Println("🌱 开始初始化列表视图默认配置...")
	// 拿到 admin 的 router 实例(只要 DB 字段)
	tmpRouter := &api.Router{DB: db}
	if err := api.SeedListViewsForCLI(tmpRouter, admin.ID); err != nil {
		log.Printf("❌ 初始化列表视图失败: %v", err)
	} else {
		log.Println("✓ 列表视图默认配置就绪")
	}

	log.Println("✅ 所有种子写入完成")
}