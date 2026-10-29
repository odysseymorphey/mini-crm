package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Redis    RedisConfig
	JWT      JWTConfig
	Rate     RateLimitConfig
}

type ServerConfig struct {
	Port            string
	Env             string
	ShutdownTimeout time.Duration
}

type DatabaseConfig struct {
	Host            string
	Port            string
	User            string
	Password        string
	Name            string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

func (d DatabaseConfig) DSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true&charset=utf8mb4&collation=utf8mb4_unicode_ci&multiStatements=true",
		d.User, d.Password, d.Host, d.Port, d.Name)
}

type RedisConfig struct {
	Host     string
	Port     string
	Password string
	DB       int
	CacheTTL time.Duration
}

func (r RedisConfig) Addr() string {
	return fmt.Sprintf("%s:%s", r.Host, r.Port)
}

type JWTConfig struct {
	Secret string
	TTL    time.Duration
}

type RateLimitConfig struct {
	RequestsPerSecond float64
	Burst             int
}

func Load() *Config {
	return &Config{
		Server: ServerConfig{
			Port:            getEnv("SERVER_PORT", "8080"),
			Env:             getEnv("APP_ENV", "development"),
			ShutdownTimeout: 10 * time.Second,
		},
		Database: DatabaseConfig{
			Host:            getEnv("DB_HOST", "localhost"),
			Port:            getEnv("DB_PORT", "3306"),
			User:            getEnv("DB_USER", "crm_user"),
			Password:        getEnv("DB_PASSWORD", "crm_pass"),
			Name:            getEnv("DB_NAME", "crm_db"),
			MaxOpenConns:    getEnvAsInt("DB_MAX_OPEN_CONNS", 25),
			MaxIdleConns:    getEnvAsInt("DB_MAX_IDLE_CONNS", 10),
			ConnMaxLifetime: 15 * time.Minute,
		},
		Redis: RedisConfig{
			Host:     getEnv("REDIS_HOST", "localhost"),
			Port:     getEnv("REDIS_PORT", "6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getEnvAsInt("REDIS_DB", 0),
			CacheTTL: time.Duration(getEnvAsInt("CACHE_TTL_MINUTES", 5)) * time.Minute,
		},
		JWT: JWTConfig{
			Secret: getEnv("JWT_SECRET", "super-secret-jwt-key-for-mini-crm-service-production"),
			TTL:    time.Duration(getEnvAsInt("JWT_TTL_HOURS", 24)) * time.Hour,
		},
		Rate: RateLimitConfig{
			RequestsPerSecond: getEnvAsFloat("RATE_LIMIT_RPS", 50),
			Burst:             getEnvAsInt("RATE_LIMIT_BURST", 100),
		},
	}
}

func getEnv(key, defaultVal string) string {
	if val, ok := os.LookupEnv(key); ok && val != "" {
		return val
	}
	return defaultVal
}

func getEnvAsInt(key string, defaultVal int) int {
	if val, ok := os.LookupEnv(key); ok {
		if intVal, err := strconv.Atoi(val); err == nil {
			return intVal
		}
	}
	return defaultVal
}

func getEnvAsFloat(key string, defaultVal float64) float64 {
	if val, ok := os.LookupEnv(key); ok {
		if floatVal, err := strconv.ParseFloat(val, 64); err == nil {
			return floatVal
		}
	}
	return defaultVal
}
