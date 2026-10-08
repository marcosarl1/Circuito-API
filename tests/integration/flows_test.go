//go:build integration

package integration

import (
	"testing"
)

func TestEventLifecycle(t *testing.T) {
	id := createEvent(t, "Evento integração", "João Pessoa")

	decoded := status(t, "GET", "/api/v1/eventos/"+id, nil, "", 200)
	if decoded["id"] != id {
		t.Fatalf("GET devolveu id diferente: %v", decoded["id"])
	}

	decoded = status(t, "PATCH", "/api/v1/eventos/"+id,
		map[string]any{"cidade": "Campina Grande"}, apiKey, 200)
	if decoded["cidade"] != "Campina Grande" {
		t.Fatalf("PATCH não aplicou: %v", decoded["cidade"])
	}

	listed := status(t, "GET", "/api/v1/eventos?q=Campina+Grande&size=100", nil, "", 200)
	found := false
	for _, item := range listed["eventos"].([]any) {
		if item.(map[string]any)["id"] == id {
			found = true
		}
	}
	if !found {
		t.Fatal("id criado não aparece na listagem")
	}

	status(t, "DELETE", "/api/v1/eventos/"+id, nil, apiKey, 204)
	status(t, "GET", "/api/v1/eventos/"+id, nil, "", 404)
}

// TestEventAuthErrors garante 401 sem chave e 404/400 nos erros de entrada.
func TestEventAuthErrors(t *testing.T) {
	id := createEvent(t, "Evento auth", "Patos")

	status(t, "POST", "/api/v1/eventos", map[string]any{"nome_evento": "x"}, "", 401)
	status(t, "POST", "/api/v1/eventos", map[string]any{"nome_evento": "x"}, "chave-errada", 401)
	status(t, "PATCH", "/api/v1/eventos/"+id, map[string]any{"cidade": "y"}, "", 401)
	status(t, "DELETE", "/api/v1/eventos/"+id, nil, "", 401)

	status(t, "GET", "/api/v1/eventos/2099010001", nil, "", 404)
	status(t, "PATCH", "/api/v1/eventos/2099010001", map[string]any{"cidade": "y"}, apiKey, 404)
	status(t, "DELETE", "/api/v1/eventos/2099010001", nil, apiKey, 404)
	status(t, "PATCH", "/api/v1/eventos/"+id, map[string]any{}, apiKey, 400)
}

// TestPaginationContract valida o formato da paginação, o clamp de size
// e a página além do fim (lista vazia, nunca null).
func TestPaginationContract(t *testing.T) {
	decoded := status(t, "GET", "/api/v1/eventos?page=1&size=1000", nil, "", 200)
	for _, field := range []string{"eventos", "total", "total_pages", "page", "size", "has_next", "has_prev"} {
		if _, exists := decoded[field]; !exists {
			t.Fatalf("campo ausente: %s", field)
		}
	}
	if decoded["size"] != float64(20) {
		t.Fatalf("size=1000 deveria limitar a 20, veio %v", decoded["size"])
	}

	decoded = status(t, "GET", "/api/v1/eventos?page=999999&size=20", nil, "", 200)
	eventos, ok := decoded["eventos"].([]any)
	if !ok {
		t.Fatalf("eventos deveria ser array, veio %T", decoded["eventos"])
	}
	if len(eventos) != 0 {
		t.Fatalf("página além do fim deveria ser vazia, veio %d itens", len(eventos))
	}
}

// TestDashboardContract valida a presença das chaves do contrato.
func TestDashboardContract(t *testing.T) {
	decoded := status(t, "GET", "/api/v1/dashboard/stats", nil, "", 200)
	for _, field := range []string{
		"total", "ativos", "passados", "proximos30d", "proximos90d",
		"semPreco", "patrocinados", "semImagem", "semLink", "semRegulamento",
		"valorMedio", "lote1Count", "porMes", "porEstado", "porCidade",
		"porDistancia", "porOrganizador", "porFonte", "densidade", "choques",
		"statusInscricoes", "comPercurso", "comKits", "porHorario", "porKit",
		"scraperHealth", "proximosEventos",
	} {
		if _, exists := decoded[field]; !exists {
			t.Fatalf("chave ausente no dashboard: %s", field)
		}
	}
}

// TestScrapeFlow cobre run (202 ou 409 se já houver job), status, 404,
// last-run e o 501 do import sem worker.
func TestScrapeFlow(t *testing.T) {
	code, decoded := do(t, "POST", "/api/v1/scrape/run", nil, scrapersKey)
	if code != 202 && code != 409 {
		t.Fatalf("run: esperado 202 ou 409, obtido %d (%v)", code, decoded)
	}

	if code == 202 {
		jobID, _ := decoded["job_id"].(string)
		if jobID == "" {
			t.Fatalf("run 202 sem job_id: %v", decoded)
		}
		detail := status(t, "GET", "/api/v1/scrape/status/"+jobID, nil, scrapersKey, 200)
		if detail["job_id"] != jobID {
			t.Fatalf("status com job_id divergente: %v", detail["job_id"])
		}
		if detail["status"] != "queued" && detail["status"] != "running" {
			t.Fatalf("status inesperado: %v", detail["status"])
		}
	}

	status(t, "GET", "/api/v1/scrape/status/job-inexistente", nil, scrapersKey, 404)
	status(t, "POST", "/api/v1/scrape/run", nil, "", 401)

	lastRun := status(t, "GET", "/api/v1/scrape/last-run", nil, scrapersKey, 200)
	if _, exists := lastRun["finished_at"]; !exists {
		t.Fatal("last-run sem chave finished_at")
	}

	status(t, "POST", "/api/v1/scrape/import", nil, scrapersKey, 501)
}

// TestSyncStatus valida o status do bucket sem disparar upload real.
// Sem bucket configurado no ambiente de teste, o POST retorna 500.
func TestSyncStatus(t *testing.T) {
	decoded := status(t, "GET", "/api/v1/sync-bucket/status", nil, apiKey, 200)
	if _, exists := decoded["in_progress"]; !exists {
		t.Fatal("status sem chave in_progress")
	}
	status(t, "GET", "/api/v1/sync-bucket/status", nil, "", 401)

	decoded = status(t, "POST", "/api/v1/sync-bucket", nil, apiKey, 500)
	if decoded["detail"] != "AWS_BUCKET_NAME não configurado" {
		t.Fatalf("detail inesperado: %v", decoded)
	}
}
