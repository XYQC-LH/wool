package database

import (
	"errors"
	"log"
	"strings"

	"nexus-api/internal/config"
	"nexus-api/internal/model"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// SeedAdminUser 创建默认管理员账号
func SeedAdminUser(cfg *config.Config) error {
	db := GetDB()

	desiredUsername := strings.TrimSpace(cfg.DefaultAdmin.Username)
	desiredEmail := strings.TrimSpace(cfg.DefaultAdmin.Email)
	desiredPassword := cfg.DefaultAdmin.Password
	if desiredUsername == "" || desiredPassword == "" {
		log.Println("默认管理员账号未配置（用户名或密码为空），跳过创建")
		return nil
	}

	// 显式配置过默认管理员时：确保指定用户名的管理员存在（即使已存在其他管理员账号）
	if cfg.DefaultAdmin.Configured {
		var existing model.User
		err := db.Where("username = ?", desiredUsername).First(&existing).Error
		if err == nil {
			if !existing.IsAdmin() {
				log.Printf("默认管理员用户名已存在但不是管理员，跳过创建: %s", desiredUsername)
				return nil
			}
			log.Printf("默认管理员账号已存在，跳过创建: %s", desiredUsername)
			return nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			log.Printf("检查默认管理员账号失败: %v", err)
			return err
		}
	} else {
		// 未显式配置时：保持原行为，仅在系统内不存在任何管理员时才创建默认管理员
		var adminCount int64
		if err := db.Model(&model.User{}).
			Where("role IN ?", []model.UserRole{model.RoleAdmin, model.RoleSuperAdmin}).
			Count(&adminCount).Error; err != nil {
			log.Printf("检查管理员账号失败: %v", err)
			return err
		}
		if adminCount > 0 {
			log.Println("管理员账号已存在，跳过创建")
			return nil
		}
	}

	// 加密密码
	hashedPassword, err := bcrypt.GenerateFromPassword(
		[]byte(desiredPassword),
		bcrypt.DefaultCost,
	)
	if err != nil {
		log.Printf("加密管理员密码失败: %v", err)
		return err
	}

	// 创建默认管理员
	admin := &model.User{
		ID:           uuid.New(),
		Username:     desiredUsername,
		Email:        desiredEmail,
		PasswordHash: string(hashedPassword),
		Role:         model.RoleAdmin,
		Status:       model.UserStatusActive,
	}

	createErr := db.Create(admin).Error
	if createErr != nil && cfg.DefaultAdmin.Configured && strings.Contains(createErr.Error(), "idx_users_email") {
		fallbackEmail := desiredUsername + "@example.com"
		if !strings.EqualFold(fallbackEmail, desiredEmail) {
			log.Printf("默认管理员邮箱已被占用，自动调整为: %s", fallbackEmail)
			admin.Email = fallbackEmail
			createErr = db.Create(admin).Error
			if createErr == nil {
				desiredEmail = fallbackEmail
			}
		}
	}
	if createErr != nil {
		log.Printf("创建默认管理员账号失败: %v", createErr)
		return createErr
	}

	log.Printf("✓ 默认管理员账号创建成功")
	log.Printf("  用户名: %s", desiredUsername)
	log.Printf("  邮箱: %s", desiredEmail)
	log.Printf("  密码: (已设置，出于安全原因不输出到日志)")
	log.Printf("⚠️  首次登录后请立即修改默认密码！")

	return nil
}
