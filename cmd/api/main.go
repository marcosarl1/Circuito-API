package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/marcosarl1/Circuito-API/internal/azure"
	"github.com/marcosarl1/Circuito-API/internal/config"
	"github.com/marcosarl1/Circuito-API/internal/handler"
	"github.com/marcosarl1/Circuito-API/internal/middleware"
	"github.com/marcosarl1/Circuito-API/internal/repository"
	"github.com/marcosarl1/Circuito-API/internal/service"
	"github.com/marcosarl1/Circuito-API/internal/storage"
)

const mongoShutdownTimeout = 5 * time.Second

// buildScrapeTrigger wires the automatic worker start. It returns nil when
// disabled (local dev, tests) so POST /scrape/run keeps queue-only behavior.
func buildScrapeTrigger(appConfig config.Config) func(context.Context) error {
	trigger := appConfig.ScraperTrigger
	if !trigger.Enabled {
		return nil
	}
	starter := &azure.JobStarter{
		SubscriptionID:   trigger.SubscriptionID,
		ResourceGroup:    trigger.ResourceGroup,
		JobName:          trigger.JobName,
		APIVersion:       "2023-05-01",
		ManagementURL:    "https://management.azure.com",
		IdentityEndpoint: os.Getenv("IDENTITY_ENDPOINT"),
		IdentityHeader:   os.Getenv("IDENTITY_HEADER"),
		HTTPClient:       &http.Client{Timeout: 30 * time.Second},
	}
	return starter.Start
}

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
	var jobStore handler.JobStore
	if store != nil {
		eventStore = store
		jobStore = store
	}
	router.HandleFunc("GET /ready", handler.Ready(store))

	router.HandleFunc("GET /api/v1/eventos", handler.ListEvents(eventStore))
	router.HandleFunc("GET /api/v1/eventos/{id}", handler.GetEvent(eventStore))
	router.HandleFunc("GET /api/v1/dashboard/stats", handler.DashboardStats(eventStore))

	requireAPIKey := handler.RequireAPIKey(appConfig.APIKey)
	router.HandleFunc("POST /api/v1/eventos", requireAPIKey(handler.CreateEvent(eventStore, func(requestContext context.Context) (string, error) {
		return store.NextEventID(requestContext, time.Now())
	})))

	router.HandleFunc("PATCH /api/v1/eventos/{id}", requireAPIKey(handler.UpdateEvent(eventStore)))

	router.HandleFunc("DELETE /api/v1/eventos/{id}", requireAPIKey(handler.DeleteEvent(eventStore)))

	requireScrapersKey := handler.RequireAPIKey(appConfig.ScrapersKey)
	scrapeRun := middleware.RateLimit(5, time.Minute)(handler.RunScrape(jobStore, service.NewScrapeJobID, buildScrapeTrigger(appConfig)))
	router.HandleFunc("POST /api/v1/scrape/run", requireScrapersKey(scrapeRun.ServeHTTP))
	router.HandleFunc("GET /api/v1/scrape/status/{id}", requireScrapersKey(handler.ScrapeStatus(jobStore)))
	router.HandleFunc("GET /api/v1/scrape/last-run", requireScrapersKey(handler.ScrapeLastRun(jobStore)))
	router.HandleFunc("POST /api/v1/scrape/import", requireScrapersKey(handler.ScrapeImport()))

	var syncer *storage.BucketSync
	if store != nil {
		syncer = storage.NewBucketSync(
			appConfig.AWSBucketName,
			appConfig.BucketJSONKey,
			store.GetAllEvents,
			store,
			storage.NewS3Uploader(appConfig.AWSRegion, appConfig.AWSAccessKeyID, appConfig.AWSSecretKey),
		)
	}
	router.HandleFunc("GET /api/v1/sync-bucket/status", requireAPIKey(handler.SyncBucketStatus(syncer)))
	router.HandleFunc("POST /api/v1/sync-bucket", requireAPIKey(handler.SyncBucket(syncer)))

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
	shutdownContext, cancelShutdown := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelShutdown()

	if err := server.Shutdown(shutdownContext); err != nil {
		slog.Error("server shutdown failed", "err", err)
	}

	if store != nil {
		mongoContext, cancelMongo := context.WithTimeout(context.Background(), mongoShutdownTimeout)
		defer cancelMongo()

		if err := store.Disconnect(mongoContext); err != nil {
			slog.Error("mongo disconnect failed", "err", err)
		}
	}
}
