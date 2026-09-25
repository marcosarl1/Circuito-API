package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
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
