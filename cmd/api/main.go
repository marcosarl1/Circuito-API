package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/marcosarl1/Circuito-API/internal/config"
	"github.com/marcosarl1/Circuito-API/internal/handler"
	"github.com/marcosarl1/Circuito-API/internal/repository"
)

func main() {
	cfg := config.Load()
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", handler.Health)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	store, err := repository.Connect(ctx, cfg)
	if err != nil {
		slog.Warn("mongo unavailable at boot", "err", err)
		store = nil
	} else if err := store.EnsureIndexes(ctx); err != nil {
		slog.Warn("indexes failed", "err", err)
	}
	mux.HandleFunc("GET /ready", handler.Ready(store))

	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	go func() {
		slog.Info("listening", "port", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("serve failed", "err", err)
			os.Exit(1)
		}
	}()
	<-ctx.Done()
	shutCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutCtx)

	fmt.Printf("port=%s db=%s coll=%s\n", cfg.Port, cfg.MongoDB, cfg.MongoCollection)
}
