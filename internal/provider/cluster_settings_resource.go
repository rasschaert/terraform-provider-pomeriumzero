package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	resource_schema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	resource_schema_planmodifier "github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	resource_schema_stringplanmodifier "github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &ClusterSettingsResource{}
var _ resource.ResourceWithImportState = &ClusterSettingsResource{}

func NewClusterSettingsResource() resource.Resource {
	return &ClusterSettingsResource{}
}

// ClusterSettingsResource defines the resource implementation.
type ClusterSettingsResource struct {
	client         *http.Client
	token          string
	organizationID string
}

// ClusterSettingsResourceModel describes the resource data model.
type ClusterSettingsResourceModel struct {
	ID                        types.String `tfsdk:"id"`
	Address                   types.String `tfsdk:"address"`
	AuthenticateServiceUrl    types.String `tfsdk:"authenticate_service_url"`
	AutoApplyChangesets       types.Bool   `tfsdk:"auto_apply_changesets"`
	CookieExpire              types.String `tfsdk:"cookie_expire"`
	CookieHttpOnly            types.Bool   `tfsdk:"cookie_http_only"`
	CookieName                types.String `tfsdk:"cookie_name"`
	DefaultUpstreamTimeout    types.String `tfsdk:"default_upstream_timeout"`
	DNSLookupFamily           types.String `tfsdk:"dns_lookup_family"`
	IdentityProvider          types.String `tfsdk:"identity_provider"`
	IdentityProviderClientId  types.String `tfsdk:"identity_provider_client_id"`
	IdentityProviderClientSecret types.String `tfsdk:"identity_provider_client_secret"`
	IdentityProviderUrl       types.String `tfsdk:"identity_provider_url"`
	LogLevel                  types.String `tfsdk:"log_level"`
	PassIdentityHeaders       types.Bool   `tfsdk:"pass_identity_headers"`
	ProxyLogLevel             types.String `tfsdk:"proxy_log_level"`
	SkipXffAppend             types.Bool   `tfsdk:"skip_xff_append"`
	TimeoutIdle               types.String `tfsdk:"timeout_idle"`
	TimeoutRead               types.String `tfsdk:"timeout_read"`
	TimeoutWrite              types.String `tfsdk:"timeout_write"`
	TracingSampleRate         types.Float64 `tfsdk:"tracing_sample_rate"`
}

// Metadata sets the resource type name for the ClusterSettingsResource.
// It appends "_cluster_settings" to the resource type name.
func (r *ClusterSettingsResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cluster_settings"
}

// Schema defines the structure and attributes of the ClusterSettingsResource.
// It specifies the fields that can be used in the Terraform configuration
// to interact with the Pomerium Zero Cluster Settings resource.
func (r *ClusterSettingsResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = resource_schema.Schema{
		MarkdownDescription: "Manages Pomerium Zero Cluster Settings.",
		Attributes: map[string]resource_schema.Attribute{
			// ID is a computed attribute that uniquely identifies the cluster settings
			"id": resource_schema.StringAttribute{
				Computed: true,
				PlanModifiers: []resource_schema_planmodifier.String{
					resource_schema_stringplanmodifier.UseStateForUnknown(),
				},
				MarkdownDescription: "The unique identifier of the cluster settings.",
			},
			// Address specifies the location of the Pomerium Zero cluster
			"address": resource_schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "The address of the Pomerium Zero cluster.",
			},
			// AuthenticateServiceUrl is the endpoint for the authentication service
			"authenticate_service_url": resource_schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "The URL of the authentication service.",
			},
			// AutoApplyChangesets determines if changes should be applied automatically
			"auto_apply_changesets": resource_schema.BoolAttribute{
				Optional:            true,
				MarkdownDescription: "Whether to automatically apply changesets.",
			},
			// CookieExpire sets the lifetime of authentication cookies
			"cookie_expire": resource_schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "The expiration time for cookies.",
			},
			// CookieHttpOnly restricts cookie access to HTTP(S) requests only
			"cookie_http_only": resource_schema.BoolAttribute{
				Optional:            true,
				MarkdownDescription: "Whether cookies should be HTTP only.",
			},
			// CookieName sets the name of the authentication cookie
			"cookie_name": resource_schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "The name of the cookie used for authentication.",
			},
			// DefaultUpstreamTimeout sets the default timeout for upstream requests
			"default_upstream_timeout": resource_schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "The default timeout for upstream requests.",
			},
			// DNSLookupFamily specifies the IP address family for DNS lookups
			"dns_lookup_family": resource_schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "The DNS lookup family to use (e.g., 'v4', 'v6').",
			},
			// IdentityProvider specifies the authentication provider
			"identity_provider": resource_schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "The identity provider to use for authentication.",
			},
			// IdentityProviderClientId is the client ID for the identity provider
			"identity_provider_client_id": resource_schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "The client ID for the identity provider.",
			},
			// IdentityProviderClientSecret is the client secret for the identity provider
			"identity_provider_client_secret": resource_schema.StringAttribute{
				Optional:            true,
				Sensitive:           true,
				MarkdownDescription: "The client secret for the identity provider.",
			},
			// IdentityProviderUrl is the URL of the identity provider
			"identity_provider_url": resource_schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "The URL of the identity provider.",
			},
			// LogLevel sets the logging verbosity for the Pomerium Zero cluster
			"log_level": resource_schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "The log level for the Pomerium Zero cluster.",
			},
			// PassIdentityHeaders determines if identity information should be passed to upstream services
			"pass_identity_headers": resource_schema.BoolAttribute{
				Optional:            true,
				MarkdownDescription: "Whether to pass identity headers to upstream services.",
			},
			// ProxyLogLevel sets the logging verbosity for the proxy component
			"proxy_log_level": resource_schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "The log level for the proxy component.",
			},
			// SkipXffAppend determines if X-Forwarded-For headers should be appended
			"skip_xff_append": resource_schema.BoolAttribute{
				Optional:            true,
				MarkdownDescription: "Whether to skip appending X-Forwarded-For headers.",
			},
			// TimeoutIdle sets the idle timeout for connections
			"timeout_idle": resource_schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "The idle timeout for connections.",
			},
			// TimeoutRead sets the read timeout for connections
			"timeout_read": resource_schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "The read timeout for connections.",
			},
			// TimeoutWrite sets the write timeout for connections
			"timeout_write": resource_schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "The write timeout for connections.",
			},
			// TracingSampleRate sets the sampling rate for tracing
			"tracing_sample_rate": resource_schema.Float64Attribute{
				Optional:            true,
				MarkdownDescription: "The sampling rate for tracing.",
			},
		},
	}
}

// Configure sets up the ClusterSettingsResource with provider-specific data
func (r *ClusterSettingsResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Check if provider data is available
	if req.ProviderData == nil {
		return
	}

	// Attempt to cast the provider data to the expected type
	provider, ok := req.ProviderData.(*pomeriumZeroProvider)
	if !ok {
		// If the cast fails, add an error to the diagnostics
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *pomeriumZeroProvider, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	// Set the resource's client, token, and organizationID from the provider data
	r.client = provider.client
	r.token = provider.token
	r.organizationID = provider.organizationID
}

// Create handles the creation of a new ClusterSettingsResource
func (r *ClusterSettingsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Initialize a new ClusterSettingsResourceModel to hold the planned state
	var plan ClusterSettingsResourceModel

	// Retrieve the planned state from the CreateRequest
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	log.Printf("[DEBUG] Creating cluster settings for cluster: %s", plan.ID.ValueString())

	// Convert the plan to a CreateClusterSettingsRequest
	settingsReq := createClusterSettingsRequest(plan)

	// Call the API to create the cluster settings
	settings, err := r.createClusterSettings(ctx, settingsReq)
	if err != nil {
		// If there's an error, add it to the diagnostics
		resp.Diagnostics.AddError("Error creating cluster settings", err.Error())
		return
	}

	// Update the plan with the ID returned from the API
	plan.ID = types.StringValue(settings.ID)

	// Set the updated plan as the new state
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

// Read handles the reading of cluster settings from the API and updates the Terraform state
func (r *ClusterSettingsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Initialize a variable to hold the current state
	var state ClusterSettingsResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get the ID of the cluster settings from the state
	id := state.ID.ValueString()
	log.Printf("[DEBUG] Reading cluster settings for cluster: %s", id)

	// Call the API to get the current cluster settings
	settings, err := r.getClusterSettings(ctx, id)
	if err != nil {
		// If the settings are not found, remove the resource from the state
		if strings.Contains(err.Error(), "settings not found") {
			resp.State.RemoveResource(ctx)
			return
		}
		// If there's any other error, add it to the diagnostics
		resp.Diagnostics.AddError("Error reading cluster settings", err.Error())
		return
	}

	// Update the state with the fetched settings
	updateClusterSettingsResourceModel(&state, settings)
	// Ensure the ID in the state matches the one from the API

	// Set the updated state
	diags = resp.State.Set(ctx, &state)
	// Append any diagnostics that occurred during state setting
	resp.Diagnostics.Append(diags...)
}

// Update handles the update operation for the ClusterSettingsResource
func (r *ClusterSettingsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Initialize a new ClusterSettingsResourceModel to hold the planned state
	var plan ClusterSettingsResourceModel

	// Retrieve the planned state from the UpdateRequest
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Extract the ID from the plan
	id := plan.ID.ValueString()
	log.Printf("[DEBUG] Updating cluster settings for cluster: %s", id)

	// Convert the plan to an UpdateClusterSettingsRequest
	settingsReq := updateClusterSettingsRequest(plan)
	// Call the API to update the cluster settings
	settings, err := r.updateClusterSettings(ctx, id, settingsReq)
	if err != nil {
		// If there's an error, add it to the diagnostics
		resp.Diagnostics.AddError("Error updating cluster settings", err.Error())
		return
	}

	// Update the plan with the response from the API
	updateClusterSettingsResourceModel(&plan, settings)

	// Set the updated plan as the new state
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

// Delete handles the deletion of a ClusterSettingsResource
func (r *ClusterSettingsResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Initialize a variable to hold the current state
	var state ClusterSettingsResourceModel

	// Retrieve the current state from the DeleteRequest
	diags := req.State.Get(ctx, &state)

	// Append any diagnostics to the response
	resp.Diagnostics.Append(diags...)

	// If there are any errors in the diagnostics, return early
	if resp.Diagnostics.HasError() {
		return
	}

	// Extract the ID from the state
	id := state.ID.ValueString()

	// Call the API to delete the cluster settings
	err := r.deleteClusterSettings(ctx, id)

	// If there's an error during deletion, add it to the diagnostics
	if err != nil {
		resp.Diagnostics.AddError("Error deleting cluster settings", err.Error())
		return
	}

	// If we reach here, the deletion was successful
	// Terraform will automatically remove the resource from the state
}

// ImportState fetches the current state of the resource from the API
func (r *ClusterSettingsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	id := req.ID

	// Set the cluster_id
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), id)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Fetch the cluster settings
	settings, err := r.getClusterSettings(ctx, id)
	if err != nil {
		resp.Diagnostics.AddError("Error importing cluster settings", fmt.Sprintf("Unable to read cluster settings for %s, error: %s", id, err))
		return
	}

	// Update the state with the fetched settings data
	var state ClusterSettingsResourceModel
	updateClusterSettingsResourceModel(&state, settings)
	state.ID = types.StringValue(settings.ID)

	// Set the full state
	diags := resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

// API helper functions
// These functions interact with the Pomerium Zero API to manage cluster settings

// createClusterSettings sends a POST request to create new cluster settings
func (r *ClusterSettingsResource) createClusterSettings(ctx context.Context, settings CreateClusterSettingsRequest) (*ClusterSettings, error) {
	// Construct the API URL
	url := fmt.Sprintf("%s/organizations/%s/clusters/%s/settings", apiBaseURL, r.organizationID, settings.ID)

	// Marshal the settings into JSON
	body, err := json.Marshal(settings)
	if err != nil {
		return nil, fmt.Errorf("error marshaling settings: %w", err)
	}

	// Create a new HTTP POST request with the marshaled settings
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	// Set the necessary headers
	req.Header.Set("Authorization", "Bearer "+r.token)
	req.Header.Set("Content-Type", "application/json")

	// Send the HTTP request
	resp, err := r.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making request: %w", err)
	}
	defer resp.Body.Close()

	// Check if the response status code is not 201 Created
	if resp.StatusCode != http.StatusCreated {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status code: %d. Response body: %s", resp.StatusCode, string(bodyBytes))
	}

	// Decode the response body into a ClusterSettings struct
	var createdSettings ClusterSettings
	if err := json.NewDecoder(resp.Body).Decode(&createdSettings); err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}

	// Return the created settings
	return &createdSettings, nil
}

// getClusterSettings retrieves the cluster settings from the API
func (r *ClusterSettingsResource) getClusterSettings(ctx context.Context, id string) (*ClusterSettings, error) {
	// Construct the API URL
	url := fmt.Sprintf("%s/organizations/%s/clusters/%s/settings", apiBaseURL, r.organizationID, id)
	log.Printf("[DEBUG] Making GET request to URL: %s", url)

	// Create a new HTTP request
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	// Set request headers
	req.Header.Set("Authorization", "Bearer "+r.token)
	req.Header.Set("Content-Type", "application/json")
	log.Printf("[DEBUG] Request headers: %+v", req.Header)

	// Send the HTTP request
	resp, err := r.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making request: %w", err)
	}
	defer resp.Body.Close()

	log.Printf("[DEBUG] Response status code: %d", resp.StatusCode)

	// Check for non-OK status codes
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		log.Printf("[DEBUG] Response body: %s", string(bodyBytes))
		return nil, fmt.Errorf("unexpected status code: %d. Response body: %s", resp.StatusCode, string(bodyBytes))
	}

	// Decode the response body into ClusterSettings struct
	var settings ClusterSettings
	if err := json.NewDecoder(resp.Body).Decode(&settings); err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}

	// Ensure the ID is not updated with the response ID
	settings.ID = id

	return &settings, nil
}

// updateClusterSettings sends a PUT request to update existing cluster settings
func (r *ClusterSettingsResource) updateClusterSettings(ctx context.Context, id string, settings UpdateClusterSettingsRequest) (*ClusterSettings, error) {
	// Construct the API URL
	url := fmt.Sprintf("%s/organizations/%s/clusters/%s/settings", apiBaseURL, r.organizationID, id)

	// Marshal the settings into JSON
	body, err := json.Marshal(settings)
	if err != nil {
		return nil, fmt.Errorf("error marshaling settings: %w", err)
	}

	// Create a new HTTP PUT request with the marshaled settings
	req, err := http.NewRequestWithContext(ctx, "PUT", url, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	// Set the necessary headers
	req.Header.Set("Authorization", "Bearer "+r.token)
	req.Header.Set("Content-Type", "application/json")

	// Send the HTTP request
	resp, err := r.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making request: %w", err)
	}
	defer resp.Body.Close()

	// Check if the response status code is not 200 OK
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status code: %d. Response body: %s", resp.StatusCode, string(bodyBytes))
	}

	// Decode the response body into a ClusterSettings struct
	var updatedSettings ClusterSettings
	if err := json.NewDecoder(resp.Body).Decode(&updatedSettings); err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}

	// Return the updated settings
	return &updatedSettings, nil
}

// deleteClusterSettings sends a DELETE request to remove cluster settings
func (r *ClusterSettingsResource) deleteClusterSettings(ctx context.Context, id string) error {
	// Construct the API URL for deleting cluster settings
	url := fmt.Sprintf("%s/organizations/%s/clusters/%s/settings", apiBaseURL, r.organizationID, id)

	// Create a new HTTP DELETE request with context
	req, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("error creating request: %w", err)
	}

	// Set the Authorization header with the bearer token
	req.Header.Set("Authorization", "Bearer "+r.token)

	// Send the HTTP request
	resp, err := r.client.Do(req)
	if err != nil {
		return fmt.Errorf("error making request: %w", err)
	}
	defer resp.Body.Close()

	// Check if the response status code is not 204 No Content
	if resp.StatusCode != http.StatusNoContent {
		// Read the response body
		bodyBytes, _ := io.ReadAll(resp.Body)
		// Return an error with the unexpected status code and response body
		return fmt.Errorf("unexpected status code: %d. Response body: %s", resp.StatusCode, string(bodyBytes))
	}

	// If we reach here, the deletion was successful
	return nil
}

// Helper functions for request/response mapping
// These functions help map the API request and response data to the Terraform resource model

// updateClusterSettingsResourceModel updates the ClusterSettingsResourceModel with the ClusterSettings data
func updateClusterSettingsResourceModel(model *ClusterSettingsResourceModel, settings *ClusterSettings) {
	// Do not update the ID with the response ID, the API returns a different ID, but the ID should
	// remain the same as the one in the state, which is the cluster ID, also known as the namespace ID.
	// model.ID = types.StringValue(settings.ID)
	model.Address = types.StringValue(settings.Address)
	model.AuthenticateServiceUrl = types.StringValue(settings.AuthenticateServiceUrl)
	model.AutoApplyChangesets = types.BoolValue(settings.AutoApplyChangesets)
	model.CookieExpire = types.StringValue(settings.CookieExpire)
	model.CookieHttpOnly = types.BoolValue(settings.CookieHttpOnly)
	model.CookieName = types.StringValue(settings.CookieName)
	model.DefaultUpstreamTimeout = types.StringValue(settings.DefaultUpstreamTimeout)
	model.DNSLookupFamily = types.StringValue(settings.DNSLookupFamily)
	model.IdentityProvider = types.StringValue(settings.IdentityProvider)
	model.IdentityProviderClientId = types.StringValue(settings.IdentityProviderClientId)
	model.IdentityProviderClientSecret = types.StringValue(settings.IdentityProviderClientSecret)
	model.IdentityProviderUrl = types.StringValue(settings.IdentityProviderUrl)
	model.LogLevel = types.StringValue(settings.LogLevel)
	model.PassIdentityHeaders = types.BoolValue(settings.PassIdentityHeaders)
	model.ProxyLogLevel = types.StringValue(settings.ProxyLogLevel)
	model.SkipXffAppend = types.BoolValue(settings.SkipXffAppend)
	model.TimeoutIdle = types.StringValue(settings.TimeoutIdle)
	model.TimeoutRead = types.StringValue(settings.TimeoutRead)
	model.TimeoutWrite = types.StringValue(settings.TimeoutWrite)
	model.TracingSampleRate = types.Float64Value(settings.TracingSampleRate)
}

// createClusterSettingsRequest creates a CreateClusterSettingsRequest from the ClusterSettingsResourceModel
func createClusterSettingsRequest(model ClusterSettingsResourceModel) CreateClusterSettingsRequest {
	return CreateClusterSettingsRequest{
		Address:                   model.Address.ValueString(),
		AuthenticateServiceUrl:    model.AuthenticateServiceUrl.ValueString(),
		AutoApplyChangesets:       model.AutoApplyChangesets.ValueBool(),
		CookieExpire:              model.CookieExpire.ValueString(),
		CookieHttpOnly:            model.CookieHttpOnly.ValueBool(),
		CookieName:                model.CookieName.ValueString(),
		DefaultUpstreamTimeout:    model.DefaultUpstreamTimeout.ValueString(),
		DNSLookupFamily:           model.DNSLookupFamily.ValueString(),
		IdentityProvider:          model.IdentityProvider.ValueString(),
		IdentityProviderClientId:  model.IdentityProviderClientId.ValueString(),
		IdentityProviderClientSecret: model.IdentityProviderClientSecret.ValueString(),
		IdentityProviderUrl:       model.IdentityProviderUrl.ValueString(),
		LogLevel:                  model.LogLevel.ValueString(),
		PassIdentityHeaders:       model.PassIdentityHeaders.ValueBool(),
		ProxyLogLevel:             model.ProxyLogLevel.ValueString(),
		SkipXffAppend:             model.SkipXffAppend.ValueBool(),
		TimeoutIdle:               model.TimeoutIdle.ValueString(),
		TimeoutRead:               model.TimeoutRead.ValueString(),
		TimeoutWrite:              model.TimeoutWrite.ValueString(),
		TracingSampleRate:         model.TracingSampleRate.ValueFloat64(),
	}
}

// updateClusterSettingsRequest creates an UpdateClusterSettingsRequest from the ClusterSettingsResourceModel
func updateClusterSettingsRequest(model ClusterSettingsResourceModel) UpdateClusterSettingsRequest {
	return UpdateClusterSettingsRequest{
		Address:                   model.Address.ValueString(),
		AuthenticateServiceUrl:    model.AuthenticateServiceUrl.ValueString(),
		AutoApplyChangesets:       model.AutoApplyChangesets.ValueBool(),
		CookieExpire:              model.CookieExpire.ValueString(),
		CookieHttpOnly:            model.CookieHttpOnly.ValueBool(),
		CookieName:                model.CookieName.ValueString(),
		DefaultUpstreamTimeout:    model.DefaultUpstreamTimeout.ValueString(),
		DNSLookupFamily:           model.DNSLookupFamily.ValueString(),
		IdentityProvider:          model.IdentityProvider.ValueString(),
		IdentityProviderClientId:  model.IdentityProviderClientId.ValueString(),
		IdentityProviderClientSecret: model.IdentityProviderClientSecret.ValueString(),
		IdentityProviderUrl:       model.IdentityProviderUrl.ValueString(),
		LogLevel:                  model.LogLevel.ValueString(),
		PassIdentityHeaders:       model.PassIdentityHeaders.ValueBool(),
		ProxyLogLevel:             model.ProxyLogLevel.ValueString(),
		SkipXffAppend:             model.SkipXffAppend.ValueBool(),
		TimeoutIdle:               model.TimeoutIdle.ValueString(),
		TimeoutRead:               model.TimeoutRead.ValueString(),
		TimeoutWrite:              model.TimeoutWrite.ValueString(),
		TracingSampleRate:         model.TracingSampleRate.ValueFloat64(),
	}
}

// API data structures
// These structures represent the data exchanged with the Pomerium Zero API
// CreateClusterSettingsRequest is used to create new cluster settings
type CreateClusterSettingsRequest struct {
	ID                        string  `json:"id"`
	Address                   string  `json:"address,omitempty"`
	AuthenticateServiceUrl    string  `json:"authenticateServiceUrl,omitempty"`
	AutoApplyChangesets       bool    `json:"autoApplyChangesets,omitempty"`
	CookieExpire              string  `json:"cookieExpire,omitempty"`
	CookieHttpOnly            bool    `json:"cookieHttpOnly,omitempty"`
	CookieName                string  `json:"cookieName,omitempty"`
	DefaultUpstreamTimeout    string  `json:"defaultUpstreamTimeout,omitempty"`
	DNSLookupFamily           string  `json:"dnsLookupFamily,omitempty"`
	IdentityProvider          string  `json:"identityProvider,omitempty"`
	IdentityProviderClientId  string  `json:"identityProviderClientId,omitempty"`
	IdentityProviderClientSecret string `json:"identityProviderClientSecret,omitempty"`
	IdentityProviderUrl       string  `json:"identityProviderUrl,omitempty"`
	LogLevel                  string  `json:"logLevel,omitempty"`
	PassIdentityHeaders       bool    `json:"passIdentityHeaders,omitempty"`
	ProxyLogLevel             string  `json:"proxyLogLevel,omitempty"`
	SkipXffAppend             bool    `json:"skipXffAppend,omitempty"`
	TimeoutIdle               string  `json:"timeoutIdle,omitempty"`
	TimeoutRead               string  `json:"timeoutRead,omitempty"`
	TimeoutWrite              string  `json:"timeoutWrite,omitempty"`
	TracingSampleRate         float64 `json:"tracingSampleRate,omitempty"`
}

// UpdateClusterSettingsRequest is used to update existing cluster settings
type UpdateClusterSettingsRequest struct {
	Address                   string  `json:"address,omitempty"`
	AuthenticateServiceUrl    string  `json:"authenticateServiceUrl,omitempty"`
	AutoApplyChangesets       bool    `json:"autoApplyChangesets,omitempty"`
	CookieExpire              string  `json:"cookieExpire,omitempty"`
	CookieHttpOnly            bool    `json:"cookieHttpOnly,omitempty"`
	CookieName                string  `json:"cookieName,omitempty"`
	DefaultUpstreamTimeout    string  `json:"defaultUpstreamTimeout,omitempty"`
	DNSLookupFamily           string  `json:"dnsLookupFamily,omitempty"`
	IdentityProvider          string  `json:"identityProvider,omitempty"`
	IdentityProviderClientId  string  `json:"identityProviderClientId,omitempty"`
	IdentityProviderClientSecret string `json:"identityProviderClientSecret,omitempty"`
	IdentityProviderUrl       string  `json:"identityProviderUrl,omitempty"`
	LogLevel                  string  `json:"logLevel,omitempty"`
	PassIdentityHeaders       bool    `json:"passIdentityHeaders"`
	ProxyLogLevel             string  `json:"proxyLogLevel,omitempty"`
	SkipXffAppend             bool    `json:"skipXffAppend"`
	TimeoutIdle               string  `json:"timeoutIdle,omitempty"`
	TimeoutRead               string  `json:"timeoutRead,omitempty"`
	TimeoutWrite              string  `json:"timeoutWrite,omitempty"`
	TracingSampleRate         float64 `json:"tracingSampleRate,omitempty"`
}

// ClusterSettings represents the cluster settings data returned by the API
type ClusterSettings struct {
	ID                        string  `json:"id"`
	Address                   string  `json:"address"`
	AuthenticateServiceUrl    string  `json:"authenticateServiceUrl"`
	AutoApplyChangesets       bool    `json:"autoApplyChangesets"`
	CookieExpire              string  `json:"cookieExpire"`
	CookieHttpOnly            bool    `json:"cookieHttpOnly"`
	CookieName                string  `json:"cookieName"`
	DefaultUpstreamTimeout    string  `json:"defaultUpstreamTimeout"`
	DNSLookupFamily           string  `json:"dnsLookupFamily"`
	IdentityProvider          string  `json:"identityProvider"`
	IdentityProviderClientId  string  `json:"identityProviderClientId"`
	IdentityProviderClientSecret string `json:"identityProviderClientSecret"`
	IdentityProviderUrl       string  `json:"identityProviderUrl"`
	LogLevel                  string  `json:"logLevel"`
	PassIdentityHeaders       bool    `json:"passIdentityHeaders"`
	ProxyLogLevel             string  `json:"proxyLogLevel"`
	SkipXffAppend             bool    `json:"skipXffAppend"`
	TimeoutIdle               string  `json:"timeoutIdle"`
	TimeoutRead               string  `json:"timeoutRead"`
	TimeoutWrite              string  `json:"timeoutWrite"`
	TracingSampleRate         float64 `json:"tracingSampleRate"`
}