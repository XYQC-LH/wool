package config

import (
	"os"
	"strconv"
	"strings"
)

// Config 应用配置
type Config struct {
	Server       ServerConfig
	Database     DatabaseConfig
	Redis        RedisConfig
	OSS          OSSConfig
	JWT          JWTConfig
	RateLimit    RateLimitConfig
	Payment      PaymentConfig
	ResourcePool ResourcePoolConfig
	DefaultAdmin DefaultAdminConfig
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Port           string
	Mode           string
	ReadTimeout    int
	WriteTimeout   int
	AllowedOrigins []string
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	Host            string
	Port            int
	User            string
	Password        string
	DBName          string
	SSLMode         string
	MaxIdleConns    int
	MaxOpenConns    int
	ConnMaxLifetime int
}

// RedisConfig Redis 配置
type RedisConfig struct {
	Host         string
	Port         int
	Password     string
	DB           int
	PoolSize     int
	MinIdleConns int
}

// OSSConfig OSS(对象存储)配置
type OSSConfig struct {
	// Driver 存储驱动：local 或 oss
	Driver string
	// LocalDir 本地存储目录（Driver=local）
	LocalDir string
	// SignSecret 本地签名密钥（Driver=local，不设置则默认使用 JWT_SECRET）
	SignSecret string

	Endpoint          string
	Bucket            string
	AccessKeyID       string
	AccessKeySecret   string
	PublicBaseURL     string
	SignExpireSeconds int
}

// JWTConfig JWT 配置
type JWTConfig struct {
	Secret      string
	ExpireHours int
	Issuer      string
}

// RateLimitConfig 速率限制配置
type RateLimitConfig struct {
	RequestsPerMinute     int
	BurstSize             int
	IPRequestsPerMinute   int
	UserRequestsPerMinute int
}

// PaymentConfig 支付配置
type PaymentConfig struct {
	StripeSecretKey      string
	StripeWebhookSecret  string
	StripePublishableKey string
}

// ResourcePoolConfig 资源池配置
type ResourcePoolConfig struct {
	URL     string
	Timeout int // 秒
}

// DefaultAdminConfig 默认管理员配置
type DefaultAdminConfig struct {
	Username string
	Password string
	Email    string
	// Configured 是否通过环境变量显式配置过默认管理员账号
	Configured bool
}

// Load 加载配置
func Load() (*Config, error) {
	jwtSecret := getEnv("JWT_SECRET", "your-secret-key-change-in-production")

	storageDriver := strings.ToLower(strings.TrimSpace(getEnv("STORAGE_DRIVER", "")))
	storageLocalDir := strings.TrimSpace(getEnv("STORAGE_LOCAL_DIR", "./data/objects"))
	storageSignSecret := strings.TrimSpace(getEnv("STORAGE_SIGN_SECRET", ""))
	if storageSignSecret == "" {
		storageSignSecret = jwtSecret
	}

	ossEndpoint := getEnv("OSS_ENDPOINT", "")
	ossBucket := getEnv("OSS_BUCKET", "")
	ossAccessKeyID := getEnv("OSS_ACCESS_KEY_ID", "")
	ossAccessKeySecret := getEnv("OSS_ACCESS_KEY_SECRET", "")
	ossPublicBaseURL := strings.TrimRight(getEnv("OSS_PUBLIC_BASE_URL", ""), "/")
	ossSignExpireSeconds := getEnvInt("OSS_SIGN_EXPIRE_SECONDS", 900)

	// Driver 未显式配置时：若 OSS 必要参数齐全则使用 OSS，否则使用本地存储
	if storageDriver == "" {
		if strings.TrimSpace(ossEndpoint) != "" &&
			strings.TrimSpace(ossBucket) != "" &&
			strings.TrimSpace(ossAccessKeyID) != "" &&
			strings.TrimSpace(ossAccessKeySecret) != "" {
			storageDriver = "oss"
		} else {
			storageDriver = "local"
		}
	}

	// 本地存储下 Bucket 仅用于元数据标识，允许自动填充默认值
	if storageDriver == "local" && strings.TrimSpace(ossBucket) == "" {
		ossBucket = "local"
	}

	defaultAdminConfigured := hasAnyEnv(
		"ADMIN_USERNAME", "DEFAULT_ADMIN_USERNAME",
		"ADMIN_PASSWORD", "DEFAULT_ADMIN_PASSWORD",
		"ADMIN_EMAIL", "DEFAULT_ADMIN_EMAIL",
	)

	return &Config{
		Server: ServerConfig{
			Port:           getEnv("SERVER_PORT", "8080"),
			Mode:           getEnv("SERVER_MODE", "debug"),
			ReadTimeout:    getEnvInt("SERVER_READ_TIMEOUT", 30),
			WriteTimeout:   getEnvInt("SERVER_WRITE_TIMEOUT", 60),
			AllowedOrigins: getEnvSlice("ALLOWED_ORIGINS", []string{"*"}),
		},
		Database: DatabaseConfig{
			Host:            getEnv("DB_HOST", "localhost"),
			Port:            getEnvInt("DB_PORT", 5432),
			User:            getEnv("DB_USER", "nexus"),
			Password:        getEnv("DB_PASSWORD", ""),
			DBName:          getEnv("DB_NAME", "nexus_api"),
			SSLMode:         getEnv("DB_SSL_MODE", "disable"),
			MaxIdleConns:    getEnvInt("DB_MAX_IDLE_CONNS", 10),
			MaxOpenConns:    getEnvInt("DB_MAX_OPEN_CONNS", 100),
			ConnMaxLifetime: getEnvInt("DB_CONN_MAX_LIFETIME", 3600),
		},
		Redis: RedisConfig{
			Host:         getEnv("REDIS_HOST", "localhost"),
			Port:         getEnvInt("REDIS_PORT", 6379),
			Password:     getEnv("REDIS_PASSWORD", ""),
			DB:           getEnvInt("REDIS_DB", 0),
			PoolSize:     getEnvInt("REDIS_POOL_SIZE", 10),
			MinIdleConns: getEnvInt("REDIS_MIN_IDLE_CONNS", 5),
		},
		OSS: OSSConfig{
			Driver:            storageDriver,
			LocalDir:          storageLocalDir,
			SignSecret:        storageSignSecret,
			Endpoint:          ossEndpoint,
			Bucket:            ossBucket,
			AccessKeyID:       ossAccessKeyID,
			AccessKeySecret:   ossAccessKeySecret,
			PublicBaseURL:     ossPublicBaseURL,
			SignExpireSeconds: ossSignExpireSeconds,
		},
		JWT: JWTConfig{
			Secret:      jwtSecret,
			ExpireHours: getEnvInt("JWT_EXPIRE_HOURS", 168), // 7 days
			Issuer:      getEnv("JWT_ISSUER", "nexus-api-gateway"),
		},
		RateLimit: RateLimitConfig{
			RequestsPerMinute:     getEnvInt("RATE_LIMIT_RPM", 60),
			BurstSize:             getEnvInt("RATE_LIMIT_BURST", 10),
			IPRequestsPerMinute:   getEnvInt("RATE_LIMIT_IP_RPM", 100),
			UserRequestsPerMinute: getEnvInt("RATE_LIMIT_USER_RPM", 60),
		},
		Payment: PaymentConfig{
			StripeSecretKey:      getEnv("STRIPE_SECRET_KEY", ""),
			StripeWebhookSecret:  getEnv("STRIPE_WEBHOOK_SECRET", ""),
			StripePublishableKey: getEnv("STRIPE_PUBLISHABLE_KEY", ""),
		},
		ResourcePool: ResourcePoolConfig{
			URL:     getEnv("RESOURCE_POOL_URL", "http://localhost:8001"),
			Timeout: getEnvInt("RESOURCE_POOL_TIMEOUT", 600),
		},
		DefaultAdmin: DefaultAdminConfig{
			Username:   getEnvFallback("ADMIN_USERNAME", "DEFAULT_ADMIN_USERNAME", "admin"),
			Password:   getEnvFallback("ADMIN_PASSWORD", "DEFAULT_ADMIN_PASSWORD", "admin123456"),
			Email:      getEnvFallback("ADMIN_EMAIL", "DEFAULT_ADMIN_EMAIL", "admin@example.com"),
			Configured: defaultAdminConfigured,
		},
	}, nil
}

// getEnv 获取环境变量，如果不存在则返回默认值
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvFallback 优先读取 primaryKey，如果为空则读取 fallbackKey，否则返回默认值
func getEnvFallback(primaryKey, fallbackKey, defaultValue string) string {
	if value := strings.TrimSpace(os.Getenv(primaryKey)); value != "" {
		return value
	}
	if value := strings.TrimSpace(os.Getenv(fallbackKey)); value != "" {
		return value
	}
	return defaultValue
}

// getEnvInt 获取整数环境变量
func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// getEnvSlice 获取切片环境变量
func getEnvSlice(key string, defaultValue []string) []string {
	if value := os.Getenv(key); value != "" {
		return strings.Split(value, ",")
	}
	return defaultValue
}

func hasAnyEnv(keys ...string) bool {
	for _, key := range keys {
		if strings.TrimSpace(os.Getenv(key)) != "" {
			return true
		}
	}
	return false
}
