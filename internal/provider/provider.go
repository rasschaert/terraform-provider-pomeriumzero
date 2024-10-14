package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const (
	// Base URL for version 0 of the Pomerium Zero API
	apiBaseURL = "https://console.pomerium.app/api/v0"
	// Endpoint exhanging the API token for a JWT
	tokenEndpoint = apiBaseURL + "/token"
	// Endpoint for retrieving organization information
	organizationsEndpoint = apiBaseURL + "/organizations"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ provider.Provider = &pomeriumZeroProvider{}
)

// New is a helper function to simplify provider server and testing implementation.
func New(version string) func() provider.Provider {
	return func() provider.Provider {
		log.Println("Creating new Pomerium Zero provider instance")
		return &pomeriumZeroProvider{
			version: version,
		}
	}
}

// pomeriumZeroProvider is the provider implementation.
type pomeriumZeroProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version        string
	client         *http.Client
	token          string
	organizationID string
}

// pomeriumZeroProviderModel describes the provider data model.
type pomeriumZeroProviderModel struct {
	APIToken types.String `tfsdk:"api_token"`
}

// Metadata returns the provider type name.
func (p *pomeriumZeroProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	log.Println("Metadata function called")
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
	log.Println("Starting provider configuration")

	var config pomeriumZeroProviderModel

	// Set the provider instance as the ProviderData
	resp.DataSourceData = p
	resp.ResourceData = p

	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)

	if resp.Diagnostics.HasError() {
		log.Println("Error in configuration:", resp.Diagnostics.Errors())
		return
	}

	// log.Printf("Configuration: API Token: %s, Organization Name: %s", config.APIToken.ValueString(), config.OrganizationName.ValueString())

	if config.APIToken.IsNull() {
		log.Println("API Token is null")
		resp.Diagnostics.AddError(
			"Missing API Token Configuration",
			"The API token is required to authenticate with Pomerium Zero.",
		)
		return
	}

	p.client = &http.Client{
		Timeout: time.Second * 10,
	}

	log.Println("Getting token")
	token, err := p.getToken(ctx, config.APIToken.ValueString())
	if err != nil {
		log.Println("Error getting token:", err)
		resp.Diagnostics.AddError(
			"Unable to Authenticate to Pomerium Zero",
			"An unexpected error occurred when authenticating to Pomerium Zero. "+
				"Please check your API token and try again.\n\n"+
				"Error: "+err.Error(),
		)
		return
	}

	p.token = token
	log.Println("Token obtained successfully")

	log.Println("Getting organization ID")
	orgID, err := p.getOrganizationID(ctx)
	if err != nil {
		log.Println("Error getting organization ID:", err)

		resp.Diagnostics.AddError(
			"Unable to Fetch Organization ID",
			"An unexpected error occurred when fetching the organization ID. "+
				"Please check your API token and try again.\n\n"+
				"Error: "+err.Error(),
		)
		return
	}

	p.organizationID = orgID
	log.Printf("Organization ID obtained successfully: %s", orgID)
}

// Exchange the API token for a JWT bearer token.
func (p *pomeriumZeroProvider) getToken(ctx context.Context, apiToken string) (string, error) {
	payload := strings.NewReader(fmt.Sprintf(`{"refreshToken": "%s"}`, apiToken))
	log.Println("Sending request to token endpoint")
	req, err := http.NewRequestWithContext(ctx, "POST", tokenEndpoint, payload)
	if err != nil {
		return "", err
	}

	req.Header.Add("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		log.Println("Error making request:", err)
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("Unexpected status code: %d", resp.StatusCode)
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var result struct {
		IDToken string `json:"idToken"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		log.Println("Error decoding response:", err)
		return "", err
	}

	return result.IDToken, nil
}

// Lookup the organization ID by name.
func (p *pomeriumZeroProvider) getOrganizationID(ctx context.Context) (string, error) {
	log.Println("Fetching organization ID")

	req, err := http.NewRequestWithContext(ctx, "GET", organizationsEndpoint, nil)
	if err != nil {
		log.Println("Error creating request:", err)
		return "", err
	}

	req.Header.Add("Authorization", "Bearer "+p.token)
	req.Header.Add("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		log.Println("Error making request:", err)
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("Unexpected status code: %d", resp.StatusCode)
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var organizations []struct {
		ID string `json:"id"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&organizations); err != nil {
		log.Println("Error decoding response:", err)
		return "", err
	}

	if len(organizations) != 1 {
		log.Println("Unexpected number of organizations returned")
		return "", fmt.Errorf("unexpected number of organizations returned")
	}

	return organizations[0].ID, nil
}

// DataSources defines the data sources implemented in the provider.
func (p *pomeriumZeroProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewClusterDataSource,
	}
}

// Resources defines the resources implemented in the provider.
func (p *pomeriumZeroProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewPolicyResource,
		NewRouteResource,
		NewClusterSettingsResource,
	}
}
