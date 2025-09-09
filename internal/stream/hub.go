package stream

import (
	"context"
	"sync"
	"time"

	"orderpulse-api/internal/logstore"
	"orderpulse-api/internal/models"
)

type Subscriber chan models.OrderEvent

type Hub struct {
	mu    sync.RWMutex
	subs  map[Subscriber]struct{}
	store logstore.Store
}

func NewHub(store logstore.Store) *Hub {
	return &Hub{subs: make(map[Subscriber]struct{}), store: store}
}

func (h *Hub) Subscribe(ctx context.Context, buf int) Subscriber {
	ch := make(Subscriber, buf)
	h.mu.Lock()
	h.subs[ch] = struct{}{}
	h.mu.Unlock()
	subsGauge.Inc()

	go func() {
		<-ctx.Done()
		h.mu.Lock()
		delete(h.subs, ch)
		close(ch)
		h.mu.Unlock()
		subsGauge.Dec()
	}()
	return ch
}

func (h *Hub) Publish(ev models.OrderEvent) {
	h.mu.RLock()
	for ch := range h.subs {
		select {
		case ch <- ev:
		default:
			dropsCtr.Inc()
		}
	}
	h.mu.RUnlock()
	if h.store != nil {
		_ = h.store.Append(ev)
	}
}

func (h *Hub) ReplaySince(since time.Time, out Subscriber) {
	if h.store == nil {
		return
	}
	_ = h.store.ReplaySince(since, func(ev models.OrderEvent) bool {
		select {
		case out <- ev:
		default:
			dropsCtr.Inc()
		}
		return true
	})
}
