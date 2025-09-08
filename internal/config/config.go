package config

import (
	"os"
	"strings"
	"time"
)

type Config struct {
	Port           string
	AllowedOrigins []string
	JWTSecret      string
	MockEnabled    bool
	BackoffMax     time.Duration
}

func env(key, def string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	return def
}

func New() *Config {
	mock := strings.EqualFold(env("MOCK_ENABLED", "true"), "true")
	max, _ := time.ParseDuration(env("BACKOFF_MAX", "30s"))
	origins := strings.Split(env("CORS_ORIGINS", "http://localhost:5173,http://localhost:3000"), ",")
	return &Config{
		Port:           env("PORT", "8080"),
		AllowedOrigins: origins,
		JWTSecret:      env("JWT_HS256_SECRET", ""),
		MockEnabled:    mock,
		BackoffMax:     max,
	}
}
