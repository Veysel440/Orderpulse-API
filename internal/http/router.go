package httpx

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"orderpulse-api/internal/config"
	"orderpulse-api/internal/stream"
	"orderpulse-api/internal/telemetry"
	"orderpulse-api/pkg/jwt"
)

func Router(cfg *config.Config, hub *stream.Hub) http.Handler {
	r := chi.NewRouter()

	r.Use(Recoverer)
	r.Use(RequestID)
	r.Use(SecureHeaders)
	r.Use(Logger)
	r.Use(Rate(300, time.Minute))
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   cfg.AllowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders:   []string{"Authorization", "Content-Type"},
		AllowCredentials: false,
	}))

	val := jwt.NewValidator(cfg.JWTSecret)

	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("ok")) })
	r.Get("/readyz", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("ok")) })
	r.Method("GET", "/metrics", promhttp.Handler())

	r.Group(func(g chi.Router) {
		g.Use(Auth(false, val))
		g.Get("/api/stream/events", stream.SSE(hub))
	})

	r.Get("/api/ws", WS(cfg.AllowedOrigins, hub, val))

	r.Group(func(g chi.Router) {
		g.Use(Auth(true, val))
		g.Use(BodyLimit(64 << 10))
		g.Use(Rate(60, time.Minute))
		g.Post("/api/telemetry", telemetry.Handle)
	})

	return r
}
