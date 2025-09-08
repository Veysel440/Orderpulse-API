package httpx

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/httprate"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog/log"
	"orderpulse-api/pkg/jwt"
)

type ctxKey string

const (
	CtxSub   ctxKey = "sub"
	CtxReqID ctxKey = "reqID"
)

var (
	httpReqs = prometheus.NewCounterVec(
		prometheus.CounterOpts{Name: "http_requests_total", Help: "count"},
		[]string{"method", "path", "status"},
	)
	httpDur = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{Name: "http_request_seconds", Help: "latency", Buckets: prometheus.DefBuckets},
		[]string{"method", "path"},
	)
)

func init() {
	prometheus.MustRegister(httpReqs, httpDur)
}

func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rr := &respRecorder{ResponseWriter: w, status: 200}
		next.ServeHTTP(rr, r)
		httpReqs.WithLabelValues(r.Method, r.URL.Path, http.StatusText(rr.status)).Inc()
		httpDur.WithLabelValues(r.Method, r.URL.Path).Observe(time.Since(start).Seconds())
		log.Info().
			Str("req_id", r.Header.Get("X-Request-Id")).
			Str("method", r.Method).
			Str("path", r.URL.Path).
			Int("status", rr.status).
			Dur("dur", time.Since(start)).
			Msg("http")
	})
}

type respRecorder struct {
	http.ResponseWriter
	status int
}

func (r *respRecorder) WriteHeader(code int) { r.status = code; r.ResponseWriter.WriteHeader(code) }

func Rate(limit int, per time.Duration) func(http.Handler) http.Handler {
	return httprate.LimitByIP(limit, per)
}

func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-Id")
		if id == "" {
			id = uuid.NewString()
		}
		w.Header().Set("X-Request-Id", id)
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), CtxReqID, id)))
	})
}

func SecureHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")
		next.ServeHTTP(w, r)
	})
}

func Recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				WriteError(w, http.StatusInternalServerError, "panic", "internal error")
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func BodyLimit(n int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.Body = http.MaxBytesReader(w, r.Body, n)
			next.ServeHTTP(w, r)
		})
	}
}

func extractToken(r *http.Request) string {
	if h := r.Header.Get("Authorization"); h != "" {
		parts := strings.SplitN(h, " ", 2)
		if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
			return parts[1]
		}
	}
	if v := r.URL.Query().Get("access_token"); v != "" {
		return v
	}
	if v := r.URL.Query().Get("token"); v != "" {
		return v
	}
	return ""
}

func Auth(optional bool, v *jwt.Validator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tok := extractToken(r)
			if tok == "" && !optional {
				WriteError(w, http.StatusUnauthorized, "unauthorized", "missing token")
				return
			}
			if tok != "" {
				sub, err := v.Validate(tok)
				if err != nil && !optional {
					WriteError(w, http.StatusUnauthorized, "unauthorized", "invalid token")
					return
				}
				if sub != "" {
					r = r.WithContext(context.WithValue(r.Context(), CtxSub, sub))
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}
