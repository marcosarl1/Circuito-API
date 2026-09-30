package service

import (
	"testing"
	"time"
)

func dashboardFixture(now time.Time) []Event {
	image := "https://example.com/img.jpg"
	link := "https://example.com/inscricao"
	edital := "https://example.com/edital.pdf"
	horario := "07:00"
	preco := "R$ 120,00"
	coletada := now.AddDate(0, 0, -1)

	return []Event{
		{
			ID:              "2026090001",
			NomeEvento:      "Corrida Futura",
			Cidade:          "joão pessoa",
			Estado:          "pb",
			Organizador:     "Org A",
			SiteColeta:      "brasilquecorre",
			DataColeta:      &coletada,
			DataRealizacao:  "",
			DatasRealizacao: []time.Time{now.AddDate(0, 0, 10)},
			Distancias:      StringSlice{"5 KM", "10 KM"},
			Horario:         &horario,
			URLInscricao:    &link,
			URLImagem:       &image,
			LinkEdital:      &edital,
			PrecosEntries:   []any{"Lote 1 - R$ 120,00", "Lote 2 - R$ 150,00"},
			Preco:           &preco,
			Patrocinado:     true,
		},
		{
			ID:              "2025010002",
			NomeEvento:      "Corrida Passada",
			Cidade:          "Campina Grande",
			Estado:          "PB",
			Organizador:     "Org A",
			SiteColeta:      "manual",
			DataRealizacao:  "10 de Janeiro de 2025",
			DatasRealizacao: []time.Time{},
			Distancias:      StringSlice{},
		},
	}
}

func TestComputeDashboardStatsTotals(t *testing.T) {
	now := time.Date(2026, time.September, 24, 12, 0, 0, 0, time.UTC)
	stats := ComputeDashboardStats(dashboardFixture(now), now)

	if stats.Total != 2 {
		t.Fatalf("total: got %d", stats.Total)
	}
	if stats.Ativos != 1 {
		t.Fatalf("ativos: got %d", stats.Ativos)
	}
	if stats.Passados != 1 {
		t.Fatalf("passados: got %d", stats.Passados)
	}
	if stats.Proximos30d != 1 {
		t.Fatalf("proximos30d: got %d", stats.Proximos30d)
	}
	if stats.SemPreco != 1 {
		t.Fatalf("semPreco: got %d", stats.SemPreco)
	}
	if stats.Patrocinados != 1 {
		t.Fatalf("patrocinados: got %d", stats.Patrocinados)
	}
	if stats.SemImagem != 1 {
		t.Fatalf("semImagem: got %d", stats.SemImagem)
	}
	if stats.Lote1Count != 1 {
		t.Fatalf("lote1Count: got %d", stats.Lote1Count)
	}
	if stats.ValorMedio != 135 {
		t.Fatalf("valorMedio: got %v", stats.ValorMedio)
	}
	if stats.StatusInscricoes.Abertas != 1 || stats.StatusInscricoes.Encerradas != 1 {
		t.Fatalf("status: got %+v", stats.StatusInscricoes)
	}
	if len(stats.PorEstado) != 1 || stats.PorEstado[0].Estado != "PB" || stats.PorEstado[0].Count != 2 {
		t.Fatalf("porEstado: got %+v", stats.PorEstado)
	}
	if len(stats.PorCidade) != 2 {
		t.Fatalf("porCidade: got %+v", stats.PorCidade)
	}
	if stats.PorCidade[0].Cidade != "João Pessoa" {
		t.Fatalf("cidade display: got %+v", stats.PorCidade)
	}
	if len(stats.PorDistancia) != 2 {
		t.Fatalf("porDistancia: got %+v", stats.PorDistancia)
	}
	if len(stats.ProximosEventos) != 1 || stats.ProximosEventos[0].ID != "2026090001" {
		t.Fatalf("proximosEventos: got %+v", stats.ProximosEventos)
	}
	if len(stats.ScraperHealth) != 1 || stats.ScraperHealth[0].Fonte != "brasilquecorre" {
		t.Fatalf("scraperHealth: got %+v", stats.ScraperHealth)
	}
	if len(stats.PorHorario) != 1 || stats.PorHorario[0].Label != "07:00" {
		t.Fatalf("porHorario: got %+v", stats.PorHorario)
	}
	if len(stats.PorMes) != 2 || stats.PorMes[0].Label != "2025-01" || stats.PorMes[1].Label != "2026-10" {
		t.Fatalf("porMes: got %+v", stats.PorMes)
	}
}

func TestComputeDashboardStatsEmpty(t *testing.T) {
	now := time.Date(2026, time.September, 24, 12, 0, 0, 0, time.UTC)
	stats := ComputeDashboardStats([]Event{}, now)

	if stats.Total != 0 || stats.ValorMedio != 0 {
		t.Fatalf("empty: got %+v", stats)
	}
	if stats.PorMes == nil || stats.ProximosEventos == nil || stats.ScraperHealth == nil {
		t.Fatal("empty: slices must be initialized, not nil")
	}
}

func TestParseDataRealizacaoPT(t *testing.T) {
	parsed := parseDataRealizacao("04 de Setembro de 2026", nil)
	if parsed == nil || parsed.Format("2006-01-02") != "2026-09-04" {
		t.Fatalf("pt date: got %v", parsed)
	}
	if parseDataRealizacao("texto inválido", nil) != nil {
		t.Fatal("invalid: expected nil")
	}
}

func TestParsePrecoValor(t *testing.T) {
	value, found := parsePrecoValor("Lote 1 - R$ 1.234,56")
	if !found || value != 1234.56 {
		t.Fatalf("got %v, %v", value, found)
	}
	if _, found := parsePrecoValor("Gratuito"); found {
		t.Fatal("expected no value")
	}
}
