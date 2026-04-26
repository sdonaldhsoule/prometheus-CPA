package config

import (
	"fmt"
	"os"
)

// Config 应用配置
type Config struct {
	// 数据库连接URL
	DatabaseURL string

	// 回调接口配置
	CallbackURL    string
	CallbackSecret string

	// JWT密钥用于安全传输
	JWTSecret string

	// CDK前缀
	CDKPrefix string

	// 站点配置
	SiteName        string
	BackgroundImage string

	// 管理员配置
	AdminUsername string
	AdminPassword string

	// CPA配置
	CPABaseURL       string
	CPAManagementKey string

	// NewAPI 登录配置
	NewAPIBaseURL string
}

// Load 从环境变量加载配置
func Load() (*Config, error) {
	cfg := &Config{
		DatabaseURL:      getEnvOrDefault("DATABASE_URL", "postgres://donation:donation123@localhost:5432/donation_station?sslmode=disable"),
		CallbackURL:      getEnvOrDefault("CALLBACK_URL", ""),
		CallbackSecret:   getEnvOrDefault("CALLBACK_SECRET", generateDefaultSecret()),
		JWTSecret:        getEnvOrDefault("JWT_SECRET", generateDefaultSecret()),
		CDKPrefix:        getEnvOrDefault("CDK_PREFIX", "DS"),
		SiteName:         getEnvOrDefault("SITE_NAME", "凭证捐赠站"),
		BackgroundImage:  getEnvOrDefault("BACKGROUND_IMAGE", ""),
		AdminUsername:    getEnvOrDefault("ADMIN_USERNAME", "admin"),
		AdminPassword:    getEnvOrDefault("ADMIN_PASSWORD", "admin123"),
		CPABaseURL:       getEnvOrDefault("CPA_BASE_URL", ""),
		CPAManagementKey: getEnvOrDefault("CPA_MANAGEMENT_KEY", ""),
		NewAPIBaseURL:    getEnvOrDefault("NEWAPI_BASE_URL", ""),
	}

	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}

	if cfg.CPABaseURL == "" {
		return nil, fmt.Errorf("CPA_BASE_URL is required")
	}

	if cfg.CPAManagementKey == "" {
		return nil, fmt.Errorf("CPA_MANAGEMENT_KEY is required")
	}

	if cfg.NewAPIBaseURL == "" {
		return nil, fmt.Errorf("NEWAPI_BASE_URL is required")
	}

	return cfg, nil
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func generateDefaultSecret() string {
	// 生产环境应该使用真正的随机密钥
	return "default-secret-change-in-production"
}
