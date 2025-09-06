package main

import (
	"context"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/rs/zerolog/log"

	"orderpulse-api/internal/config"
	httpx "orderpulse-api/internal/http"
	"orderpulse-api/internal/stream"
)

func main() {
	cfg := config.New()
	httpx.SetupLogger("info")

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
		log.Info().Str("port", cfg.Port).Msg("listening")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("server")
		}
	}()

	<-ctx.Done()
	stop()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)
}
