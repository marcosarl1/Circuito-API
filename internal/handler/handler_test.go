package handler

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/marcosarl1/Circuito-API/internal/service"
)

func TestHealthReturnsOK(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/health", nil)

	recorder := httptest.NewRecorder()

	Health(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}

	if contentType := recorder.Header().Get("Content-Type"); contentType != "application/json" {
		t.Fatalf("expected application/json, got %q", contentType)
	}

	var response map[string]string
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("could not decode response: %v", err)
	}

	if response["status"] != "ok" {
		t.Fatalf("expected status=ok, got %q", response["status"])
	}
}
func TestReadyReturnsServiceUnavailableWithoutStore(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/ready", nil)

	recorder := httptest.NewRecorder()

	Ready(nil)(recorder, request)

	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status %d, got %d", http.StatusServiceUnavailable, recorder.Code)
	}

	var response map[string]string
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("could not decode response: %v", err)
	}

	if response["status"] != "not_ready" {
		t.Fatalf("expected status=not_ready, got %q", response["status"])
	}
}

func TestRequireAPIKeyRejectsMissingKey(t *testing.T) {
	protectedHandler := RequireAPIKey("expected-key")

	request := httptest.NewRequest(http.MethodPost, "/api/v1/eventos", nil)

	recorder := httptest.NewRecorder()

	protectedHandler(func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(http.StatusNoContent)
	})(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, recorder.Code)
	}

	var response map[string]string
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("could not decode response: %v", err)
	}

	if response["detail"] != "Chave API inválida" {
		t.Fatalf("unexpected detail: %q", response["detail"])
	}
}

func TestRequireAPIKeyAllowsValidKey(t *testing.T) {
	protectedHandler := RequireAPIKey("expected-key")

	request := httptest.NewRequest(http.MethodPost, "/api/v1/eventos", nil)
	request.Header.Set("X-API-Key", "expected-key")

	recorder := httptest.NewRecorder()

	protectedHandler(func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(http.StatusNoContent)
	})(recorder, request)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, recorder.Code)
	}
}

func TestGetEventReturnsServiceUnavailableWithoutStore(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/api/v1/eventos/2026090001", nil)

	recorder := httptest.NewRecorder()

	GetEvent(nil)(recorder, request)

	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status %d, got %d", http.StatusServiceUnavailable, recorder.Code)
	}

	var response map[string]string
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("could not decode response: %v", err)
	}

	if response["detail"] != "Banco de dados indisponível" {
		t.Fatalf("unexpected detail: %q", response["detail"])
	}
}

func newTestRequest(
	method string,
	path string,
	body string,
) *http.Request {
	request := httptest.NewRequest(method, path, nil)

	if body != "" {
		request.Body = io.NopCloser(strings.NewReader(body))
	}
	return request
}

func TestGetEventReturnsEvent(t *testing.T) {
	store := newFakeEventStore()
	store.events["2026090001"] = service.Event{
		ID:         "2026090001",
		NomeEvento: "Corrida Teste",
		Cidade:     "João Pessoa",
		Estado:     "PB",
	}

	request := newTestRequest(http.MethodGet, "/api/v1/eventos/2026090001", "")
	request.SetPathValue("id", "2026090001")

	recorder := httptest.NewRecorder()

	GetEvent(store)(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}

	var evento service.Event
	if err := json.NewDecoder(recorder.Body).Decode(&evento); err != nil {
		t.Fatalf("could not decode response: %v", err)
	}

	if evento.ID != "2026090001" {
		t.Fatalf("expected id 2026090001, got %q", evento.ID)
	}
}

func TestGetEventReturnsNotFound(t *testing.T) {
	store := newFakeEventStore()

	request := newTestRequest(http.MethodGet, "/api/v1/eventos/2026090001", "")
	request.SetPathValue("id", "2026090001")

	recorder := httptest.NewRecorder()

	GetEvent(store)(recorder, request)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expecteed status %d, got %d", http.StatusNotFound, recorder.Code)
	}
}

func TestCreateEventReturnsCreated(t *testing.T) {
	store := newFakeEventStore()

	nextID := func(requestContext context.Context) (string, error) {
		return "2026090001", nil
	}

	request := newTestRequest(http.MethodPost, "/api/v1/eventos",
		`{
			"nome_evento": "Corrida teste",
			"cidade": "Joao Pessoa",
			"estado": "PB"
			}`,
	)

	recorder := httptest.NewRecorder()

	CreateEvent(store, nextID)(recorder, request)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, recorder.Code)
	}

	var evento service.Event
	if err := json.NewDecoder(recorder.Body).Decode(&evento); err != nil {
		t.Fatalf("could not decode response, got %v", err)
	}

	if evento.ID != "2026090001" {
		t.Fatalf("expected id 2026090001, got %q", evento.ID)
	}
}

func TestUpdateEventReturnsUpdatedEvent(t *testing.T) {
	store := newFakeEventStore()
	store.events["2026090001"] = service.Event{
		ID:         "2026090001",
		NomeEvento: "Nome antiga",
		Cidade:     "Cidade antiga",
	}

	request := newTestRequest(http.MethodPatch, "/api/v1/eventos/2026090001",
		`{
			"nome_evento": "Nome novo",
			"cidade": "Cidade nova"
		}`,
	)
	request.SetPathValue("id", "2026090001")

	recorder := httptest.NewRecorder()

	UpdateEvent(store)(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}

	var evento service.Event
	if err := json.NewDecoder(recorder.Body).Decode(&evento); err != nil {
		t.Fatalf("could not decode response: %v", err)
	}

	if evento.NomeEvento != "Nome novo" {
		t.Fatalf("expected nome_evento Nome Novo, got %q", evento.NomeEvento)
	}

	if evento.Cidade != "Cidade nova" {
		t.Fatalf("expected cidade Cidade nova, got %q", evento.Cidade)
	}
}

func TestDeleteEventReturnsNoContent(t *testing.T) {
	store := newFakeEventStore()
	store.events["2026090001"] = service.Event{
		ID: "2026090001",
	}

	request := newTestRequest(http.MethodDelete, "/api/v1/eventos/2026090001", "")
	request.SetPathValue("id", "2026090001")

	recorder := httptest.NewRecorder()

	DeleteEvent(store)(recorder, request)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, recorder.Code)
	}

	if _, exists := store.events["2026090001"]; exists {
		t.Fatal("expected evento to be deleted")
	}
}
