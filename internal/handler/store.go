package handler

import (
	"context"

	"github.com/marcosarl1/Circuito-API/internal/repository"
	"github.com/marcosarl1/Circuito-API/internal/service"
)

// Compile-time proofs that the Mongo store satisfies the handler contracts.
var (
	_ EventStore = (*repository.Store)(nil)
	_ JobStore   = (*repository.Store)(nil)
	_ UserStore  = (*repository.Store)(nil)
)

type EventStore interface {
	ListEvents(
		requestContext context.Context,
		page int64,
		size int64,
		estado string,
		search string) ([]service.Event, int64, error)

	FindEvent(
		requestContext context.Context,
		eventID string) (*service.Event, error)

	CreateEvent(
		requestContext context.Context,
		newEvent service.Event) (*service.Event, error)

	UpdateEvent(
		requestContext context.Context,
		eventID string,
		updates map[string]any) (*service.Event, error)

	DeleteEvent(
		requestContext context.Context,
		eventID string) (bool, error)

	GetDashboardEvents(
		requestContext context.Context) ([]service.Event, error)
}

// JobStore abstracts scrape job persistence so handlers stay testable
// without MongoDB. *repository.Store implements it.
type JobStore interface {
	AcquireScrapeJob(
		requestContext context.Context,
		jobID string,
		startedAt string) (*service.ScrapeJob, error)

	GetScrapeJob(
		requestContext context.Context,
		jobID string) (*service.ScrapeJob, error)

	GetLastScrapeRun(
		requestContext context.Context) (*string, error)

	AbandonScrapeJob(
		requestContext context.Context,
		jobID string,
		reason string) error

	ConfirmScrapeJob(
		requestContext context.Context,
		jobID string) (*service.ScrapeJob, error)

	FindAwaitingScrapeJob(
		requestContext context.Context) (*service.ScrapeJob, error)

	DeleteScrapePayload(
		requestContext context.Context,
		jobID string) error
}

// UserStore abstracts admin account and session persistence so auth
// handlers stay testable without MongoDB. *repository.Store implements it.
type UserStore interface {
	CountUsers(
		requestContext context.Context) (int64, error)

	CreateUser(
		requestContext context.Context,
		user service.User) (*service.User, error)

	FindUserByUsername(
		requestContext context.Context,
		username string) (*service.User, error)

	FindUserByID(
		requestContext context.Context,
		userID string) (*service.User, error)

	StoreRefreshToken(
		requestContext context.Context,
		token service.RefreshToken) error

	TakeRefreshToken(
		requestContext context.Context,
		hash string) (*service.RefreshToken, error)

	RevokeRefreshToken(
		requestContext context.Context,
		hash string) error

	UpdateUserPassword(
		requestContext context.Context,
		userID string,
		hash string) error
}
