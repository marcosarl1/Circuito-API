package repository

import (
	"context"
	"regexp"
	"time"

	"github.com/marcosarl1/Circuito-API/internal/service"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func (s *Store) ListEvents(ctx context.Context, page, size int64, estado, q string) ([]service.Evento, int64, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	filter := bson.M{}
	if estado != "" {
		filter["estado"] = estado
	}
	if q != "" {
		rx := regexp.QuoteMeta(q)
		filter["$or"] = []bson.M{
			{"nome_evento": bson.M{"$regex": rx, "$options": "i"}},
			{"cidade": bson.M{"$regex": rx, "$options": "i"}},
		}
	}
	total, err := s.Collection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 20
	}
	cur, err := s.Collection.Find(ctx, filter, options.Find().SetSort(bson.D{{Key: "datas_realizacao", Value: -1}}).SetSkip((page-1)*size).SetLimit(size))
	if err != nil {
		return nil, 0, err
	}
	defer cur.Close(ctx)
	var out []service.Evento
	if err := cur.All(ctx, &out); err != nil {
		return nil, 0, err
	}
	return out, total, nil
}

func (s *Store) FindEvent(ctx context.Context, id string) (*service.Evento, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	nid, err := service.NormalizeEventID(id)
	if err != nil {
		return nil, err
	}
	var e service.Evento
	if err := s.Collection.FindOne(ctx, bson.M{"_id": nid}).Decode(&e); err != nil {
		return nil, err
	}
	return &e, nil
}
