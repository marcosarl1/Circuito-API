package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCORSAllowsConfiguredOrigin(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/health", nil)

	request.Header.Set("Origin", "https://app.example.com")

	recorder := httptest.NewRecorder()

	handlerCalled := false

	handler := CORS("https://app.example.com")(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		handlerCalled = true
		writer.WriteHeader(http.StatusOK)
	}))

	handler.ServeHTTP(recorder, request)

	if !handlerCalled {
		t.Fatal("expected handler to be called")
	}

	allowedOrigin := recorder.Header().Get("Access-Control-Allow-Origin")

	if allowedOrigin != "https://app.example.com" {
		t.Fatalf("expected origin https://app.example.com, got %q", allowedOrigin)
	}
}

func TestCORSHandlesPreflightRequest(t *testing.T) {
	request := httptest.NewRequest(http.MethodOptions, "/api/v1/eventos", nil)

	request.Header.Set("Origin", "https://app.example.com")
	request.Header.Set("Access-Control-Request-Method", "PATCH")

	recorder := httptest.NewRecorder()

	handlerCalled := false

	handler := CORS("https://app.example.com")(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		handlerCalled = true
		writer.WriteHeader(http.StatusOK)
	}))

	handler.ServeHTTP(recorder, request)

	if handlerCalled {
		t.Fatal("expected preflight to stop before handler")
	}

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, recorder.Code)
	}
}
