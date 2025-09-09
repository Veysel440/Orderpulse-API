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

	LogPath      string
	LogMaxBytes  int64
	LogRetention time.Duration

	KafkaBrokers []string
	KafkaTopic   string
	KafkaGroup   string
	KafkaEnabled bool

	AmqpURL     string
	AmqpQueue   string
	AmqpEnabled bool
}

func env(k, d string) string {
	if v, ok := os.LookupEnv(k); ok {
		return v
	}
	return d
}

func parseKeyMap(s string) map[string]string {
	m := map[string]string{}
	for _, p := range strings.Split(s, ",") {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		kv := strings.SplitN(p, ":", 2)
		if len(kv) != 2 {
			continue
		}
		m[strings.TrimSpace(kv[0])] = strings.TrimSpace(kv[1])
	}
	return m
}

func asBool(s string) bool {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func New() *Config {
	keys := parseKeyMap(env("JWT_KEYS", ""))
	if len(keys) == 0 {
		if s := env("JWT_HS256_SECRET", ""); s != "" {
			keys["default"] = s
		}
	}
	skew, _ := time.ParseDuration(env("JWT_SKEW", "2m"))
	backoff, _ := time.ParseDuration(env("BACKOFF_MAX", "30s"))
	ret, _ := time.ParseDuration(env("LOG_RETENTION", "168h"))

	var max int64 = 64 << 20 // 64MB
	if v := env("LOG_MAX_BYTES", "67108864"); v != "" {
		var x int64
		for _, c := range v {
			if c < '0' || c > '9' {
				x = max
				break
			}
			x = x*10 + int64(c-'0')
		}
		if x > 0 {
			max = x
		}
	}

	return &Config{
		Port:           env("PORT", "8080"),
		AllowedOrigins: strings.Split(env("CORS_ORIGINS", "http://localhost:5173,http://localhost:3000"), ","),
		JWTKeys:        keys,
		Skew:           skew,
		MockEnabled:    asBool(env("MOCK_ENABLED", "true")),
		BackoffMax:     backoff,
		MetricsUser:    env("METRICS_USER", ""),
		MetricsPass:    env("METRICS_PASS", ""),
		LogPath:        env("LOG_PATH", "./data/events.log"),
		LogMaxBytes:    max,
		LogRetention:   ret,
		KafkaBrokers:   splitTrim(env("KAFKA_BROKERS", "")),
		KafkaTopic:     env("KAFKA_TOPIC", "orders"),
		KafkaGroup:     env("KAFKA_GROUP", "orderpulse"),
		KafkaEnabled:   asBool(env("KAFKA_ENABLED", "false")),
		AmqpURL:        env("AMQP_URL", "amqp://guest:guest@localhost:5672/"),
		AmqpQueue:      env("AMQP_QUEUE", "orders"),
		AmqpEnabled:    asBool(env("AMQP_ENABLED", "false")),
	}
}

func splitTrim(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
