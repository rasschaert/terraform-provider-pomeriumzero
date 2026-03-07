package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// errNotFound is returned by apiClient methods when the API responds with 404.
var errNotFound = fmt.Errorf("not found")

// apiClient handles all HTTP communication with the Pomerium Zero API.
type apiClient struct {
	http           *http.Client
	token          string
	organizationID string
}

// do executes an HTTP request, setting the Authorization header and returning
// the response body. The caller is responsible for interpreting the status code.
func (c *apiClient) do(ctx context.Context, method, url string, body io.Reader) (*http.Response, error) {
	var bodyBytes []byte
	if body != nil {
		var err error
		bodyBytes, err = io.ReadAll(body)
		if err != nil {
			return nil, fmt.Errorf("error reading request body: %w", err)
		}
		tflog.Trace(ctx, "API request", map[string]interface{}{
			"method": method,
			"url":    url,
			"body":   string(bodyBytes),
		})
		body = bytes.NewReader(bodyBytes)
	} else {
		tflog.Trace(ctx, "API request", map[string]interface{}{
			"method": method,
			"url":    url,
		})
	}

	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	if bodyBytes != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making request: %w", err)
	}
	return resp, nil
}

// get performs a GET request and JSON-decodes the response body into out.
// Returns errNotFound if the server responds with 404.
func (c *apiClient) get(ctx context.Context, url string, out interface{}) error {
	resp, err := c.do(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("error reading response: %w", err)
	}
	tflog.Trace(ctx, "API response", map[string]interface{}{
		"status": resp.StatusCode,
		"body":   string(body),
	})

	if resp.StatusCode == http.StatusNotFound {
		return errNotFound
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status %d: %s", resp.StatusCode, body)
	}
	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("error decoding response: %w", err)
	}
	return nil
}

// post performs a POST request with a JSON body and decodes the response into out.
// Expects the server to respond with expectedStatus (typically 201 Created).
func (c *apiClient) post(ctx context.Context, url string, in interface{}, expectedStatus int, out interface{}) error {
	encoded, err := json.Marshal(in)
	if err != nil {
		return fmt.Errorf("error encoding request: %w", err)
	}
	resp, err := c.do(ctx, http.MethodPost, url, bytes.NewReader(encoded))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("error reading response: %w", err)
	}
	tflog.Trace(ctx, "API response", map[string]interface{}{
		"status": resp.StatusCode,
		"body":   string(body),
	})
	if resp.StatusCode != expectedStatus {
		return fmt.Errorf("unexpected status %d: %s", resp.StatusCode, body)
	}
	if out != nil {
		if err := json.Unmarshal(body, out); err != nil {
			return fmt.Errorf("error decoding response: %w", err)
		}
	}
	return nil
}

// put performs a PUT request with a JSON body and decodes the 200 response into out.
func (c *apiClient) put(ctx context.Context, url string, in interface{}, out interface{}) error {
	encoded, err := json.Marshal(in)
	if err != nil {
		return fmt.Errorf("error encoding request: %w", err)
	}
	resp, err := c.do(ctx, http.MethodPut, url, bytes.NewReader(encoded))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("error reading response: %w", err)
	}
	tflog.Trace(ctx, "API response", map[string]interface{}{
		"status": resp.StatusCode,
		"body":   string(body),
	})
	if resp.StatusCode == http.StatusNotFound {
		return errNotFound
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status %d: %s", resp.StatusCode, body)
	}
	if out != nil {
		if err := json.Unmarshal(body, out); err != nil {
			return fmt.Errorf("error decoding response: %w", err)
		}
	}
	return nil
}

// delete performs a DELETE request and expects 204 No Content.
func (c *apiClient) delete(ctx context.Context, url string) error {
	resp, err := c.do(ctx, http.MethodDelete, url, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	tflog.Trace(ctx, "API response", map[string]interface{}{
		"status": resp.StatusCode,
		"body":   string(body),
	})
	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("unexpected status %d: %s", resp.StatusCode, body)
	}
	return nil
}

// url helpers

func (c *apiClient) clustersURL() string {
	return fmt.Sprintf("%s/organizations/%s/clusters", apiBaseURL, c.organizationID)
}

func (c *apiClient) clusterURL(id string) string {
	return fmt.Sprintf("%s/organizations/%s/clusters/%s", apiBaseURL, c.organizationID, id)
}

func (c *apiClient) clusterSettingsURL(clusterID string) string {
	return fmt.Sprintf("%s/organizations/%s/clusters/%s/settings", apiBaseURL, c.organizationID, clusterID)
}

func (c *apiClient) policiesURL() string {
	return fmt.Sprintf("%s/organizations/%s/policies", apiBaseURL, c.organizationID)
}

func (c *apiClient) policyURL(id string) string {
	return fmt.Sprintf("%s/organizations/%s/policies/%s", apiBaseURL, c.organizationID, id)
}

func (c *apiClient) routesURL() string {
	return fmt.Sprintf("%s/organizations/%s/routes", apiBaseURL, c.organizationID)
}

func (c *apiClient) routeURL(id string) string {
	return fmt.Sprintf("%s/organizations/%s/routes/%s", apiBaseURL, c.organizationID, id)
}

func (c *apiClient) serviceAccountsURL(clusterID string) string {
	return fmt.Sprintf("%s/organizations/%s/clusters/%s/serviceAccounts", apiBaseURL, c.organizationID, clusterID)
}

func (c *apiClient) serviceAccountURL(clusterID, serviceAccountID string) string {
	return fmt.Sprintf("%s/organizations/%s/clusters/%s/serviceAccounts/%s", apiBaseURL, c.organizationID, clusterID, serviceAccountID)
}

func (c *apiClient) serviceAccountTokenURL(clusterID, serviceAccountID string) string {
	return fmt.Sprintf("%s/organizations/%s/clusters/%s/serviceAccounts/%s/token", apiBaseURL, c.organizationID, clusterID, serviceAccountID)
}
