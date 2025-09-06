package stream

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"orderpulse-api/internal/models"
)

func parseSince(r *http.Request) time.Time {
	if id := r.Header.Get("Last-Event-ID"); id != "" {
		if ns, err := strconv.ParseInt(id, 10, 64); err == nil {
			return time.Unix(0, ns)
		}
	}
	if s := r.URL.Query().Get("since"); s != "" {
		if d, err := time.ParseDuration(s); err == nil {
			return time.Now().Add(-d)
		}
		if t, err := time.Parse(time.RFC3339, s); err == nil {
			return t
		}
	}
	return time.Time{}
}

func SSE(hub *Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		types := map[string]struct{}{}
		statuses := map[string]struct{}{}
		for _, t := range strings.Split(r.URL.Query().Get("types"), ",") {
			if t = strings.TrimSpace(t); t != "" {
				types[t] = struct{}{}
			}
		}
		for _, s := range strings.Split(r.URL.Query().Get("statuses"), ",") {
			if s = strings.TrimSpace(s); s != "" {
				statuses[s] = struct{}{}
			}
		}
		filter := func(e models.OrderEvent) bool {
			if len(types) > 0 {
				if _, ok := types[e.Type]; !ok {
					return false
				}
			}
			if len(statuses) > 0 {
				if _, ok := statuses[e.Status]; !ok {
					return false
				}
			}
			return true
		}

		ctx, cancel := context.WithCancel(r.Context())
		defer cancel()
		sub := hub.Subscribe(ctx, 512)

		if since := parseSince(r); !since.IsZero() {
			hub.ReplaySince(since, sub)
		}

		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "stream unsupported", http.StatusInternalServerError)
			return
		}

		ping := time.NewTicker(15 * time.Second)
		defer ping.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ping.C:
				_, _ = fmt.Fprintf(w, ": ping\n\n")
				flusher.Flush()
			case ev := <-sub:
				if !filter(ev) {
					break
				}
				b, _ := json.Marshal(ev)
				_, _ = fmt.Fprintf(w, "id: %d\n", ev.TS.UnixNano())
				_, _ = fmt.Fprintf(w, "event: order\n")
				_, _ = fmt.Fprintf(w, "data: %s\n\n", b)
				flusher.Flush()
			}
		}
	}
}
