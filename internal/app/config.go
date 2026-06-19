package app

import (
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Addr              string
	DatabaseURL       string
	PublicBaseURL     string
	AppSecret         []byte
	CookieName        string
	CookieSecure      bool
	TrustProxy        bool
	StaticDir         string
	MaxBodyBytes      int64
	RetentionDays     int
	CleanupInterval   int
	CaptureStatusCode int
}

func LoadConfig() (Config, error) {
	cfg := Config{
		Addr:              env("ADDR", ":8080"),
		DatabaseURL:       os.Getenv("DATABASE_URL"),
		PublicBaseURL:     strings.TrimRight(os.Getenv("PUBLIC_BASE_URL"), "/"),
		CookieName:        env("COOKIE_NAME", "webhook_owner"),
		CookieSecure:      envBool("COOKIE_SECURE", false),
		TrustProxy:        envBool("TRUST_PROXY", false),
		StaticDir:         env("STATIC_DIR", "frontend/dist"),
		MaxBodyBytes:      envInt64("MAX_BODY_BYTES", 2*1024*1024),
		RetentionDays:     envInt("RETENTION_DAYS", 30),
		CleanupInterval:   envInt("CLEANUP_INTERVAL_MINUTES", 60),
		CaptureStatusCode: envInt("CAPTURE_STATUS_CODE", httpStatusOK),
	}

	if cfg.DatabaseURL == "" {
		return cfg, errors.New("DATABASE_URL is required")
	}
	secret := os.Getenv("APP_SECRET")
	if len(secret) < 32 {
		return cfg, errors.New("APP_SECRET must be at least 32 characters")
	}
	cfg.AppSecret = []byte(secret)

	if cfg.RetentionDays < 1 {
		return cfg, errors.New("RETENTION_DAYS must be at least 1")
	}
	if cfg.CleanupInterval < 1 {
		return cfg, errors.New("CLEANUP_INTERVAL_MINUTES must be at least 1")
	}
	if cfg.MaxBodyBytes < 1 || cfg.MaxBodyBytes > 50*1024*1024 {
		return cfg, errors.New("MAX_BODY_BYTES must be between 1 byte and 50 MiB")
	}
	if cfg.CaptureStatusCode < 100 || cfg.CaptureStatusCode > 599 {
		return cfg, fmt.Errorf("CAPTURE_STATUS_CODE must be an HTTP status code")
	}

	return cfg, nil
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func envBool(key string, fallback bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func envInt(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func envInt64(key string, fallback int64) int64 {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return fallback
	}
	return parsed
}

func tokenHash(token string) string {
	sum := sha256Bytes([]byte(token))
	return hex.EncodeToString(sum)
}

const httpStatusOK = 200
