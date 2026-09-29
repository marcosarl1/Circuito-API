package service

import "testing"

func TestNormalizeDistanciasFromString(t *testing.T) {
	distancias := normalizeDistancias(" 5 KM, 10KM ,, 21K ")

	if len(distancias) != 3 {
		t.Fatalf("expected 3 items, got %d: %#v", len(distancias), distancias)
	}

	if distancias[0] != "5 KM" {
		t.Fatalf("expected %q, got %q", "5KM", distancias[0])
	}

	if distancias[2] != "21K" {
		t.Fatalf("expected %q, got %q", "21K", distancias[2])
	}
}

func TestNormalizeDistanciasFromEmptyString(t *testing.T) {
	distancias := normalizeDistancias("   ")

	if distancias == nil {
		t.Fatal("expected empty slice, got nil")
	}

	if len(distancias) != 0 {
		t.Fatalf("expected 0 items, got %d", len(distancias))
	}
}

func TestNormalizeDistanciasFromMixedSlice(t *testing.T) {
	distancias := normalizeDistancias([]any{5, " 10 KM ", 21.5, "  "})

	if len(distancias) != 3 {
		t.Fatalf("expected 3 items, got %d: %#v", len(distancias), distancias)
	}

	if distancias[0] != "5" {
		t.Fatalf("expected %q, got %q", "5", distancias[0])
	}

	if distancias[2] != "21.5" {
		t.Fatalf("expected %q, got %q", "21.5", distancias[2])
	}
}

func TestParsePrecoesEntriesFromJSONString(t *testing.T) {
	entries := parsePrecosEntries(`["Lote 1 - R$ 120,00", "Lote 2 - R$ 150,00"]`)

	if len(entries) != 2 {
		t.Fatalf("expected 2 atnries, got %d: %#v", len(entries), entries)
	}

	first, ok := entries[0].(string)
	if !ok {
		t.Fatalf("expected string, got %T", entries[0])
	}

	if first != "Lote 1 - R$ 120,00" {
		t.Fatalf("unexpected entry: %q", first)
	}
}

func TestParsePRecoesEntriesFromInvalidJSONString(t *testing.T) {
	entries := parsePrecosEntries("não é json")

	if entries == nil {
		t.Fatal("expected empty slice, got nil")
	}

	if len(entries) != 0 {
		t.Fatalf("expected 0 entries, got %d", len(entries))
	}
}

func TestParsePrecosEntriesFromNil(t *testing.T) {
	entries := parsePrecosEntries(nil)

	if entries == nil {
		t.Fatal("expected empty slice, got nil")
	}
}

func TestValidateHorarioAcceptsValidValue(t *testing.T) {
	value := "07:30"

	if err := validateHorario(&value); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if value != "07:30" {
		t.Fatalf("expected %q, got %q", "07:30", value)
	}
}

func TestValidateHorarioRejectsInvalidValue(t *testing.T) {
	value := "7:30"

	if err := validateHorario(&value); err == nil {
		t.Fatal("expected error for invalid horario")
	}
}

func TestValidateHorarioRejectsOutOfRangeValue(t *testing.T) {
	value := "99:99"

	if err := validateHorario(&value); err == nil {
		t.Fatal("expected error for out of range horario")
	}
}
