package main

import (
	"fmt"
	"log"

	"firemail/internal/config"
	"firemail/internal/database"
	"firemail/internal/models"
	"firemail/internal/security"

	"github.com/joho/godotenv"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func main() {
	// 加载环境变量 - 优先加载.env.local，然后是.env
	if err := godotenv.Load(".env.local"); err != nil {
		// 如果.env.local不存在，尝试加载.env
		if err := godotenv.Load(".env"); err != nil {
			log.Println("Warning: No .env file found, using system environment variables")
		} else {
			log.Println("Loaded configuration from .env file")
		}
	} else {
		log.Println("Loaded configuration from .env.local file")
	}

	// 初始化配置
	cfg := config.Load()
	encryptionStatus, err := security.ConfigureFieldEncryption(security.EncryptionConfig{
		EncryptionKey: cfg.Encryption.Key,
		JWTSecret:     cfg.Auth.JWTSecret,
		Environment:   cfg.Server.Env,
	})
	if err != nil {
		log.Fatalf("❌ Failed to configure database field encryption: %v", err)
	}
	fmt.Printf("🔧 配置信息:\n")
	fmt.Printf("   Admin Username: %s\n", cfg.Auth.AdminUsername)
	fmt.Printf("   Admin Password: [redacted]\n")
	fmt.Printf("   Database Path: %s\n", cfg.Database.Path)
	fmt.Printf("   Encryption Source: %s\n", encryptionStatus.Source)
	fmt.Println("   IMPORTANT: Save JWT_SECRET or ENCRYPTION_KEY; losing it makes encrypted email account credentials unrecoverable.")
	fmt.Println()

	// 初始化数据库
	db, err := database.Initialize(cfg.Database.Path)
	if err != nil {
		log.Fatalf("❌ Failed to initialize database: %v", err)
	}

	// 检查数据库中的用户
	var users []models.User
	if err := db.Find(&users).Error; err != nil {
		log.Fatalf("❌ Failed to query users: %v", err)
	}

	fmt.Printf("📊 数据库中的用户数量: %d\n", len(users))

	if len(users) == 0 {
		fmt.Println("⚠️  数据库中没有用户，这很奇怪，应该有默认admin用户")
		createAdminUser(db, cfg)
		return
	}

	fmt.Println("👥 现有用户:")
	for _, user := range users {
		fmt.Printf("   ID: %d, Username: %s, Active: %t, Role: %s\n",
			user.ID, user.Username, user.IsActive, user.Role)
	}
	fmt.Println()

	// 查找admin用户
	var adminUser models.User
	if err := db.Where("username = ?", cfg.Auth.AdminUsername).First(&adminUser).Error; err != nil {
		fmt.Printf("❌ 找不到用户名为 '%s' 的用户\n", cfg.Auth.AdminUsername)
		fmt.Println("🔧 正在创建admin用户...")
		createAdminUser(db, cfg)
		return
	}

	fmt.Printf("✅ 找到admin用户: %s (ID: %d)\n", adminUser.Username, adminUser.ID)

	// 测试密码
	fmt.Printf("🔐 测试密码 '%s'...\n", cfg.Auth.AdminPassword)
	if adminUser.CheckPassword(cfg.Auth.AdminPassword) {
		fmt.Println("✅ 密码验证成功！")
		fmt.Println("🎉 登录应该可以正常工作")
	} else {
		fmt.Println("❌ 密码验证失败！")
		fmt.Println("🔧 正在重置admin密码...")
		resetAdminPassword(db, &adminUser, cfg.Auth.AdminPassword)
	}

	// 检查用户状态
	if !adminUser.IsActive {
		fmt.Println("⚠️  用户账户未激活，正在激活...")
		adminUser.IsActive = true
		if err := db.Save(&adminUser).Error; err != nil {
			log.Printf("❌ Failed to activate user: %v", err)
		} else {
			fmt.Println("✅ 用户账户已激活")
		}
	}

	fmt.Println("\n🚀 诊断完成！现在可以尝试登录了。")
}

func createAdminUser(db *gorm.DB, cfg *config.Config) {
	admin := &models.User{
		Username:    cfg.Auth.AdminUsername,
		Password:    cfg.Auth.AdminPassword, // 会在BeforeCreate钩子中自动加密
		DisplayName: "Administrator",
		Role:        "admin",
		IsActive:    true,
	}

	if err := db.Create(admin).Error; err != nil {
		log.Fatalf("❌ Failed to create admin user: %v", err)
	}

	fmt.Printf("✅ 成功创建admin用户: %s\n", admin.Username)
}

func resetAdminPassword(db *gorm.DB, user *models.User, newPassword string) {
	// 手动加密密码
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		log.Fatalf("❌ Failed to hash password: %v", err)
	}

	user.Password = string(hashedPassword)
	if err := db.Save(user).Error; err != nil {
		log.Fatalf("❌ Failed to update password: %v", err)
	}

	fmt.Printf("✅ 成功重置密码为: %s\n", newPassword)
}
