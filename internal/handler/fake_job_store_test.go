package handler

import (
	"context"

	"github.com/marcosarl1/Circuito-API/internal/repository"
	"github.com/marcosarl1/Circuito-API/internal/service"
)

type fakeJobStore struct {
	jobs    map[string]*service.ScrapeJob
	locked  bool
	lastRun *string
}

func newFakeJobStore() *fakeJobStore {
	return &fakeJobStore{jobs: map[string]*service.ScrapeJob{}}
}

func (store *fakeJobStore) AcquireScrapeJob(_ context.Context, jobID, startedAt string) (*service.ScrapeJob, error) {
	if store.locked {
		return nil, repository.ErrScrapeInProgress
	}
	store.locked = true
	job := &service.ScrapeJob{JobID: jobID, Status: service.JobStatusQueued, StartedAt: startedAt}
	store.jobs[jobID] = job
	return job, nil
}

func (store *fakeJobStore) GetScrapeJob(_ context.Context, jobID string) (*service.ScrapeJob, error) {
	job, exists := store.jobs[jobID]
	if !exists {
		return nil, repository.ErrScrapeJobNotFound
	}
	return job, nil
}

func (store *fakeJobStore) GetLastScrapeRun(_ context.Context) (*string, error) {
	return store.lastRun, nil
}

func (store *fakeJobStore) AbandonScrapeJob(_ context.Context, jobID, reason string) error {
	job, exists := store.jobs[jobID]
	if !exists {
		return repository.ErrScrapeJobNotFound
	}
	job.Status = service.JobStatusFailed
	job.Error = &reason
	store.locked = false
	return nil
}
