package azure

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func testStarter(serverURL string) *JobStarter {
	return &JobStarter{
		SubscriptionID:   "sub",
		ResourceGroup:    "rg",
		JobName:          "job",
		APIVersion:       "2023-05-01",
		ManagementURL:    serverURL + "/mgmt",
		IdentityEndpoint: serverURL + "/msi",
		IdentityHeader:   "secret",
		HTTPClient:       &http.Client{Timeout: 10 * time.Second},
	}
}

func TestStartTriggersExecution(t *testing.T) {
	var msiHit, armHit bool
	var authHeader, apiVersion string

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch {
		case strings.HasPrefix(request.URL.Path, "/msi"):
			msiHit = true
			if request.Header.Get("X-IDENTITY-HEADER") != "secret" {
				t.Errorf("missing identity header")
			}
			if !strings.Contains(request.URL.RawQuery, "management.azure.com") {
				t.Errorf("wrong resource: %s", request.URL.RawQuery)
			}
			_, _ = writer.Write([]byte(`{"access_token":"token-123"}`))
		case strings.HasSuffix(request.URL.Path, "/start"):
			armHit = true
			authHeader = request.Header.Get("Authorization")
			apiVersion = request.URL.Query().Get("api-version")
			writer.WriteHeader(http.StatusCreated)
		default:
			t.Errorf("unexpected path: %s", request.URL.Path)
			writer.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	starter := testStarter(server.URL)
	if err := starter.Start(t.Context()); err != nil {
		t.Fatalf("start: %v", err)
	}
	if !msiHit || !armHit {
		t.Fatal("expected both MSI and ARM calls")
	}
	if authHeader != "Bearer token-123" {
		t.Fatalf("auth header: %q", authHeader)
	}
	if apiVersion != "2023-05-01" {
		t.Fatalf("api-version: %q", apiVersion)
	}
}

func TestStartFailsOnARMRejection(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if strings.HasPrefix(request.URL.Path, "/msi") {
			_, _ = writer.Write([]byte(`{"access_token":"token-123"}`))
			return
		}
		writer.WriteHeader(http.StatusForbidden)
	}))
	defer server.Close()

	starter := testStarter(server.URL)
	if err := starter.Start(t.Context()); err == nil {
		t.Fatal("expected error on 403")
	}
}

func TestStartFailsWithoutIdentity(t *testing.T) {
	starter := &JobStarter{
		SubscriptionID: "sub",
		ResourceGroup:  "rg",
		JobName:        "job",
		APIVersion:     "2023-05-01",
		ManagementURL:  "http://localhost",
		HTTPClient:     &http.Client{Timeout: 10 * time.Second},
	}
	if err := starter.Start(t.Context()); err == nil {
		t.Fatal("expected error without identity endpoint")
	}
}
