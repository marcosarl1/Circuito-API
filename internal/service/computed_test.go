package service

import (
	"testing"
	"time"
)

func TestFormatDataRealizacao(t *testing.T) {
	value := time.Date(2026, time.September, 4, 0, 0, 0, 0, time.UTC)

	formatted := formatDataRealizacao(value)

	if formatted != "04 de Setembro de 2026" {
		t.Fatalf(
			"expected %q, got %q",
			"04 de Setembro de 2026",
			formatted,
		)
	}
}

func TestSplitCategorias(t *testing.T) {
	categorias := splitCategorias("Corrida de Rua,  Corrida de Trilha ,")

	if len(categorias) != 2 {
		t.Fatalf("expected 2 items, got %d: %#v", len(categorias), categorias)
	}

	if categorias[0] != "Corrida de Rua" {
		t.Fatalf("unexpected item: %q", categorias[0])
	}

	if categorias[1] != "Corrida de Trilha" {
		t.Fatalf("unexpected item: %q", categorias[1])
	}
}

func TestNormalizeDerivesDataRealizacao(t *testing.T) {
	event := Event{
		DatasRealizacao: []time.Time{
			time.Date(2026, time.September, 4, 0, 0, 0, 0, time.UTC),
		},
	}

	event.Normalize()

	if event.DataRealizacao != "04 de Setembro de 2026" {
		t.Fatalf(
			"expected derived data_realizacao, got %q",
			event.DataRealizacao,
		)
	}
}

func TestNormalizeKeepsExistingDataRealizacao(t *testing.T) {
	event := Event{
		DataRealizacao: "Texto Manual",
		DatasRealizacao: []time.Time{
			time.Date(2026, time.September, 4, 0, 0, 0, 0, time.UTC),
		},
	}

	event.Normalize()

	if event.DataRealizacao != "Texto Manual" {
		t.Fatalf(
			"expected existing value to be kept, got %q",
			event.DataRealizacao,
		)
	}
}

func TestExtractPriceFromText(t *testing.T) {
	result := extractPriceFromText("Lote 1 - R$ 120,00")

	if result != "LOTE 1 — R$ 120,00" {
		t.Fatalf(
			"expected %q, got %q",
			"LOTE 1 — R$ 120,00",
			result,
		)
	}
}

func TestExtractPriceFromTextWithoutLabel(t *testing.T) {
	result := extractPriceFromText("R$ 120,00")

	if result != "R$ 120,00" {
		t.Fatalf("expected %q, got %q", "R$ 120,00", result)
	}
}

func TestExtractPriceFromTextWithoutPrice(t *testing.T) {
	result := extractPriceFromText("Gratuito")

	if result != "Gratuito" {
		t.Fatalf("expected %q, got %q", "Gratuito", result)
	}
}

func TestFormatBRL(t *testing.T) {
	testCases := []struct {
		value    float64
		expected string
	}{
		{value: 120, expected: "R$ 120,00"},
		{value: 1234.56, expected: "R$ 1.234,56"},
		{value: 1000000, expected: "R$ 1.000.000,00"},
		{value: 99.9, expected: "R$ 99,90"},
	}

	for _, testCase := range testCases {
		result := formatBRL(testCase.value)

		if result != testCase.expected {
			t.Fatalf(
				"formatBRL(%v): expected %q, got %q",
				testCase.value,
				testCase.expected,
				result,
			)
		}
	}
}

func TestFormatTextEntryWithPipeSeparator(t *testing.T) {
	result := formatTextEntry("R$ 120,00 | Lote 1")

	if result != "LOTE 1 — R$ 120,00" {
		t.Fatalf(
			"expected %q, got %q",
			"LOTE 1 — R$ 120,00",
			result,
		)
	}
}

func TestFormatPriceListRemovesDuplicates(t *testing.T) {
	result := formatPriceList(
		[]any{
			"Lote 1 - R$ 120,00",
			"Lote 1 - R$ 120,00",
			"Lote 2 - R$ 150,00",
		},
		nil,
	)

	if len(result) != 2 {
		t.Fatalf("expected 2 items, got %d: %#v", len(result), result)
	}
}

func TestFormatPriceListFallsBackToRawPrice(t *testing.T) {
	rawPrice := "R$ 90,00; R$ 100,00"

	result := formatPriceList([]any{}, &rawPrice)

	if len(result) != 2 {
		t.Fatalf("expected 2 items, got %d: %#v", len(result), result)
	}

	if result[0] != "R$ 90,00" {
		t.Fatalf("unexpected item: %q", result[0])
	}
}

func TestFormatPriceListReturnsEmptyWhenNoData(t *testing.T) {
	result := formatPriceList(nil, nil)

	if result == nil {
		t.Fatal("expected empty slice, got nil")
	}

	if len(result) != 0 {
		t.Fatalf("expected 0 items, got %d", len(result))
	}
}
