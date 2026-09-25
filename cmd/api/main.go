package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/marcosarl1/Circuito-API/internal/config"
	"github.com/marcosarl1/Circuito-API/internal/handler"
	"github.com/marcosarl1/Circuito-API/internal/middleware"
	"github.com/marcosarl1/Circuito-API/internal/repository"
)

func main() {
	appConfig, err := config.Load()
	if err != nil {
		slog.Error("configuration load failed", "err", err)
		os.Exit(1)
	}
	router := http.NewServeMux()
	router.HandleFunc("GET /health", handler.Health)

	appContext, stopSignals := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stopSignals()

	store, err := repository.Connect(appContext, appConfig)
	if err != nil {
		slog.Warn("mongo unavailable at boot", "err", err)
		store = nil
	} else if err := store.EnsureIndexes(appContext); err != nil {
		slog.Warn("indexes failed", "err", err)
	}
	var eventStore handler.EventStore
	if store != nil {
		eventStore = store
	}
	router.HandleFunc("GET /ready", handler.Ready(store))

	router.HandleFunc("GET /api/v1/eventos", handler.ListEvents(eventStore))
	router.HandleFunc("GET /api/v1/eventos/{id}", handler.GetEvent(eventStore))

	requireAPIKey := handler.RequireAPIKey(appConfig.APIKey)
	router.HandleFunc("POST /api/v1/eventos", requireAPIKey(handler.CreateEvent(eventStore, func(requestContext context.Context) (string, error) {
		return store.NextEventID(requestContext, time.Now())
	})))

	router.HandleFunc("PATCH /api/v1/eventos/{id}", requireAPIKey(handler.UpdateEvent(eventStore)))

	router.HandleFunc("DELETE /api/v1/eventos/{id}", requireAPIKey(handler.DeleteEvent(eventStore)))

	server := &http.Server{
		Addr:         ":" + appConfig.Port,
		Handler:      middleware.RequestID(middleware.CORS(appConfig.CorsOrigin)(router)),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	go func() {
		slog.Info("listening", "port", appConfig.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("serve failed", "err", err)
			os.Exit(1)
		}
	}()
	<-appContext.Done()
	shutdownContext, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = server.Shutdown(shutdownContext)
}
