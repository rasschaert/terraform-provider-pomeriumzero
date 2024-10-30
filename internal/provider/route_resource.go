package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &RouteResource{}
	_ resource.ResourceWithImportState = &RouteResource{}
)

// NewRouteResource is a helper function to simplify the provider implementation.
func NewRouteResource() resource.Resource {
	return &RouteResource{}
}

// RouteResource defines the resource implementation.
type RouteResource struct {
	client         *http.Client
	token          string
	organizationID string
}

// RouteResourceModel describes the resource data model.
type RouteResourceModel struct {
	ID                                        types.String `tfsdk:"id"`
	Name                                      types.String `tfsdk:"name"`
	NamespaceID                               types.String `tfsdk:"namespace_id"`
	From                                      types.String `tfsdk:"from"`
	To                                        types.List   `tfsdk:"to"`
	AllowSpdy                                 types.Bool   `tfsdk:"allow_spdy"`
	AllowWebsockets                           types.Bool   `tfsdk:"allow_websockets"`
	EnableGoogleCloudServerlessAuthentication types.Bool   `tfsdk:"enable_google_cloud_serverless_authentication"`
	PassIdentityHeaders                       types.Bool   `tfsdk:"pass_identity_headers"`
	PreserveHostHeader                        types.Bool   `tfsdk:"preserve_host_header"`
	ShowErrorDetails                          types.Bool   `tfsdk:"show_error_details"`
	TLSSkipVerify                             types.Bool   `tfsdk:"tls_skip_verify"`
	TLSUpstreamAllowRenegotiation             types.Bool   `tfsdk:"tls_upstream_allow_renegotiation"`
	PolicyIDs                                 types.List   `tfsdk:"policy_ids"`
	Prefix                                    types.String `tfsdk:"prefix"`
	PrefixRewrite                             types.String `tfsdk:"prefix_rewrite"`
	KubernetesServiceAccountToken             types.String `tfsdk:"kubernetes_service_account_token"`
}

// Metadata sets the resource type name for the RouteResource.
// It appends "_route" to the resource type name.
func (r *RouteResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_route"
}

// Schema defines the structure and attributes of the RouteResource.
// It specifies the fields that can be used in the Terraform configuration
// to interact with the Pomerium Zero Route resource.
func (r *RouteResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description:         "Route resource in Pomerium Zero.",
		MarkdownDescription: "Manages a route resource in Pomerium Zero.",

		Attributes: map[string]schema.Attribute{
			// ID of the route, automatically generated
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The unique identifier of the route.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			// Name of the route, required field
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The name of the route. Must be unique within the namespace.",
			},
			// Namespace ID for the route, required field
			"namespace_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The ID of the namespace where the route will be created.",
			},
			// Source URL for the route, required field
			"from": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The source URL for the route. This is the URL that Pomerium will listen on.",
			},
			// Destination URLs for the route, required field
			"to": schema.ListAttribute{
				ElementType:         types.StringType,
				Required:            true,
				MarkdownDescription: "A list of destination URLs for the route. These are the backend servers that Pomerium will forward requests to.",
			},
			// Allow SPDY protocol, optional field with default value
			"allow_spdy": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
				MarkdownDescription: "If set to `true`, allows the use of the SPDY protocol for this route.",
			},
			// Allow WebSocket connections, optional field with default value
			"allow_websockets": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
				MarkdownDescription: "If set to `true`, allows WebSocket connections for this route.",
			},
			// Enable Google Cloud Serverless Authentication, optional field with default value
			"enable_google_cloud_serverless_authentication": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
				MarkdownDescription: "If set to `true`, enables Google Cloud Serverless Authentication for this route.",
			},
			// Pass identity headers, optional field
			"pass_identity_headers": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "If set to `true`, passes identity headers to the upstream service.",
			},
			// Preserve host header, optional field with default value
			"preserve_host_header": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
				MarkdownDescription: "If set to `true`, preserves the original host header when proxying requests.",
			},
			// Show error details, optional field with default value
			"show_error_details": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
				MarkdownDescription: "If set to `true`, shows detailed error messages when errors occur.",
			},
			// Skip TLS verification, optional field with default value
			"tls_skip_verify": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
				MarkdownDescription: "If set to `true`, skips TLS verification for upstream connections. Use with caution.",
			},
			// Allow TLS renegotiation for upstream connections, optional field with default value
			"tls_upstream_allow_renegotiation": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
				MarkdownDescription: "If set to `true`, allows TLS renegotiation for upstream connections.",
			},
			// List of policy IDs associated with the route, optional field
			"policy_ids": schema.ListAttribute{
				ElementType:         types.StringType,
				Optional:            true,
				MarkdownDescription: "A list of policy IDs to associate with this route. These policies will be applied to requests matching this route.",
			},
			// URL prefix for the route, optional field
			"prefix": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "The URL prefix for the route. If specified, only requests with this prefix will be matched.",
			},
			// Rewrite prefix for the route, optional field
			"prefix_rewrite": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "If specified, rewrites the URL prefix before forwarding the request to the upstream service.",
			},
			// Kubernetes service account token, optional field
			"kubernetes_service_account_token": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "The Kubernetes service account token to use for authentication.",
				Sensitive:           true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

// Configure sets up the RouteResource with the provider's configuration.
func (r *RouteResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Attempt to cast the provider data to the expected type
	if req.ProviderData == nil {
		return
	}

	// Set the RouteResource fields with the provider's data
	provider, ok := req.ProviderData.(*pomeriumZeroProvider)
	if !ok {
		// If the cast fails, add an error to the diagnostics
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *pomeriumZeroProvider, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	// Set the RouteResource fields with the provider's data
	r.client = provider.client
	r.token = provider.token
	r.organizationID = provider.organizationID
}

// Create handles the creation of a new RouteResource
func (r *RouteResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Initialize a new RouteResourceModel to hold the planned state
	var plan RouteResourceModel

	// Retrieve the planned state from the CreateRequest
	diags := req.Plan.Get(ctx, &plan)

	// Append any diagnostics to the response
	resp.Diagnostics.Append(diags...)

	// If there are any errors in the diagnostics, return early
	if resp.Diagnostics.HasError() {
		return
	}

	// Call the createRoute method to create the route in the external system
	route, err := r.createRoute(ctx, &plan)
	if err != nil {
		// If there's an error, add it to the diagnostics
		resp.Diagnostics.AddError(
			"Error creating route",
			fmt.Sprintf("Could not create route, unexpected error: %s", err),
		)
		return
	}

	// Set the state with the newly created route
	diags = resp.State.Set(ctx, route)

	// Append any diagnostics that occurred during state setting
	resp.Diagnostics.Append(diags...)
}

// Read handles the reading of an existing RouteResource
func (r *RouteResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Initialize a new RouteResourceModel to hold the current state
	var state RouteResourceModel

	// Retrieve the current state from the ReadRequest
	diags := req.State.Get(ctx, &state)

	// Append any diagnostics to the response
	resp.Diagnostics.Append(diags...)

	// If there are any errors in the diagnostics, return early
	if resp.Diagnostics.HasError() {
		return
	}

	// Call the readRoute method to fetch the route from the external system
	route, err := r.readRoute(ctx, state.ID.ValueString())
	if err != nil {
		// If there's an error, add it to the diagnostics
		resp.Diagnostics.AddError(
			"Error Reading Route",
			fmt.Sprintf("Could not read route ID %s: %s", state.ID.ValueString(), err),
		)
		return
	}

	// Log the raw API response for debugging purposes
	log.Printf("[DEBUG] Raw API response for route %s: %+v", state.ID.ValueString(), route)

	// Map the API response to our RouteResourceModel
	newState := mapRouteResponseToModel(ctx, route)

	// Set the new state
	diags = resp.State.Set(ctx, newState)

	// Append any diagnostics that occurred during state setting
	resp.Diagnostics.Append(diags...)
}

// Update handles the update operation for the RouteResource
func (r *RouteResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Initialize a RouteResourceModel to hold the planned state
	var plan RouteResourceModel

	// Get the planned state from the UpdateRequest
	diags := req.Plan.Get(ctx, &plan)

	// Append any diagnostics to the response
	resp.Diagnostics.Append(diags...)

	// If there are any errors in the diagnostics, return early
	if resp.Diagnostics.HasError() {
		return
	}

	// Call the updateRoute method to update the route in the external system
	route, err := r.updateRoute(ctx, &plan)
	if err != nil {
		// If there's an error, add it to the diagnostics
		resp.Diagnostics.AddError(
			"Error Updating Route",
			fmt.Sprintf("Could not update route, unexpected error: %s", err),
		)
		return
	}

	// Set the state with the updated route
	diags = resp.State.Set(ctx, route)

	// Append any diagnostics that occurred during state setting
	resp.Diagnostics.Append(diags...)
}

// Delete handles the deletion of a RouteResource
func (r *RouteResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Initialize a RouteResourceModel to hold the current state
	var state RouteResourceModel

	// Retrieve the current state from the DeleteRequest
	diags := req.State.Get(ctx, &state)

	// Append any diagnostics to the response
	resp.Diagnostics.Append(diags...)

	// If there are any errors in the diagnostics, return early
	if resp.Diagnostics.HasError() {
		return
	}

	// Call the deleteRoute method to delete the route in the external system
	err := r.deleteRoute(ctx, state.ID.ValueString())
	if err != nil {
		// If there's an error, add it to the diagnostics
		resp.Diagnostics.AddError(
			"Error Deleting Route",
			fmt.Sprintf("Could not delete route, unexpected error: %s", err),
		)
		return
	}
	// If we reach here, the route was successfully deleted
}

// ImportState handles the importing of an existing RouteResource
func (r *RouteResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// createRoute creates a new route in the external system
func (r *RouteResource) createRoute(ctx context.Context, plan *RouteResourceModel) (RouteResourceModel, error) {
	// Construct the URL for creating a route
	url := fmt.Sprintf("%s/organizations/%s/routes", apiBaseURL, r.organizationID)

	// Create the request body from the plan
	routeReq := createRouteRequest(plan)
	body, err := json.Marshal(routeReq)
	if err != nil {
		return RouteResourceModel{}, fmt.Errorf("error marshaling route: %w", err)
	}

	// Log the request body for debugging
	log.Printf("[DEBUG] Create route request body: %s", string(body))

	// Create a new HTTP POST request
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(body))
	if err != nil {
		return RouteResourceModel{}, fmt.Errorf("error creating request: %w", err)
	}

	// Set the necessary headers
	req.Header.Set("Authorization", "Bearer "+r.token)
	req.Header.Set("Content-Type", "application/json")

	// Send the request
	resp, err := r.client.Do(req)
	if err != nil {
		return RouteResourceModel{}, fmt.Errorf("error making request: %w", err)
	}
	defer resp.Body.Close()

	// Read the response body
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return RouteResourceModel{}, fmt.Errorf("error reading response body: %w", err)
	}

	// Log the response for debugging
	log.Printf("[DEBUG] Create route response status: %d, body: %s", resp.StatusCode, string(responseBody))

	// Check if the status code indicates a successful creation
	if resp.StatusCode != http.StatusCreated {
		return RouteResourceModel{}, fmt.Errorf("unexpected status code: %d. Response body: %s", resp.StatusCode, string(responseBody))
	}

	// Unmarshal the response body into a map
	var apiResponse map[string]interface{}
	if err := json.Unmarshal(responseBody, &apiResponse); err != nil {
		return RouteResourceModel{}, fmt.Errorf("error unmarshaling response: %w", err)
	}

	// Map the API response to our RouteResourceModel
	return mapRouteResponseToModel(ctx, apiResponse), nil
}

// readRoute fetches the details of a specific route from the API
func (r *RouteResource) readRoute(ctx context.Context, id string) (map[string]interface{}, error) {
	// Construct the URL for the API request
	url := fmt.Sprintf("%s/organizations/%s/routes/%s", apiBaseURL, r.organizationID, id)

	// Create a new GET request with the provided context
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	// Set the Authorization header with the bearer token
	req.Header.Set("Authorization", "Bearer "+r.token)

	// Send the HTTP request
	resp, err := r.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making request: %w", err)
	}
	defer resp.Body.Close()

	// Check if the response status code is OK (200)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Decode the JSON response body into a map
	var route map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&route); err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}

	// Return the decoded route data
	return route, nil
}

// updateRoute updates an existing route in the external system
func (r *RouteResource) updateRoute(ctx context.Context, plan *RouteResourceModel) (RouteResourceModel, error) {
	// Construct the URL for updating a specific route
	url := fmt.Sprintf("%s/organizations/%s/routes/%s", apiBaseURL, r.organizationID, plan.ID.ValueString())

	// Create the request body from the plan
	routeReq := updateRouteRequest(plan)
	body, err := json.Marshal(routeReq)
	if err != nil {
		return RouteResourceModel{}, fmt.Errorf("error marshaling route: %w", err)
	}

	// Create a new HTTP PUT request
	req, err := http.NewRequestWithContext(ctx, "PUT", url, bytes.NewBuffer(body))
	if err != nil {
		return RouteResourceModel{}, fmt.Errorf("error creating request: %w", err)
	}

	// Set the necessary headers
	req.Header.Set("Authorization", "Bearer "+r.token)
	req.Header.Set("Content-Type", "application/json")

	// Send the request
	resp, err := r.client.Do(req)
	if err != nil {
		return RouteResourceModel{}, fmt.Errorf("error making request: %w", err)
	}
	defer resp.Body.Close()

	// Read the response body
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return RouteResourceModel{}, fmt.Errorf("error reading response body: %w", err)
	}

	// Check if the status code indicates a successful update
	if resp.StatusCode != http.StatusOK {
		return RouteResourceModel{}, fmt.Errorf("unexpected status code: %d. Response body: %s", resp.StatusCode, string(responseBody))
	}

	// Unmarshal the response body into a map
	var apiResponse map[string]interface{}
	if err := json.Unmarshal(responseBody, &apiResponse); err != nil {
		return RouteResourceModel{}, fmt.Errorf("error unmarshaling response: %w", err)
	}

	// Map the API response to our RouteResourceModel and return it
	return mapRouteResponseToModel(ctx, apiResponse), nil
}

// deleteRoute sends a DELETE request to remove a specific route from the Pomerium Zero API
func (r *RouteResource) deleteRoute(ctx context.Context, id string) error {
	// Construct the URL for deleting a specific route
	url := fmt.Sprintf("%s/organizations/%s/routes/%s", apiBaseURL, r.organizationID, id)

	// Create a new DELETE request with the provided context
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

	// Check if the response status code is 204 No Content (expected for successful deletion)
	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// If we reach here, the deletion was successful
	return nil
}

// createRouteRequest constructs a map representing the API request payload for creating a route
func createRouteRequest(model *RouteResourceModel) map[string]interface{} {
	// Initialize the request map with required and non-nullable fields
	req := map[string]interface{}{
		"name":        model.Name.ValueString(),
		"namespaceId": model.NamespaceID.ValueString(),
		"from":        model.From.ValueString(),
		"allowSpdy":   model.AllowSpdy.ValueBool(),
		"enableGoogleCloudServerlessAuthentication": model.EnableGoogleCloudServerlessAuthentication.ValueBool(),
		"showErrorDetails":                          model.ShowErrorDetails.ValueBool(),
		"tlsSkipVerify":                             model.TLSSkipVerify.ValueBool(),
		"tlsUpstreamAllowRenegotiation":             model.TLSUpstreamAllowRenegotiation.ValueBool(),
		"kubernetesServiceAccountToken":             model.KubernetesServiceAccountToken.ValueString(),
	}

	// Add 'to' field if it's not null
	if !model.To.IsNull() {
		var to []string
		model.To.ElementsAs(context.Background(), &to, false)
		req["to"] = to
	}

	// Add optional boolean fields if they're not null
	if !model.AllowWebsockets.IsNull() {
		req["allowWebsockets"] = model.AllowWebsockets.ValueBool()
	}
	if !model.PassIdentityHeaders.IsNull() {
		req["passIdentityHeaders"] = model.PassIdentityHeaders.ValueBool()
	}
	if !model.PreserveHostHeader.IsNull() {
		req["preserveHostHeader"] = model.PreserveHostHeader.ValueBool()
	}

	// Add 'policyIds' field if it's not null
	if !model.PolicyIDs.IsNull() {
		var policyIDs []string
		model.PolicyIDs.ElementsAs(context.Background(), &policyIDs, false)
		req["policyIds"] = policyIDs
	}

	// Add optional string fields if they're not null
	if !model.Prefix.IsNull() {
		req["prefix"] = model.Prefix.ValueString()
	}
	if !model.PrefixRewrite.IsNull() {
		req["prefixRewrite"] = model.PrefixRewrite.ValueString()
	}
	if !model.KubernetesServiceAccountToken.IsNull() {
		req["kubernetesServiceAccountToken"] = model.KubernetesServiceAccountToken.ValueString()
	}

	// Return the constructed request map
	return req
}

// updateRouteRequest constructs a map representing the API request payload for updating a route
func updateRouteRequest(model *RouteResourceModel) map[string]interface{} {
	// For this implementation, update request is the same as create request
	return createRouteRequest(model)
}

// mapRouteResponseToModel converts the API response to a RouteResourceModel
func mapRouteResponseToModel(ctx context.Context, apiResponse map[string]interface{}) RouteResourceModel {
	// Initialize the model with required string fields
	model := RouteResourceModel{
		ID:          types.StringValue(apiResponse["id"].(string)),
		Name:        types.StringValue(apiResponse["name"].(string)),
		NamespaceID: types.StringValue(apiResponse["namespaceId"].(string)),
		From:        types.StringValue(apiResponse["from"].(string)),
	}

	// Handle the 'to' field, which is a list of strings
	if to, ok := apiResponse["to"].([]interface{}); ok {
		toList, _ := types.ListValueFrom(ctx, types.StringType, to)
		model.To = toList
	}

	// Helper function to safely convert interface{} to bool
	toBool := func(v interface{}) types.Bool {
		if v == nil {
			return types.BoolNull()
		}
		if b, ok := v.(bool); ok {
			return types.BoolValue(b)
		}
		return types.BoolNull()
	}

	// Set boolean fields using the toBool helper function
	model.AllowSpdy = toBool(apiResponse["allowSpdy"])
	model.AllowWebsockets = toBool(apiResponse["allowWebsockets"])
	model.EnableGoogleCloudServerlessAuthentication = toBool(apiResponse["enableGoogleCloudServerlessAuthentication"])
	model.PassIdentityHeaders = toBool(apiResponse["passIdentityHeaders"])
	model.PreserveHostHeader = toBool(apiResponse["preserveHostHeader"])
	model.ShowErrorDetails = toBool(apiResponse["showErrorDetails"])
	model.TLSSkipVerify = toBool(apiResponse["tlsSkipVerify"])
	model.TLSUpstreamAllowRenegotiation = toBool(apiResponse["tlsUpstreamAllowRenegotiation"])

	// Handle the 'policyIds' field, which is a list of strings
	if policyIDs, ok := apiResponse["policyIds"].([]interface{}); ok {
		policyIDsList, _ := types.ListValueFrom(ctx, types.StringType, policyIDs)
		model.PolicyIDs = policyIDsList
	}

	// Handle optional string fields
	if prefix, ok := apiResponse["prefix"].(string); ok {
		model.Prefix = types.StringValue(prefix)
	}
	if prefixRewrite, ok := apiResponse["prefixRewrite"].(string); ok {
		model.PrefixRewrite = types.StringValue(prefixRewrite)
	}
	if kubernetesServiceAccountToken, ok := apiResponse["kubernetesServiceAccountToken"].(string); ok {
		model.KubernetesServiceAccountToken = types.StringValue(kubernetesServiceAccountToken)
	}

	// Return the populated model
	return model
}
