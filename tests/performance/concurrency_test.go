//go:build perf

package performance

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"testing"
	"time"
)

// postEvent cria um evento e devolve status + id (vazio se falhou).
func postEvent(baseURL, apiKey, nome string) (int, string) {
	body, _ := json.Marshal(map[string]any{
		"nome_evento": nome,
		"cidade":      "João Pessoa",
		"estado":      "PB",
	})
	request, err := http.NewRequest("POST", baseURL+"/api/v1/eventos", bytes.NewReader(body))
	if err != nil {
		return 0, ""
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-API-Key", apiKey)
	response, err := client.Do(request)
	if err != nil {
		return 0, ""
	}
	defer response.Body.Close()
	raw, _ := io.ReadAll(response.Body)
	if response.StatusCode != 201 {
		return response.StatusCode, ""
	}
	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return response.StatusCode, ""
	}
	id, _ := decoded["id"].(string)
	return response.StatusCode, id
}

// TestConcurrentCreates dispara creates em paralelo: todos devem retornar
// 201 com IDs distintos. Um ID repetido indicaria corrida no gerador.
func TestConcurrentCreates(t *testing.T) {
	const workers = 50

	var mutex sync.Mutex
	ids := make(map[string]struct{}, workers)
	failures := 0

	var group sync.WaitGroup
	start := time.Now()
	for i := range workers {
		group.Add(1)
		go func(index int) {
			defer group.Done()
			status, id := postEvent(baseURL, apiKey, fmt.Sprintf("Evento concorrente %d", index))
			mutex.Lock()
			defer mutex.Unlock()
			if status != 201 || id == "" {
				failures++
				return
			}
			if _, exists := ids[id]; exists {
				t.Errorf("id duplicado: %s", id)
				return
			}
			ids[id] = struct{}{}
		}(i)
	}
	group.Wait()
	elapsed := time.Since(start)

	if failures > 0 {
		t.Fatalf("%d creates falharam", failures)
	}
	if len(ids) != workers {
		t.Fatalf("esperados %d ids distintos, obtidos %d", workers, len(ids))
	}
	t.Logf("concurrent creates: workers=%d distintos=%d total=%s", workers, len(ids), elapsed)

	for id := range ids {
		request, _ := http.NewRequest("DELETE", baseURL+"/api/v1/eventos/"+id, nil)
		request.Header.Set("X-API-Key", apiKey)
		if response, err := client.Do(request); err == nil {
			_, _ = io.Copy(io.Discard, response.Body)
			_ = response.Body.Close()
		}
	}
}

// TestConcurrentScrapeRun dispara runs em paralelo: exatamente um deve
// vencer (202) e os demais receber 409. Dois 202 indicariam lock quebrado.
func TestConcurrentScrapeRun(t *testing.T) {
	const workers = 20

	var mutex sync.Mutex
	accepted := 0
	rejected := 0

	var group sync.WaitGroup
	for range workers {
		group.Add(1)
		go func() {
			defer group.Done()
			request, err := http.NewRequest("POST", baseURL+"/api/v1/scrape/run", nil)
			if err != nil {
				return
			}
			request.Header.Set("X-API-Key", scrapersKey)
			response, err := client.Do(request)
			if err != nil {
				return
			}
			_, _ = io.Copy(io.Discard, response.Body)
			_ = response.Body.Close()
			mutex.Lock()
			defer mutex.Unlock()
			switch response.StatusCode {
			case 202:
				accepted++
			case 409, 429:
				// 409: lock ativo; 429: rate limiter (5/min) barrou a rajada.
				rejected++
			default:
				t.Errorf("status inesperado: %d", response.StatusCode)
			}
		}()
	}
	group.Wait()

	if accepted != 1 {
		t.Fatalf("esperado exatamente 1 aceite, obtidos %d", accepted)
	}
	if rejected != workers-1 {
		t.Fatalf("esperadas %d rejeições (409/429), obtidas %d", workers-1, rejected)
	}
	t.Logf("concurrent scrape run: accepted=%d rejected=%d", accepted, rejected)
}

// TestParallelReads mede latência sob leitura paralela.
func TestParallelReads(t *testing.T) {
	const workers = 100

	var mutex sync.Mutex
	var max time.Duration
	failures := 0

	var group sync.WaitGroup
	for range workers {
		group.Add(1)
		go func() {
			defer group.Done()
			start := time.Now()
			response, err := client.Get(baseURL + "/api/v1/eventos?page=1&size=20")
			if err != nil {
				mutex.Lock()
				failures++
				mutex.Unlock()
				return
			}
			_, _ = io.Copy(io.Discard, response.Body)
			_ = response.Body.Close()
			elapsed := time.Since(start)
			mutex.Lock()
			if response.StatusCode != 200 {
				failures++
			}
			if elapsed > max {
				max = elapsed
			}
			mutex.Unlock()
		}()
	}
	group.Wait()

	if failures > 0 {
		t.Fatalf("%d leituras falharam", failures)
	}
	t.Logf("parallel reads: workers=%d max=%s", workers, max)
	if max > 10*time.Second {
		t.Fatalf("leitura paralela acima do orçamento: %s > 10s", max)
	}
}
