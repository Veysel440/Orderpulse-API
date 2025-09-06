package httpx

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"orderpulse-api/internal/stream"

	"github.com/gorilla/websocket"
)

func WS(allowedOrigins []string, hub *stream.Hub) http.HandlerFunc {
	upgrader := websocket.Upgrader{
		Subprotocols: []string{"bearer"},
		CheckOrigin: func(r *http.Request) bool {
			origin := r.Header.Get("Origin")
			if origin == "" { // CLI/servers
				return true
			}
			for _, o := range allowedOrigins {
				o = strings.TrimSpace(o)
				if o == "*" || strings.EqualFold(o, origin) {
					return true
				}
			}
			return false
		},
	}

	return func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		conn.SetReadLimit(1)

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
