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

func (store *Store) scrapeJobs() *mongo.Collection {
	return store.DB.Collection("scrape_jobs")
}

func (store *Store) scrapeState() *mongo.Collection {
	return store.DB.Collection("scrape_state")
}

func (store *Store) AcquireScrapeJob(requestContext context.Context, jobID, startedAt string) (*service.ScrapeJob, error) {
	requestContext, cancel := context.WithTimeout(requestContext, 5*time.Second)
	defer cancel()

	job := &service.ScrapeJob{
		JobID:     jobID,
		Status:    service.JobStatusQueued,
		StartedAt: startedAt,
		Active:    true,
	}
	if _, err := store.scrapeJobs().InsertOne(requestContext, job); err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return nil, ErrScrapeInProgress
		}
		return nil, err
	}
	return job, nil
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
