package httpx

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"orderpulse-api/internal/stream"
	"orderpulse-api/pkg/jwt"
)

func WS(allowedOrigins []string, hub *stream.Hub, v *jwt.Validator) http.HandlerFunc {
	upgrader := websocket.Upgrader{
		Subprotocols: []string{"bearer"},
		CheckOrigin: func(r *http.Request) bool {
			origin := r.Header.Get("Origin")
			if origin == "" {
				return true
			}
			for _, o := range allowedOrigins {
				if strings.EqualFold(strings.TrimSpace(o), origin) {
					return true
				}
			}
			return false
		},
	}

	return func(w http.ResponseWriter, r *http.Request) {
		tok := extractToken(r)
		if _, err := v.Validate(tok); err != nil {
			WriteError(w, http.StatusUnauthorized, "unauthorized", "invalid token")
			return
		}

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		sub := hub.Subscribe(r.Context(), 256)

		tick := time.NewTicker(15 * time.Second)
		defer tick.Stop()
		go func() {
			for range tick.C {
				_ = conn.WriteControl(websocket.PingMessage, []byte("ping"), time.Now().Add(5*time.Second))
			}
		}()

		for {
			select {
			case <-r.Context().Done():
				return
			case ev := <-sub:
				b, _ := json.Marshal(ev)
				if err := conn.WriteMessage(websocket.TextMessage, b); err != nil {
					return
				}
			}
		}
	}
}
