package repository

import (
	"context"
	"errors"
	"time"

	"github.com/marcosarl1/Circuito-API/internal/service"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

var (
	ErrScrapeInProgress  = errors.New("scrape already in progress")
	ErrScrapeJobNotFound = errors.New("scrape job not found")
)

const scrapeLockID = "scrape"

func (store *Store) scrapeJobs() *mongo.Collection {
	return store.DB.Collection("scrape_jobs")
}

func (store *Store) scrapeLocks() *mongo.Collection {
	return store.DB.Collection("scrape_locks")
}

func (store *Store) scrapeState() *mongo.Collection {
	return store.DB.Collection("scrape_state")
}

func (store *Store) AcquireScrapeJob(requestContext context.Context, jobID, startedAt string) (*service.ScrapeJob, error) {
	requestContext, cancel := context.WithTimeout(requestContext, 5*time.Second)
	defer cancel()

	lock := bson.M{"_id": scrapeLockID, "job_id": jobID, "acquired_at": startedAt}
	if _, err := store.scrapeLocks().InsertOne(requestContext, lock); err != nil {
		if !mongo.IsDuplicateKeyError(err) {
			return nil, err
		}
		if reclaimed := store.reclaimStaleScrapeLock(requestContext); reclaimed {
			return store.AcquireScrapeJob(context.Background(), jobID, startedAt)
		}
		return nil, ErrScrapeInProgress
	}

	job := &service.ScrapeJob{
		JobID:     jobID,
		Status:    service.JobStatusQueued,
		StartedAt: startedAt,
	}
	if _, err := store.scrapeJobs().InsertOne(requestContext, job); err != nil {
		_, _ = store.scrapeLocks().DeleteOne(context.Background(), bson.M{"_id": scrapeLockID, "job_id": jobID})
		return nil, err
	}
	return job, nil
}

func (store *Store) reclaimStaleScrapeLock(requestContext context.Context) bool {
	var lock struct {
		JobID string `bson:"job_id"`
	}
	if err := store.scrapeLocks().FindOne(requestContext, bson.M{"_id": scrapeLockID}).Decode(&lock); err != nil {
		return false
	}
	job, err := store.GetScrapeJob(requestContext, lock.JobID)
	if err == nil && (job.Status == service.JobStatusQueued || job.Status == service.JobStatusRunning) {
		return false
	}
	_, err = store.scrapeLocks().DeleteOne(requestContext, bson.M{"_id": scrapeLockID, "job_id": lock.JobID})
	return err == nil
}

func (store *Store) GetScrapeJob(requestContext context.Context, jobID string) (*service.ScrapeJob, error) {
	requestContext, cancel := context.WithTimeout(requestContext, 5*time.Second)
	defer cancel()
	var job service.ScrapeJob
	err := store.scrapeJobs().FindOne(requestContext, bson.M{"_id": jobID}).Decode(&job)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrScrapeJobNotFound
		}
		return nil, err
	}
	return &job, nil
}

func (store *Store) GetLastScrapeRun(requestContext context.Context) (*string, error) {
	requestContext, cancel := context.WithTimeout(requestContext, 5*time.Second)
	defer cancel()
	var state struct {
		FinishedAt *string `bson:"finished_at"`
	}
	err := store.scrapeState().FindOne(requestContext, bson.M{"_id": "last_scrape"}).Decode(&state)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, err
	}
	return state.FinishedAt, nil
}
