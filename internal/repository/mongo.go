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

func Connect(ctx context.Context, cfg config.Config) (*Store, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	client, err := mongo.Connect(options.Client().ApplyURI(cfg.MongoURI))
	if err != nil {
		return nil, err
	}
	if err := client.Ping(ctx, nil); err != nil {
		return nil, err
	}
	db := client.Database(cfg.MongoDB)
	return &Store{
		Client:     client,
		DB:         db,
		Collection: db.Collection(cfg.MongoCollection),
		Counters:   db.Collection("counters"),
	}, nil
}

func (s *Store) EnsureIndexes(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	_, err := s.Collection.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "datas_realizacao", Value: -1}}},
		{Keys: bson.D{{Key: "estado", Value: 1}, {Key: "datas_realizacao", Value: -1}}},
		{Keys: bson.D{{Key: "nome_evento", Value: 1}}},
		{Keys: bson.D{{Key: "cidade", Value: 1}}},
	})
	return err
}

func (s *Store) NextEventID(ctx context.Context, now time.Time) (string, error) {
	prefix := service.EventPrefix(now)
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	var res struct {
		Seq int64 `bson:"seq"`
	}
	err := s.Counters.FindOneAndUpdate(
		ctx,
		bson.M{"_id": prefix},
		bson.M{"$inc": bson.M{"seq": 1}},
		options.FindOneAndUpdate().SetUpsert(true).SetReturnDocument(options.After),
	).Decode(&res)
	if err != nil {
		return "", fmt.Errorf("counter: %w", err)
	}
	return service.BuildEventID(prefix, res.Seq)
}
