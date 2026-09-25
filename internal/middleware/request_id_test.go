package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequestIDPreservesIncomingValue(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/health", nil)

	request.Header.Set("X-Request-Id", "request-123")

	recorder := httptest.NewRecorder()

	handlerCalled := false

	handler := RequestID(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		handlerCalled = true
		writer.WriteHeader(http.StatusNoContent)
	}))
	handler.ServeHTTP(recorder, request)

	if !handlerCalled {
		t.Fatal("expected next handler to be called")
	}

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, recorder.Code)
	}

	if responseID := recorder.Header().Get("X-Request-Id"); responseID != "request-123" {
		t.Fatalf("expected X-Request-Id request-123, got %q", responseID)
	}
}

func TestRequestIDGeneratesValueWhenMissing(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/health", nil)

	recorder := httptest.NewRecorder()

	handler := RequestID(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(http.StatusOK)
	}))

	handler.ServeHTTP(recorder, request)

	responseID := recorder.Header().Get("X-Request-Id")

	if responseID == "" {
		t.Fatal("expected X-Request-Id to be generated")
	}

	if len(responseID) != 32 {
		t.Fatalf("expected generated request id length 32, got %d", len(responseID))
	}
}
