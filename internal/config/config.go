package config

import (
	"os"
	"strings"
	"time"
)

type Config struct {
	Port           string
	AllowedOrigins []string

	JWTKeys map[string]string
	Skew    time.Duration

	MockEnabled bool
	BackoffMax  time.Duration

	MetricsUser string
	MetricsPass string

	LogPath string
}

func env(k, def string) string {
	if v, ok := os.LookupEnv(k); ok {
		return v
	}
	return def
}

func parseKeyMap(s string) map[string]string {
	out := map[string]string{}
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		kv := strings.SplitN(part, ":", 2)
		if len(kv) != 2 {
			continue
		}
		out[strings.TrimSpace(kv[0])] = strings.TrimSpace(kv[1])
	}
	return out
}

func New() *Config {
	mock := strings.EqualFold(env("MOCK_ENABLED", "true"), "true")
	max, _ := time.ParseDuration(env("BACKOFF_MAX", "30s"))
	origins := strings.Split(env("CORS_ORIGINS", "http://localhost:5173,http://localhost:3000"), ",")
	keys := parseKeyMap(env("JWT_KEYS", ""))

	if len(keys) == 0 {
		if s := env("JWT_HS256_SECRET", ""); s != "" {
			keys["default"] = s
		}
	}
	skew, _ := time.ParseDuration(env("JWT_SKEW", "2m"))

	return &Config{
		Port:           env("PORT", "8080"),
		AllowedOrigins: origins,
		JWTKeys:        keys,
		Skew:           skew,
		MockEnabled:    mock,
		BackoffMax:     max,
		MetricsUser:    env("METRICS_USER", ""),
		MetricsPass:    env("METRICS_PASS", ""),
		LogPath:        env("LOG_PATH", "./data/events.log"),
	}
}
