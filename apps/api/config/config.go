package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	App     AppConfig
	DB      DBConfig
	Redis   RedisConfig
	JWT     JWTConfig
	SMTP    SMTPConfig
	Storage StorageConfig
	CORS    CORSConfig
	Rate    RateLimitConfig
}

type AppConfig struct {
	Env       string
	Port      string
	SecretKey string
}

type DBConfig struct {
	Host         string
	Port         string
	Name         string
	User         string
	Password     string
	SSLMode      string
	MaxOpenConns int
	MaxIdleConns int
}

type RedisConfig struct {
	Addr     string
	Password string
	DB       int
}

type JWTConfig struct {
	PrivateKeyPath string
	PublicKeyPath  string
	AccessTTL      time.Duration
	RefreshTTL     time.Duration
}

type SMTPConfig struct {
	Host        string
	Port        int
	Username    string
	Password    string
	FromAddress string
	FromName    string
}

type StorageConfig struct {
	Driver    string // local | s3
	LocalPath string
	S3        S3Config
}

type S3Config struct {
	Endpoint     string
	Bucket       string
	AccessKey    string
	SecretKey    string
	Region       string
	UsePathStyle bool
}

type CORSConfig struct {
	AllowedOrigins []string
}

type RateLimitConfig struct {
	RequestsPerMinute     int
	AuthRequestsPerMinute int
}

func Load() (*Config, error) {
	accessTTL, err := parseDurationMinutes(getEnv("JWT_ACCESS_TTL_MINUTES", "15"))
	if err != nil {
		return nil, fmt.Errorf("JWT_ACCESS_TTL_MINUTES: %w", err)
	}

	refreshTTL, err := parseDurationDays(getEnv("JWT_REFRESH_TTL_DAYS", "30"))
	if err != nil {
		return nil, fmt.Errorf("JWT_REFRESH_TTL_DAYS: %w", err)
	}

	return &Config{
		App: AppConfig{
			Env:       getEnv("APP_ENV", "development"),
			Port:      getEnv("APP_PORT", "8080"),
			SecretKey: requireEnv("APP_SECRET_KEY"),
		},
		DB: DBConfig{
			Host:         getEnv("DB_HOST", "localhost"),
			Port:         getEnv("DB_PORT", "5432"),
			Name:         getEnv("DB_NAME", "novudesk"),
			User:         getEnv("DB_USER", "novudesk"),
			Password:     requireEnv("DB_PASSWORD"),
			SSLMode:      getEnv("DB_SSL_MODE", "disable"),
			MaxOpenConns: parseInt(getEnv("DB_MAX_OPEN_CONNS", "25")),
			MaxIdleConns: parseInt(getEnv("DB_MAX_IDLE_CONNS", "25")),
		},
		Redis: RedisConfig{
			Addr:     getEnv("REDIS_ADDR", "localhost:6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       parseInt(getEnv("REDIS_DB", "0")),
		},
		JWT: JWTConfig{
			PrivateKeyPath: getEnv("JWT_PRIVATE_KEY_PATH", "./config/keys/private.pem"),
			PublicKeyPath:  getEnv("JWT_PUBLIC_KEY_PATH", "./config/keys/public.pem"),
			AccessTTL:      accessTTL,
			RefreshTTL:     refreshTTL,
		},
		SMTP: SMTPConfig{
			Host:        getEnv("SMTP_HOST", "localhost"),
			Port:        parseInt(getEnv("SMTP_PORT", "1025")),
			Username:    getEnv("SMTP_USERNAME", ""),
			Password:    getEnv("SMTP_PASSWORD", ""),
			FromAddress: getEnv("SMTP_FROM_ADDRESS", "noreply@novudesk.dev"),
			FromName:    getEnv("SMTP_FROM_NAME", "NovuDesk"),
		},
		Storage: StorageConfig{
			Driver:    getEnv("STORAGE_DRIVER", "local"),
			LocalPath: getEnv("STORAGE_LOCAL_PATH", "./uploads"),
			S3: S3Config{
				Endpoint:     getEnv("S3_ENDPOINT", ""),
				Bucket:       getEnv("S3_BUCKET", "novudesk"),
				AccessKey:    getEnv("S3_ACCESS_KEY", ""),
				SecretKey:    getEnv("S3_SECRET_KEY", ""),
				Region:       getEnv("S3_REGION", "us-east-1"),
				UsePathStyle: parseBool(getEnv("S3_USE_PATH_STYLE", "false")),
			},
		},
		CORS: CORSConfig{
			AllowedOrigins: []string{getEnv("CORS_ALLOWED_ORIGINS", "http://localhost:5173")},
		},
		Rate: RateLimitConfig{
			RequestsPerMinute:     parseInt(getEnv("RATE_LIMIT_REQUESTS_PER_MINUTE", "60")),
			AuthRequestsPerMinute: parseInt(getEnv("RATE_LIMIT_AUTH_REQUESTS_PER_MINUTE", "10")),
		},
	}, nil
}

// DSN returns the PostgreSQL connection string.
func (c *DBConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%s dbname=%s user=%s password=%s sslmode=%s",
		c.Host, c.Port, c.Name, c.User, c.Password, c.SSLMode,
	)
}

func (c *AppConfig) IsDevelopment() bool {
	return c.Env == "development"
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func requireEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		panic(fmt.Sprintf("required environment variable %q is not set", key))
	}
	return v
}

func parseInt(s string) int {
	n, _ := strconv.Atoi(s)
	return n
}

func parseBool(s string) bool {
	b, _ := strconv.ParseBool(s)
	return b
}

func parseDurationMinutes(s string) (time.Duration, error) {
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0, err
	}
	return time.Duration(n) * time.Minute, nil
}

func parseDurationDays(s string) (time.Duration, error) {
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0, err
	}
	return time.Duration(n) * 24 * time.Hour, nil
}
