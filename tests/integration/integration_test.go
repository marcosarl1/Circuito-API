//go:build integration

// go test -tags integration ./tests/integration/ -v -count=1
package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"testing"
	"time"

	"github.com/marcosarl1/Circuito-API/tests/testenv"
)

var (
	baseURL     string
	apiKey      string
	scrapersKey string
	httpClient  = &http.Client{Timeout: 90 * time.Second}
	eventIDRe   = regexp.MustCompile(`^\d{10}$`)
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

// do faz uma chamada HTTP e devolve status + corpo decodificado.
func do(t *testing.T, method, path string, body any, key string) (int, map[string]any) {
	t.Helper()

	var reader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		reader = bytes.NewReader(raw)
	}

	request, err := http.NewRequest(method, baseURL+path, reader)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	if key != "" {
		request.Header.Set("X-API-Key", key)
	}

	response, err := httpClient.Do(request)
	if err != nil {
		t.Fatalf("%s %s: %v", method, path, err)
	}
	defer response.Body.Close()

	raw, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	if len(raw) == 0 {
		// Respostas sem corpo (ex.: 204 No Content) não têm Content-Type.
		return response.StatusCode, map[string]any{}
	}
	if contentType := response.Header.Get("Content-Type"); contentType != "application/json" {
		t.Fatalf("%s %s: Content-Type=%q corpo=%.120s", method, path, contentType, raw)
	}

	decoded := map[string]any{}
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &decoded); err != nil {
			t.Fatalf("%s %s: corpo não é JSON: %v (%.120s)", method, path, err, raw)
		}
	}
	return response.StatusCode, decoded
}

func status(t *testing.T, method, path string, body any, key string, want int) map[string]any {
	t.Helper()
	got, decoded := do(t, method, path, body, key)
	if got != want {
		t.Fatalf("%s %s: esperado %d, obtido %d (%v)", method, path, want, got, decoded)
	}
	return decoded
}

// createEvent cria um evento de teste e agenda a remoção ao final.
func createEvent(t *testing.T, nome, cidade string) string {
	t.Helper()
	decoded := status(t, "POST", "/api/v1/eventos",
		map[string]any{"nome_evento": nome, "cidade": cidade, "estado": "PB"}, apiKey, 201)
	id, _ := decoded["id"].(string)
	if !eventIDRe.MatchString(id) {
		t.Fatalf("id fora do formato YYYYMM####: %q", id)
	}
	t.Cleanup(func() {
		request, _ := http.NewRequest("DELETE", baseURL+"/api/v1/eventos/"+id, nil)
		request.Header.Set("X-API-Key", apiKey)
		if response, err := httpClient.Do(request); err == nil {
			response.Body.Close()
		}
	})
	return id
}
