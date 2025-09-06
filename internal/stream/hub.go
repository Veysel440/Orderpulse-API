package stream

import (
	"context"
	"sync"
	"time"

	"orderpulse-api/internal/models"
)

type Subscriber chan models.OrderEvent

type Hub struct {
	mu      sync.RWMutex
	subs    map[Subscriber]struct{}
	hmu     sync.RWMutex
	hist    []models.OrderEvent
	histMax int
	histTTL time.Duration
}

func NewHub() *Hub {
	return &Hub{
		subs:    make(map[Subscriber]struct{}),
		hist:    make([]models.OrderEvent, 0, 5000),
		histMax: 5000,
		histTTL: time.Hour,
	}
}

func (h *Hub) Subscribe(ctx context.Context, buf int) Subscriber {
	ch := make(Subscriber, buf)
	h.mu.Lock()
	h.subs[ch] = struct{}{}
	h.mu.Unlock()
	go func() {
		<-ctx.Done()
		h.mu.Lock()
		delete(h.subs, ch)
		close(ch)
		h.mu.Unlock()
	}()
	return ch
}

func (h *Hub) Publish(ev models.OrderEvent) {
	h.mu.RLock()
	for ch := range h.subs {
		select {
		case ch <- ev:
		default:
		}
	}
	h.mu.RUnlock()

	h.hmu.Lock()
	cut := time.Now().Add(-h.histTTL)
	h.hist = append(h.hist, ev)
	if len(h.hist) > h.histMax {
		h.hist = h.hist[len(h.hist)-h.histMax:]
	}
	i := 0
	for ; i < len(h.hist) && h.hist[i].TS.Before(cut); i++ {
	}
	if i > 0 && i < len(h.hist) {
		h.hist = h.hist[i:]
	}
	h.hmu.Unlock()
}

func (h *Hub) ReplaySince(since time.Time, out Subscriber) {
	h.hmu.RLock()
	defer h.hmu.RUnlock()
	for _, ev := range h.hist {
		if ev.TS.After(since) {
			select {
			case out <- ev:
			default:
			}
		}
	}
}
