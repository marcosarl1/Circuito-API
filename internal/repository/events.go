package repository

import (
	"context"
	"errors"
	"regexp"
	"time"

	"github.com/marcosarl1/Circuito-API/internal/service"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

var ErrEventNotFound = errors.New("event not found")

func (store *Store) ListEvents(requestContext context.Context, page, size int64, estado, search string) ([]service.Event, int64, error) {
	requestContext, cancel := context.WithTimeout(requestContext, 5*time.Second)
	defer cancel()
	filter := bson.M{}
	if estado != "" {
		filter["estado"] = estado
	}
	if search != "" {
		escapedSearch := regexp.QuoteMeta(search)
		filter["$or"] = []bson.M{
			{"nome_evento": bson.M{"$regex": escapedSearch, "$options": "i"}},
			{"cidade": bson.M{"$regex": escapedSearch, "$options": "i"}},
		}
	}
	total, err := store.Collection.CountDocuments(requestContext, filter)
	if err != nil {
		return nil, 0, err
	}
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 20
	}
	cursor, err := store.Collection.Find(requestContext, filter, options.Find().SetSort(bson.D{{Key: "datas_realizacao", Value: -1}}).SetSkip((page-1)*size).SetLimit(size))
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(requestContext)
	var eventos []service.Event
	if err := cursor.All(requestContext, &eventos); err != nil {
		return nil, 0, err
	}
	return eventos, total, nil
}

func (store *Store) FindEvent(requestContext context.Context, eventID string) (*service.Event, error) {
	requestContext, cancel := context.WithTimeout(requestContext, 5*time.Second)
	defer cancel()
	normalizedID, err := service.NormalizeEventID(eventID)
	if err != nil {
		return nil, err
	}
	var evento service.Event
	err = store.Collection.FindOne(requestContext, bson.M{"_id": normalizedID}).Decode(&evento)

	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, ErrEventNotFound
	}

	if err != nil {
		return nil, err
	}

	return &evento, nil
}

func (store *Store) CreateEvent(requestContext context.Context, newEvent service.Event) (*service.Event, error) {
	requestContext, cancel := context.WithTimeout(requestContext, 5*time.Second)
	defer cancel()
	if _, err := store.Collection.InsertOne(requestContext, newEvent); err != nil {
		return nil, err
	}
	return &newEvent, nil
}

func (store *Store) UpdateEvent(requestContext context.Context, eventID string, updates map[string]any) (*service.Event, error) {
	requestContext, cancel := context.WithTimeout(requestContext, 5*time.Second)
	defer cancel()

	normalizedID, err := service.NormalizeEventID(eventID)
	if err != nil {
		return nil, err
	}

	updateDocument := bson.M{"$set": updates}

	updateOptions := options.FindOneAndUpdate().SetReturnDocument(options.After)

	var updatedEvent service.Event

	err = store.Collection.FindOneAndUpdate(
		requestContext,
		bson.M{"_id": normalizedID},
		updateDocument,
		updateOptions).Decode(&updatedEvent)

	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, ErrEventNotFound
	}

	if err != nil {
		return nil, err
	}

	return &updatedEvent, nil
}

func (store *Store) DeleteEvent(requestContext context.Context, eventID string) (bool, error) {
	requestContext, cancel := context.WithTimeout(requestContext, 5*time.Second)
	defer cancel()
	normalizedID, err := service.NormalizeEventID(eventID)
	if err != nil {
		return false, err
	}
	deleteResult, err := store.Collection.DeleteOne(requestContext, bson.M{"_id": normalizedID})
	if err != nil {
		return false, err
	}
	return deleteResult.DeletedCount == 1, nil
}
