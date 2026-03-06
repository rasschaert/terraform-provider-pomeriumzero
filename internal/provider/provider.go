package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

const (
	// Base URL for version 0 of the Pomerium Zero API
	apiBaseURL = "https://console.pomerium.app/api/v0"
	// Endpoint exchanging the API token for a JWT
	tokenEndpoint = apiBaseURL + "/token"
	// Endpoint for retrieving organization information
	organizationsEndpoint = apiBaseURL + "/organizations"
)

// Ensure the implementation satisfies the expected interfaces.
var _ provider.Provider = &pomeriumZeroProvider{}

// New is a helper function to simplify provider server and testing implementation.
func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &pomeriumZeroProvider{version: version}
	}
}

// pomeriumZeroProvider is the provider implementation.
type pomeriumZeroProvider struct {
	version string
	client  *apiClient
}

// pomeriumZeroProviderModel describes the provider data model.
type pomeriumZeroProviderModel struct {
	APIToken types.String `tfsdk:"api_token"`
}

// Metadata returns the provider type name.
func (p *pomeriumZeroProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "pomeriumzero"
	resp.Version = p.version
}

// Schema defines the provider-level schema for configuration data.
func (p *pomeriumZeroProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"api_token": schema.StringAttribute{
				Required:    true,
				Sensitive:   true,
				Description: "The API token for authenticating with Pomerium Zero",
			},
		},
	}
}

// Configure prepares a Pomerium Zero API client for data sources and resources.
func (p *pomeriumZeroProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config pomeriumZeroProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if config.APIToken.IsNull() {
		resp.Diagnostics.AddError(
			"Missing API Token Configuration",
			"The API token is required to authenticate with Pomerium Zero.",
		)
		return
	}

	httpClient := &http.Client{Timeout: 10 * time.Second}

	tflog.Debug(ctx, "Exchanging API token for JWT")
	token, err := getToken(ctx, httpClient, config.APIToken.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Authenticate to Pomerium Zero",
			"An unexpected error occurred when authenticating to Pomerium Zero. "+
				"Please check your API token and try again.\n\nError: "+err.Error(),
		)
		return
	}

	tflog.Debug(ctx, "Fetching organization ID")
	orgID, err := getOrganizationID(ctx, httpClient, token)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Fetch Organization ID",
			"An unexpected error occurred when fetching the organization ID. "+
				"Please check your API token and try again.\n\nError: "+err.Error(),
		)
		return
	}
	tflog.Debug(ctx, "Provider configured", map[string]interface{}{"organization_id": orgID})

	p.client = &apiClient{
		http:           httpClient,
		token:          token,
		organizationID: orgID,
	}

	resp.DataSourceData = p.client
	resp.ResourceData = p.client
}

// getToken exchanges an API token for a JWT bearer token.
func getToken(ctx context.Context, client *http.Client, apiToken string) (string, error) {
	payload, err := json.Marshal(map[string]string{"refreshToken": apiToken})
	if err != nil {
		return "", fmt.Errorf("error encoding token request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenEndpoint, strings.NewReader(string(payload)))
	if err != nil {
		return "", fmt.Errorf("error creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("error making request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var result struct {
		IDToken string `json:"idToken"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("error decoding response: %w", err)
	}
	return result.IDToken, nil
}

// getOrganizationID looks up the organization ID for the authenticated user.
func getOrganizationID(ctx context.Context, client *http.Client, token string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, organizationsEndpoint, nil)
	if err != nil {
		return "", fmt.Errorf("error creating request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("error making request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var organizations []struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&organizations); err != nil {
		return "", fmt.Errorf("error decoding response: %w", err)
	}

	if len(organizations) == 0 {
		return "", fmt.Errorf("no organizations found for this API token")
	}
	if len(organizations) > 1 {
		return "", fmt.Errorf("multiple organizations found; this provider supports single-organization tokens only")
	}
	return organizations[0].ID, nil
}

// DataSources defines the data sources implemented in the provider.
func (p *pomeriumZeroProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewClusterDataSource,
		NewPolicyDataSource,
	}
}

// Resources defines the resources implemented in the provider.
func (p *pomeriumZeroProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewClusterResource,
		NewClusterSettingsResource,
		NewPolicyResource,
		NewRouteResource,
		NewServiceAccountResource,
	}
}
