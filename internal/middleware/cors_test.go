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

func TestCORSRejectsUnconfiguredOrigin(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/health", nil)
	request.Header.Set("Origin", "https://randomsite.com")
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

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}

	if allowedOrigin := recorder.Header().Get("Access-Control-Allow-Origin"); allowedOrigin != "" {
		t.Fatalf("expected no Access-Control-Allow-Origin, got %q", allowedOrigin)
	}
}

func TestCORSPreflightWithoutAllowedOriginDoesNotExposeOrigin(t *testing.T) {
	request := httptest.NewRequest(http.MethodOptions, "/api/v1/eventos", nil)
	request.Header.Set("Origin", "http://randomsite.com")
	request.Header.Set("Access-Control-Request-Method", "PATCH")

	recorder := httptest.NewRecorder()

	handler := CORS("https://app.example.com")(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(http.StatusOK)
	}))

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, recorder.Code)
	}

	if allowedOrigin := recorder.Header().Get("Access-Control-Allow-Origin"); allowedOrigin != "" {
		t.Fatalf("expected no Access-Control-Allow-Origin, got %q", allowedOrigin)
	}
}

func TestCORSWildcardAllowsAnyOrigin(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/health", nil)
	request.Header.Set("Origin", "https://any.example.com")

	recorder := httptest.NewRecorder()

	handler := CORS("*")(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(http.StatusOK)
	}))

	handler.ServeHTTP(recorder, request)

	if allowedOrigin := recorder.Header().Get("Access-Control-Allow-Origin"); allowedOrigin != "https://any.example.com" {
		t.Fatalf("expected wildcard to allow origin, got %q", allowedOrigin)
	}
}

func TestCORSAllowsMultipleConfiguredOrigins(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/health", nil)
	request.Header.Set("Origin", "https://admin.example.com")

	recorder := httptest.NewRecorder()

	handler := CORS("https://app.example.com,https://admin.example.com")(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(http.StatusOK)
	}))

	handler.ServeHTTP(recorder, request)

	if allowedOrigin := recorder.Header().Get("Access-Control-Allow-Origin"); allowedOrigin != "https://admin.example.com" {
		t.Fatalf("expected admin origin to be allowed, got %q", allowedOrigin)
	}
}
