package azure

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// JobStarter triggers a Manual Container Apps Job execution through the
// Azure Resource Manager API, authenticating with the app's managed
// identity. No secrets are stored: the token comes from the platform
// identity endpoint available inside Container Apps.
type JobStarter struct {
	SubscriptionID string
	ResourceGroup  string
	JobName        string
	APIVersion     string
	// ManagementURL and identity endpoint/header are fields (not globals)
	// so tests can point them at fake servers.
	ManagementURL    string
	IdentityEndpoint string
	IdentityHeader   string
	HTTPClient       *http.Client
	Timeout          time.Duration
}

// Start triggers one job execution. A 2xx from ARM means accepted.
func (starter *JobStarter) Start(requestContext context.Context) error {
	requestContext, cancel := context.WithTimeout(requestContext, starter.timeout())
	defer cancel()

	token, err := starter.accessToken(requestContext)
	if err != nil {
		return fmt.Errorf("managed identity token: %w", err)
	}

	startURL := fmt.Sprintf(
		"%s/subscriptions/%s/resourceGroups/%s/providers/Microsoft.App/jobs/%s/start?api-version=%s",
		strings.TrimSuffix(starter.ManagementURL, "/"),
		url.PathEscape(starter.SubscriptionID),
		url.PathEscape(starter.ResourceGroup),
		url.PathEscape(starter.JobName),
		url.PathEscape(starter.APIVersion),
	)
	request, err := http.NewRequestWithContext(requestContext, http.MethodPost, startURL, nil)
	if err != nil {
		return err
	}
	request.Header.Set("Authorization", "Bearer "+token)

	response, err := starter.HTTPClient.Do(request)
	if err != nil {
		return fmt.Errorf("start job execution: %w", err)
	}
	defer response.Body.Close()
	_, _ = io.Copy(io.Discard, response.Body)

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("start job execution: status %d", response.StatusCode)
	}
	return nil
}

func (starter *JobStarter) timeout() time.Duration {
	if starter.Timeout > 0 {
		return starter.Timeout
	}
	return 10 * time.Second
}

// accessToken fetches an ARM token from the platform identity endpoint
// (App Service / Container Apps MSI protocol).
func (starter *JobStarter) accessToken(requestContext context.Context) (string, error) {
	if starter.IdentityEndpoint == "" || starter.IdentityHeader == "" {
		return "", fmt.Errorf("identity endpoint not configured (not running on Azure?)")
	}

	tokenURL := starter.IdentityEndpoint + "?resource=https://management.azure.com/&api-version=2019-08-01"
	request, err := http.NewRequestWithContext(requestContext, http.MethodGet, tokenURL, nil)
	if err != nil {
		return "", err
	}
	request.Header.Set("X-IDENTITY-HEADER", starter.IdentityHeader)

	response, err := starter.HTTPClient.Do(request)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()

	var payload struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return "", fmt.Errorf("decode token: %w", err)
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 || payload.AccessToken == "" {
		return "", fmt.Errorf("token request: status %d", response.StatusCode)
	}
	return payload.AccessToken, nil
}
