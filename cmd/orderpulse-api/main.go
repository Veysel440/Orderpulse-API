package main

import (
	"context"
	"net/http"
	_ "os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"orderpulse-api/internal/config"
	httpx "orderpulse-api/internal/http"
	"orderpulse-api/internal/stream"
)

func main() {
	zerolog.TimeFieldFormat = time.RFC3339

	cfg := config.New()
	hub := stream.NewHub()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if cfg.MockEnabled {
		gen := &stream.Generator{Hub: hub}
		go gen.Run(ctx)
	}

	srv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: httpx.Router(cfg, hub),
	}

	go func() {
		log.Info().Str("addr", srv.Addr).Msg("listening")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("server")
		}
	}()

	<-ctx.Done()
	shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdown)
}
