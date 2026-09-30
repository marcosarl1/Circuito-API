package service

import (
	"encoding/json"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

func TestStringSliceUnmarshalsBSONString(t *testing.T) {
	data, err := bson.Marshal(bson.M{"value": "3km (caminhada), 5km e 10km (corrida)"})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded struct {
		Value StringSlice `bson:"value"`
	}
	if err := bson.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if len(decoded.Value) != 2 {
		t.Fatalf("expected 2 items, got %d: %#v", len(decoded.Value), decoded.Value)
	}

	if decoded.Value[0] != "3km (caminhada)" {
		t.Fatalf("unexpected item: %q", decoded.Value[0])
	}
}

func TestStringSliceUnmarshalsBSONArray(t *testing.T) {
	data, err := bson.Marshal(bson.M{"value": bson.A{"5 KM", "10 KM"}})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded struct {
		Value StringSlice `bson:"value"`
	}
	if err := bson.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if len(decoded.Value) != 2 || decoded.Value[1] != "10 KM" {
		t.Fatalf("unexpected value: %#v", decoded.Value)
	}
}

func TestStringSliceUnmarshalsBSONMixedArray(t *testing.T) {
	data, err := bson.Marshal(bson.M{"value": bson.A{5, " 10 KM ", 21.5}})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded struct {
		Value StringSlice `bson:"value"`
	}
	if err := bson.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if len(decoded.Value) != 3 || decoded.Value[0] != "5" || decoded.Value[2] != "21.5" {
		t.Fatalf("unexpected value: %#v", decoded.Value)
	}
}

func TestStringSliceUnmarshalsJSON(t *testing.T) {
	var slice StringSlice

	if err := json.Unmarshal([]byte(`["5 KM","10 KM"]`), &slice); err != nil {
		t.Fatalf("array: %v", err)
	}
	if len(slice) != 2 {
		t.Fatalf("array: unexpected %#v", slice)
	}

	if err := json.Unmarshal([]byte(`"5 KM, 10 KM"`), &slice); err != nil {
		t.Fatalf("string: %v", err)
	}
	if len(slice) != 2 || slice[0] != "5 KM" {
		t.Fatalf("string: unexpected %#v", slice)
	}

	if err := json.Unmarshal([]byte(`null`), &slice); err != nil {
		t.Fatalf("null: %v", err)
	}
	if slice == nil || len(slice) != 0 {
		t.Fatalf("null: expected empty slice, got %#v", slice)
	}
}

func TestKitUnmarshalsBSONWithStringDate(t *testing.T) {
	data, err := bson.Marshal(bson.M{
		"nome":          "Kit Básico",
		"itens":         bson.A{"Camiseta", "Medalha"},
		"data_retirada": "2026-01-24T00:00:00",
	})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var kit Kit
	if err := bson.Unmarshal(data, &kit); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if kit.Nome != "Kit Básico" {
		t.Fatalf("unexpected nome: %q", kit.Nome)
	}
	if len(kit.Itens) != 2 {
		t.Fatalf("unexpected itens: %#v", kit.Itens)
	}
	if kit.DataRetirada == nil {
		t.Fatal("expected data_retirada to be parsed")
	}
	if kit.DataRetirada.Format("2006-01-02") != "2026-01-24" {
		t.Fatalf("unexpected date: %v", kit.DataRetirada)
	}
}

func TestKitUnmarshalsBSONWithUnparseableDate(t *testing.T) {
	data, err := bson.Marshal(bson.M{
		"nome":          "Kit",
		"data_retirada": "data indefinida",
	})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var kit Kit
	if err := bson.Unmarshal(data, &kit); err != nil {
		t.Fatalf("unmarshal must not fail the document: %v", err)
	}
	if kit.DataRetirada != nil {
		t.Fatalf("expected nil date, got %v", kit.DataRetirada)
	}
}

func TestEventUnmarshalsLegacyDocument(t *testing.T) {
	data, err := bson.Marshal(bson.M{
		"_id":         "2026020001",
		"nome_evento": "Evento legado",
		"distancias":  "3km (caminhada), 5km e 10km (corrida)",
		"kits": bson.A{
			bson.M{"nome": "Kit", "data_retirada": "2026-01-24T00:00:00"},
		},
	})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var evento Event
	if err := bson.Unmarshal(data, &evento); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	evento.Normalize()

	if len(evento.Distancias) != 2 {
		t.Fatalf("unexpected distancias: %#v", evento.Distancias)
	}
	if evento.Kits[0].DataRetirada == nil {
		t.Fatal("expected kit date to be parsed")
	}
	if evento.ListaPrecos == nil {
		t.Fatal("expected lista_precos to be initialized")
	}
}

func TestParseFlexibleDateTime(t *testing.T) {
	moment := time.Date(2026, 5, 15, 0, 0, 0, 0, time.UTC)

	if parsed := parseFlexibleDateTime(moment); parsed == nil || !parsed.Equal(moment) {
		t.Fatalf("time.Time: unexpected %v", parsed)
	}
	if parseFlexibleDateTime(nil) != nil {
		t.Fatal("nil: expected nil")
	}
	if parseFlexibleDateTime(123) != nil {
		t.Fatal("int: expected nil")
	}
	if parsed := parseFlexibleDateTime("2026-05-15"); parsed == nil {
		t.Fatal("date string: expected parsed value")
	}
}
