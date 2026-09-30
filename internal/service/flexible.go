package service

import (
	"encoding/json"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

type StringSlice []string

func (slice *StringSlice) UnmarshalBSONValue(bsonType byte, data []byte) error {
	switch bson.Type(bsonType) {
	case bson.TypeString:
		var value string
		if err := bson.UnmarshalValue(bson.TypeString, data, &value); err != nil {
			*slice = StringSlice{}
			return nil
		}
		*slice = StringSlice(normalizeDistancias(value))
		return nil
	case bson.TypeArray:
		var items []string
		if err := bson.UnmarshalValue(bson.TypeArray, data, &items); err == nil {
			*slice = StringSlice(normalizeDistancias(items))
			return nil
		}
		var generic []any
		if err := bson.UnmarshalValue(bson.TypeArray, data, &generic); err != nil {
			*slice = StringSlice{}
			return nil
		}
		normalized := make([]string, 0, len(generic))
		for _, item := range generic {
			if trimmed := strings.TrimSpace(toLegacyString(item)); trimmed != "" {
				normalized = append(normalized, trimmed)
			}
		}
		*slice = StringSlice(normalized)
		return nil
	default:
		*slice = StringSlice{}
		return nil
	}
}

// UnmarshalJSON accepts an array, a comma-separated string, or null.
func (slice *StringSlice) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		*slice = StringSlice{}
		return nil
	}
	var items []string
	if err := json.Unmarshal(data, &items); err == nil {
		*slice = StringSlice(normalizeDistancias(items))
		return nil
	}
	var value string
	if err := json.Unmarshal(data, &value); err == nil {
		*slice = StringSlice(normalizeDistancias(value))
		return nil
	}
	var generic []any
	if err := json.Unmarshal(data, &generic); err != nil {
		*slice = StringSlice{}
		return nil
	}
	normalized := make([]string, 0, len(generic))
	for _, item := range generic {
		if trimmed := strings.TrimSpace(toLegacyString(item)); trimmed != "" {
			normalized = append(normalized, trimmed)
		}
	}
	*slice = StringSlice(normalized)
	return nil
}

func (kit *Kit) UnmarshalBSON(data []byte) error {
	var raw struct {
		Nome          string      `bson:"nome"`
		Itens         StringSlice `bson:"itens"`
		LocalRetirada *string     `bson:"local_retirada"`
		DataRetirada  any         `bson:"data_retirada"`
	}
	if err := bson.Unmarshal(data, &raw); err != nil {
		return err
	}
	kit.Nome = raw.Nome
	kit.Itens = []string(raw.Itens)
	kit.LocalRetirada = raw.LocalRetirada
	kit.DataRetirada = parseFlexibleDateTime(raw.DataRetirada)
	return nil
}

func parseFlexibleDateTime(value any) *time.Time {
	switch typedValue := value.(type) {
	case nil:
		return nil
	case time.Time:
		moment := typedValue
		return &moment
	case bson.DateTime:
		moment := typedValue.Time()
		return &moment
	case string:
		trimmed := strings.TrimSpace(typedValue)
		if trimmed == "" {
			return nil
		}
		for _, layout := range []string{
			time.RFC3339,
			"2006-01-02T15:04:05",
			"2006-01-02",
			"02/01/2006",
		} {
			if parsed, err := time.Parse(layout, trimmed); err == nil {
				return &parsed
			}
		}
		return nil
	default:
		return nil
	}
}
