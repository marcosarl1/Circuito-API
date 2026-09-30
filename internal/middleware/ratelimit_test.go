package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRateLimitAllowsUpToLimit(t *testing.T) {
	handler := RateLimit(2, time.Minute)(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusNoContent)
	}))

	for attempt := 0; attempt < 2; attempt++ {
		request := httptest.NewRequest(http.MethodPost, "/api/v1/scrape/run", nil)
		request.RemoteAddr = "10.0.0.1:1234"
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, request)
		if recorder.Code != http.StatusNoContent {
			t.Fatalf("attempt %d: expected 204, got %d", attempt, recorder.Code)
		}
	}

	request := httptest.NewRequest(http.MethodPost, "/api/v1/scrape/run", nil)
	request.RemoteAddr = "10.0.0.1:5678"
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d", recorder.Code)
	}
	if contentType := recorder.Header().Get("Content-Type"); contentType != "application/json" {
		t.Fatalf("expected JSON error, got %q", contentType)
	}
}

func TestRateLimitIsPerClient(t *testing.T) {
	handler := RateLimit(1, time.Minute)(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusNoContent)
	}))

	first := httptest.NewRequest(http.MethodPost, "/api/v1/scrape/run", nil)
	first.RemoteAddr = "10.0.0.1:1111"
	firstRecorder := httptest.NewRecorder()
	handler.ServeHTTP(firstRecorder, first)

	second := httptest.NewRequest(http.MethodPost, "/api/v1/scrape/run", nil)
	second.RemoteAddr = "10.0.0.2:2222"
	secondRecorder := httptest.NewRecorder()
	handler.ServeHTTP(secondRecorder, second)

	if secondRecorder.Code != http.StatusNoContent {
		t.Fatalf("different client: expected 204, got %d", secondRecorder.Code)
	}
}

func TestRateLimitRefillsAfterWindow(t *testing.T) {
	limiter := &RateLimiter{requests: 1, window: time.Minute, hits: map[string][]time.Time{}}
	now := time.Now()
	if !limiter.allow("client", now) {
		t.Fatal("expected first call to pass")
	}
	if limiter.allow("client", now) {
		t.Fatal("expected second call to be limited")
	}
	if !limiter.allow("client", now.Add(2*time.Minute)) {
		t.Fatal("expected window refill to pass")
	}
}
