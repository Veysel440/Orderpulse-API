package main

import (
	"context"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"orderpulse-api/internal/config"
	httpx "orderpulse-api/internal/http"
	kcons "orderpulse-api/internal/input/kafka"
	acons "orderpulse-api/internal/input/rabbitmq"
	"orderpulse-api/internal/logstore"
	"orderpulse-api/internal/stream"
)

func main() {
	zerolog.TimeFieldFormat = time.RFC3339
	cfg := config.New()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	store, err := logstore.NewFileStore(cfg.LogPath, cfg.LogMaxBytes, cfg.LogRetention)
	if err != nil {
		log.Fatal().Err(err).Msg("logstore")
	}

	hub := stream.NewHub(store)

	if cfg.MockEnabled {
		gen := &stream.Generator{Hub: hub}
		go gen.Run(ctx)
	}
	if cfg.KafkaEnabled && len(cfg.KafkaBrokers) > 0 {
		go func() {
			log.Info().Strs("brokers", cfg.KafkaBrokers).Str("topic", cfg.KafkaTopic).Msg("kafka consume")
			if err := kcons.New(cfg.KafkaBrokers, cfg.KafkaTopic, cfg.KafkaGroup, hub).Run(ctx); err != nil {
				log.Error().Err(err).Msg("kafka")
			}
		}()
	}
	if cfg.AmqpEnabled {
		go func() {
			log.Info().Str("queue", cfg.AmqpQueue).Msg("amqp consume")
			if err := acons.New(cfg.AmqpURL, cfg.AmqpQueue, hub).Run(ctx); err != nil {
				log.Error().Err(err).Msg("amqp")
			}
		}()
	}

	srv := &http.Server{Addr: ":" + cfg.Port, Handler: httpx.Router(cfg, hub)}
	go func() {
		log.Info().Str("addr", srv.Addr).Str("log", cfg.LogPath).Msg("listening")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("server")
		}
	}()

	<-ctx.Done()
	shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdown)
}
