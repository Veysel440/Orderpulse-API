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
	"orderpulse-api/internal/logstore"
	"orderpulse-api/internal/stream"
)

func main() {
	zerolog.TimeFieldFormat = time.RFC3339
	cfg := config.New()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	store, err := logstore.NewFileStore(cfg.LogPath)
	if err != nil {
		log.Fatal().Err(err).Msg("logstore")
	}

	hub := stream.NewHub(store)
	if cfg.MockEnabled {
		gen := &stream.Generator{Hub: hub}
		go gen.Run(ctx)
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
