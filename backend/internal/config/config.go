package config

import (
	"fmt"
	"strings"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Redis    RedisConfig
	JWT      JWTConfig
	Log      LogConfig
	MinIO    MinIOConfig
	Turnstile TurnstileConfig
}

type ServerConfig struct {
	Port    string
	BaseURL string // 外部访问 URL(拼接 webhook URL 用)
	Mode string // debug / release
}

type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
	SSLMode  string
}

func (d DatabaseConfig) DSN() string {
	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		d.Host, d.Port, d.User, d.Password, d.DBName, d.SSLMode)
}

type RedisConfig struct {
	Host     string
	Port     string
	Password string
	DB       int
}

func (r RedisConfig) Addr() string {
	return fmt.Sprintf("%s:%s", r.Host, r.Port)
}

type JWTConfig struct {
	Secret          string
	AccessTokenTTL  int // seconds
	RefreshTokenTTL int // seconds
}

type LogConfig struct {
	Level string
}

type MinIOConfig struct {
	Endpoint  string
	AccessKey string
	SecretKey string
	Bucket    string
	UseSSL    bool
}

type TurnstileConfig struct {
	SecretKey string
	VerifyURL string
}

// Load 加载配置：先读 .env，再读环境变量
func Load() (*Config, error) {
	// 尝试加载 .env（如果存在）
	_ = godotenv.Load()

	viper.SetEnvPrefix("CRM")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	setDefaults()

	cfg := &Config{
		Server: ServerConfig{
			Port:    getEnv("CRM_SERVER_PORT", "8080"),
			Mode:    getEnv("CRM_SERVER_MODE", "debug"),
			BaseURL: getEnv("CRM_SERVER_BASE_URL", "http://localhost:8080"),
		},
		Database: DatabaseConfig{
			Host:     getEnv("CRM_DATABASE_HOST", "localhost"),
			Port:     getEnv("CRM_DATABASE_PORT", "5432"),
			User:     getEnv("CRM_DATABASE_USER", "crm"),
			Password: getEnv("CRM_DATABASE_PASSWORD", "crm_password"),
			DBName:   getEnv("CRM_DATABASE_DBNAME", "crm_db"),
			SSLMode:  getEnv("CRM_DATABASE_SSLMODE", "disable"),
		},
		Redis: RedisConfig{
			Host:     getEnv("CRM_REDIS_HOST", "localhost"),
			Port:     getEnv("CRM_REDIS_PORT", "6379"),
			Password: getEnv("CRM_REDIS_PASSWORD", ""),
			DB:       getIntEnv("CRM_REDIS_DB", 0),
		},
		JWT: JWTConfig{
			Secret:          getEnv("CRM_JWT_SECRET", "change-me-in-production"),
			AccessTokenTTL:  getIntEnv("CRM_JWT_ACCESS_TTL", 86400),    // 24h
			RefreshTokenTTL: getIntEnv("CRM_JWT_REFRESH_TTL", 604800), // 7d
		},
		Log: LogConfig{
			Level: getEnv("CRM_LOG_LEVEL", "info"),
		},
		MinIO: MinIOConfig{
			Endpoint:  getEnv("CRM_MINIO_ENDPOINT", "localhost:9000"),
			AccessKey: getEnv("CRM_MINIO_ACCESS_KEY", "minio"),
			SecretKey: getEnv("CRM_MINIO_SECRET_KEY", "minio_password"),
			Bucket:    getEnv("CRM_MINIO_BUCKET", "crm-files"),
			UseSSL:    getBoolEnv("CRM_MINIO_USESSL", false),
		},
		Turnstile: TurnstileConfig{
			SecretKey: getEnv("CRM_TURNSTILE_SECRET", ""),
			VerifyURL: getEnv("CRM_TURNSTILE_VERIFY_URL", "https://challenges.cloudflare.com/turnstile/v0/siteverify"),
		},
	}

	return cfg, nil
}

func getEnv(key, def string) string {
	if v := viper.GetString(key); v != "" {
		return v
	}
	return def
}

func getIntEnv(key string, def int) int {
	if viper.IsSet(key) {
		return viper.GetInt(key)
	}
	return def
}

func getBoolEnv(key string, def bool) bool {
	if viper.IsSet(key) {
		return viper.GetBool(key)
	}
	return def
}

func setDefaults() {
	viper.SetDefault("CRM_SERVER_PORT", "8080")
	viper.SetDefault("CRM_SERVER_MODE", "debug")
	viper.SetDefault("CRM_DATABASE_HOST", "localhost")
	viper.SetDefault("CRM_DATABASE_PORT", "5432")
	viper.SetDefault("CRM_REDIS_HOST", "localhost")
	viper.SetDefault("CRM_REDIS_PORT", "6379")
	viper.SetDefault("CRM_LOG_LEVEL", "info")
	viper.SetDefault("CRM_JWT_ACCESS_TTL", 86400)
	viper.SetDefault("CRM_JWT_REFRESH_TTL", 604800)
}