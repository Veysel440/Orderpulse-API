package httpx

import (
	"context"
	"net/http"
	"strings"
	"time"

	"orderpulse-api/pkg/jwt"

	"github.com/go-chi/httprate"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

type ctxKey string

const (
	CtxSub   ctxKey = "sub"
	CtxReqID ctxKey = "reqID"
)

func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Info().
			Str("method", r.Method).
			Str("path", r.URL.Path).
			Str("req_id", w.Header().Get("X-Request-Id")).
			Dur("dur", time.Since(start)).
			Msg("req")
	})
}

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
		ctx := context.WithValue(r.Context(), CtxReqID, id)
		next.ServeHTTP(w, r.WithContext(ctx))
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

func SecureHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")
		next.ServeHTTP(w, r)
	})
}

func Auth(optional bool, v *jwt.Validator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h := r.Header.Get("Authorization")
			if h == "" && !optional {
				http.Error(w, "missing token", http.StatusUnauthorized)
				return
			}
			if h != "" {
				parts := strings.SplitN(h, " ", 2)
				if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
					http.Error(w, "bad auth", http.StatusUnauthorized)
					return
				}
				sub, err := v.Validate(parts[1])
				if err != nil && !optional {
					http.Error(w, "invalid token", http.StatusUnauthorized)
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
