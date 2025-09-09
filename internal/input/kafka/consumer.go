package kafka

import (
	"context"
	"encoding/json"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/segmentio/kafka-go"
	"orderpulse-api/internal/models"
	"orderpulse-api/internal/stream"
)

type Consumer struct {
	reader *kafka.Reader
	hub    *stream.Hub
}

func New(brokers []string, topic, group string, hub *stream.Hub) *Consumer {
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  brokers,
		Topic:    topic,
		GroupID:  group,
		MaxBytes: 10e6,
	})
	return &Consumer{reader: r, hub: hub}
}

func (c *Consumer) Run(ctx context.Context) error {
	defer c.reader.Close()
	for {
		m, err := c.reader.ReadMessage(ctx)
		if err != nil {
			return err
		}
		ev := models.OrderEvent{}
		if err := json.Unmarshal(m.Value, &ev); err != nil {
			log.Warn().Err(err).Msg("kafka decode")
			continue
		}
		if ev.TS.IsZero() {
			ev.TS = time.Now().UTC()
		}
		c.hub.Publish(ev)
	}
}
