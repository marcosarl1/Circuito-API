package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/marcosarl1/Circuito-API/internal/config"
	"github.com/marcosarl1/Circuito-API/internal/service"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type Store struct {
	Client     *mongo.Client
	DB         *mongo.Database
	Collection *mongo.Collection
	Counters   *mongo.Collection
}

func Connect(requestContext context.Context, appConfig config.Config) (*Store, error) {
	requestContext, cancel := context.WithTimeout(requestContext, 10*time.Second)
	defer cancel()
	client, err := mongo.Connect(options.Client().ApplyURI(appConfig.MongoURI))
	if err != nil {
		return nil, err
	}
	if err := client.Ping(requestContext, nil); err != nil {
		return nil, err
	}
	database := client.Database(appConfig.MongoDB)
	return &Store{
		Client:     client,
		DB:         database,
		Collection: database.Collection(appConfig.MongoCollection),
		Counters:   database.Collection("counters"),
	}, nil
}

func (store *Store) EnsureIndexes(requestContext context.Context) error {
	requestContext, cancel := context.WithTimeout(requestContext, 30*time.Second)
	defer cancel()
	_, err := store.Collection.Indexes().CreateMany(requestContext, []mongo.IndexModel{
		{Keys: bson.D{{Key: "datas_realizacao", Value: -1}}},
		{Keys: bson.D{{Key: "estado", Value: 1}, {Key: "datas_realizacao", Value: -1}}},
		{Keys: bson.D{{Key: "nome_evento", Value: 1}}},
		{Keys: bson.D{{Key: "cidade", Value: 1}}},
	})
	return err
}

func (store *Store) NextEventID(requestContext context.Context, now time.Time) (string, error) {
	prefix := service.EventPrefix(now)
	requestContext, cancel := context.WithTimeout(requestContext, 5*time.Second)
	defer cancel()
	var counterResult struct {
		Seq int64 `bson:"seq"`
	}
	err := store.Counters.FindOneAndUpdate(
		requestContext,
		bson.M{"_id": prefix},
		bson.M{"$inc": bson.M{"seq": 1}},
		options.FindOneAndUpdate().SetUpsert(true).SetReturnDocument(options.After),
	).Decode(&counterResult)
	if err != nil {
		return "", fmt.Errorf("counter: %w", err)
	}
	return service.BuildEventID(prefix, counterResult.Seq)
}
