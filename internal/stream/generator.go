package stream

import (
	"context"
	"math/rand/v2"
	"time"

	"github.com/google/uuid"
	"orderpulse-api/internal/models"
)

type Generator struct{ Hub *Hub }

func (g *Generator) Run(ctx context.Context) {
	t := time.NewTicker(50 * time.Millisecond)
	defer t.Stop()

	status := []string{"pending", "paid", "failed", "shipped"}
	types := []string{"order.created", "status_changed", "order.packed", "order.shipped"}

	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			ev := models.OrderEvent{
				ID:      uuid.NewString(),
				OrderID: uuid.NewString()[:8],
				Type:    types[rand.IntN(len(types))],
				Status:  status[rand.IntN(len(status))],
				Amount:  10 + rand.IntN(990),
				TS:      time.Now().UTC(),
			}
			g.Hub.Publish(ev)
		}
	}
}
