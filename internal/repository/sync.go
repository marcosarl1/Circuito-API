package repository

import (
	"context"
	"errors"
	"time"

	"github.com/marcosarl1/Circuito-API/internal/service"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func (store *Store) GetAllEvents(requestContext context.Context) ([]service.Event, error) {
	requestContext, cancel := context.WithTimeout(requestContext, 30*time.Second)
	defer cancel()
	cursor, err := store.Collection.Find(requestContext, bson.M{}, options.Find().SetBatchSize(500))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(requestContext)
	var eventos []service.Event
	if err := cursor.All(requestContext, &eventos); err != nil {
		return nil, err
	}
	for index := range eventos {
		eventos[index].Normalize()
	}
	if eventos == nil {
		eventos = []service.Event{}
	}
	return eventos, nil
}

func (store *Store) GetBucketSyncState(requestContext context.Context) (*service.SyncState, error) {
	requestContext, cancel := context.WithTimeout(requestContext, 5*time.Second)
	defer cancel()
	var state service.SyncState
	err := store.DB.Collection("bucket_sync").FindOne(requestContext, bson.M{"_id": "state"}).Decode(&state)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, err
	}
	return &state, nil
}

func (store *Store) SetBucketSyncState(requestContext context.Context, state service.SyncState) error {
	requestContext, cancel := context.WithTimeout(requestContext, 5*time.Second)
	defer cancel()
	_, err := store.DB.Collection("bucket_sync").UpdateOne(
		requestContext,
		bson.M{"_id": "state"},
		bson.M{"$set": bson.M{"sha256": state.SHA256, "eventos": state.Eventos, "synced_at": state.SyncedAt}},
		options.UpdateOne().SetUpsert(true),
	)
	return err
}
