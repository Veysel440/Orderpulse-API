package rabbitmq

import (
	"context"
	"encoding/json"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/rs/zerolog/log"
	"orderpulse-api/internal/models"
	"orderpulse-api/internal/stream"
)

type Consumer struct {
	url   string
	queue string
	hub   *stream.Hub
}

func New(url, queue string, hub *stream.Hub) *Consumer {
	return &Consumer{url: url, queue: queue, hub: hub}
}

func (c *Consumer) Run(ctx context.Context) error {
	conn, err := amqp.Dial(c.url)
	if err != nil {
		return err
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		return err
	}
	defer ch.Close()

	msgs, err := ch.Consume(c.queue, "", true, false, false, false, nil)
	if err != nil {
		return err
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case m, ok := <-msgs:
			if !ok {
				return nil
			}
			ev := models.OrderEvent{}
			if err := json.Unmarshal(m.Body, &ev); err != nil {
				log.Warn().Err(err).Msg("amqp decode")
				continue
			}
			if ev.TS.IsZero() {
				ev.TS = time.Now().UTC()
			}
			c.hub.Publish(ev)
		}
	}
}
