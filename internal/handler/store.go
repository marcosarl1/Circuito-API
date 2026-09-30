package handler

import (
	"context"

	"github.com/marcosarl1/Circuito-API/internal/service"
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
}
