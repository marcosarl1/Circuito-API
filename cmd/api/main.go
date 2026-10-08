package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
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

// seedAdmin creates the first admin account from env when the users
// collection is empty. It runs once: existing deployments ignore the vars.
// The password must satisfy the same policy as user-chosen passwords;
// rotate ADMIN_PASSWORD right after the first login.
func seedAdmin(requestContext context.Context, store *repository.Store, appConfig config.Config) {
	if appConfig.AdminEmail == "" || appConfig.AdminPassword == "" {
		return
	}
	count, err := store.CountUsers(requestContext)
	if err != nil {
		slog.Error("admin seed failed: cannot count users", "err", err)
		return
	}
	if count > 0 {
		return
	}
	if err := service.ValidateNewPassword(appConfig.AdminPassword); err != nil {
		slog.Error("admin seed failed: weak ADMIN_PASSWORD", "err", err)
		return
	}
	hash, err := service.HashPassword(appConfig.AdminPassword)
	if err != nil {
		slog.Error("admin seed failed", "err", err)
		return
	}
	user := service.User{
		ID:        service.NewUserID(),
		Email:     strings.ToLower(strings.TrimSpace(appConfig.AdminEmail)),
		Role:      service.RoleAdmin,
		CreatedAt: time.Now().UTC(),
	}
	user.PasswordHash = hash
	if _, err := store.CreateUser(requestContext, user); err != nil {
		slog.Error("admin seed failed", "err", err)
		return
	}
	slog.Info("admin account seeded", "email", user.Email)
}

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
	} else {
		if err := store.EnsureIndexes(appContext); err != nil {
			slog.Warn("indexes failed", "err", err)
		}
		if err := store.EnsureAuthIndexes(appContext); err != nil {
			slog.Warn("auth indexes failed", "err", err)
		}
		seedAdmin(appContext, store, appConfig)
	}
	var eventStore handler.EventStore
	var jobStore handler.JobStore
	var userStore handler.UserStore
	if store != nil {
		eventStore = store
		jobStore = store
		userStore = store
	}
	router.HandleFunc("GET /ready", handler.Ready(store))

	router.HandleFunc("GET /api/v1/eventos", handler.ListEvents(eventStore))
	router.HandleFunc("GET /api/v1/eventos/{id}", handler.GetEvent(eventStore))

	auth := handler.AuthConfig{UserStore: userStore, JWTSecret: appConfig.JWTSecret}
	limitedLogin := middleware.RateLimit(5, time.Minute)(handler.Login(auth))
	router.HandleFunc("POST /api/v1/auth/login", limitedLogin.ServeHTTP)
	router.HandleFunc("POST /api/v1/auth/refresh", handler.Refresh(auth))
	router.HandleFunc("POST /api/v1/auth/logout", handler.Logout(auth))
	router.HandleFunc("GET /api/v1/auth/me", handler.RequireUser(auth)(handler.Me()))
	router.HandleFunc("PATCH /api/v1/auth/password", handler.RequireUser(auth)(handler.ChangePassword(auth)))

	// Transition guard: user JWT preferred, service X-API-Key accepted.
	// Remove the key path once the frontend logs in with JWT.
	requireWriter := handler.RequireUserOrKey(auth, appConfig.APIKey)
	router.HandleFunc("POST /api/v1/eventos", requireWriter(handler.CreateEvent(eventStore, func(requestContext context.Context) (string, error) {
		return store.NextEventID(requestContext, time.Now())
	})))
	router.HandleFunc("GET /api/v1/dashboard/stats", handler.DashboardStats(eventStore))

	router.HandleFunc("PATCH /api/v1/eventos/{id}", requireWriter(handler.UpdateEvent(eventStore)))

	router.HandleFunc("DELETE /api/v1/eventos/{id}", requireWriter(handler.DeleteEvent(eventStore)))

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
	router.HandleFunc("GET /api/v1/sync-bucket/status", requireWriter(handler.SyncBucketStatus(syncer)))
	router.HandleFunc("POST /api/v1/sync-bucket", requireWriter(handler.SyncBucket(syncer)))

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
