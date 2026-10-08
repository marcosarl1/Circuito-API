//go:build perf

// go test -tags perf ./tests/performance/ -v -count=1
package performance

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"testing"
	"time"

	"github.com/marcosarl1/Circuito-API/tests/testenv"
)

var (
	baseURL     string
	apiKey      string
	scrapersKey string
	client      = &http.Client{Timeout: 120 * time.Second}
)

func TestMain(m *testing.M) {
	env, cleanup, err := testenv.StartContext(context.Background())
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	baseURL = env.BaseURL
	apiKey = env.APIKey
	scrapersKey = env.ScrapersKey
	code := m.Run()
	cleanup()
	os.Exit(code)
}

// seedEvents cria count eventos via POST, exercitando o caminho real
// de escrita (geração de ID + insert).
func seedEvents(t *testing.T, count int) {
	t.Helper()
	for i := range count {
		body, _ := json.Marshal(map[string]any{
			"nome_evento": fmt.Sprintf("Evento perf %04d", i),
			"cidade":      fmt.Sprintf("Cidade %d", i%50),
			"estado":      "PB",
		})
		request, err := http.NewRequest("POST", baseURL+"/api/v1/eventos", bytes.NewReader(body))
		if err != nil {
			t.Fatalf("new request: %v", err)
		}
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("X-API-Key", apiKey)
		response, err := client.Do(request)
		if err != nil {
			t.Fatalf("seed %d: %v", i, err)
		}
		_, _ = io.Copy(io.Discard, response.Body)
		_ = response.Body.Close()
		if response.StatusCode != 201 {
			t.Fatalf("seed %d: status %d", i, response.StatusCode)
		}
	}
}

// measure executa calls GETs sequenciais e devolve p50/p95/max em ms.
func measure(t *testing.T, path string, calls int) (p50, p95, max float64) {
	t.Helper()
	samples := make([]float64, 0, calls)
	for range calls {
		start := time.Now()
		response, err := client.Get(baseURL + path)
		if err != nil {
			t.Fatalf("GET %s: %v", path, err)
		}
		_, _ = io.Copy(io.Discard, response.Body)
		_ = response.Body.Close()
		if response.StatusCode != 200 {
			t.Fatalf("GET %s: status %d", path, response.StatusCode)
		}
		samples = append(samples, float64(time.Since(start).Milliseconds()))
	}
	sort.Float64s(samples)
	p50 = samples[int(float64(len(samples))*0.50)]
	p95 = samples[min(int(float64(len(samples))*0.95), len(samples)-1)]
	max = samples[len(samples)-1]
	t.Logf("%s: calls=%d p50=%.0fms p95=%.0fms max=%.0fms", path, calls, p50, p95, max)
	return p50, p95, max
}

func TestListLatency(t *testing.T) {
	seedEvents(t, 1000)
	_, p95, _ := measure(t, "/api/v1/eventos?page=1&size=20", 50)
	if p95 > 500 {
		t.Fatalf("list p95 acima do orçamento: %.0fms > 500ms", p95)
	}
}

func TestDashboardLatency(t *testing.T) {
	_, p95, _ := measure(t, "/api/v1/dashboard/stats", 5)
	if p95 > 2000 {
		t.Fatalf("dashboard p95 acima do orçamento: %.0fms > 2000ms", p95)
	}
}
