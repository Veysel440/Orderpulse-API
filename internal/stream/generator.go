package stream

import (
	"context"
	"math/rand"
	"time"

	"orderpulse-api/internal/models"

	"github.com/google/uuid"
)

type Generator struct {
	Hub *Hub
}

func (g *Generator) Run(ctx context.Context) {
	t := time.NewTicker(50 * time.Millisecond)
	defer t.Stop()
	status := []string{"pending", "paid", "failed", "shipped"}
	evtypes := []string{"order.created", "status_changed", "order.packed", "order.shipped"}
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			ev := models.OrderEvent{
				ID:      uuid.NewString(),
				OrderID: uuid.NewString()[0:8],
				Type:    evtypes[rand.Intn(len(evtypes))],
				Status:  status[rand.Intn(len(status))],
				Amount:  10 + rand.Intn(990),
				TS:      time.Now().UTC(),
			}
			g.Hub.Publish(ev)
		}
	}
}
