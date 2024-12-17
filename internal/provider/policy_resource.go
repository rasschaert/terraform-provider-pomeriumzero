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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &PolicyResource{}
var _ resource.ResourceWithImportState = &PolicyResource{}

// NewPolicyResource creates a new PolicyResource.
func NewPolicyResource() resource.Resource {
	return &PolicyResource{}
}

// PolicyResource defines the resource implementation.
type PolicyResource struct {
	client         *http.Client
	token          string
	organizationID string
}

// PolicyResourceModel describes the resource data model.
type PolicyResourceModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	Enforced    types.Bool   `tfsdk:"enforced"`
	Explanation types.String `tfsdk:"explanation"`
	NamespaceID types.String `tfsdk:"namespace_id"`
	PPL         types.String `tfsdk:"ppl"`
	Remediation types.String `tfsdk:"remediation"`
}

// Metadata sets the resource type name for the PolicyResource.
// It appends "_policy" to the resource type name.
func (r *PolicyResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_policy"
}

// Schema defines the structure and attributes of the PolicyResource.
// It specifies the fields that can be used in the Terraform configuration
// to interact with the Pomerium Zero Policy resource.
func (r *PolicyResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Pomerium Zero Policy.",
		Attributes: map[string]schema.Attribute{
			// ID is a computed attribute representing the unique identifier of the policy
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The unique identifier of the policy.",
			},
			// Name is a required attribute for the policy name
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The name of the policy.",
			},
			// Description is a required attribute providing details about the policy
			"description": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "A description of the policy.",
			},
			// Enforced is a required boolean attribute indicating if the policy is active
			"enforced": schema.BoolAttribute{
				Required:            true,
				MarkdownDescription: "Whether the policy is enforced or not.",
			},
			// Explanation is a required attribute providing context for the policy
			"explanation": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "An explanation of the policy.",
			},
			// NamespaceID is a required attribute linking the policy to a specific namespace
			"namespace_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The ID of the namespace this policy belongs to.",
			},
			// PPL is a required attribute containing the Pomerium Policy Language definition
			"ppl": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The Pomerium Policy Language (PPL) definition for this policy.",
			},
			// Remediation is a required attribute providing guidance on addressing policy violations
			"remediation": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Instructions for remediating policy violations.",
			},
		},
	}
}

// Configure prepares a Pomerium Zero API client for the PolicyResource.
func (r *PolicyResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Check if the provider data is nil
	if req.ProviderData == nil {
		return
	}

	// Type assert the provider data to the expected type
	provider, ok := req.ProviderData.(*pomeriumZeroProvider)
	if !ok {
		// If the provider data is not the expected type, add an error to the response diagnostics
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *pomeriumZeroProvider, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	// Set the provider data as the ResourceData
	r.client = provider.client
	r.token = provider.token
	r.organizationID = provider.organizationID
}

// Create creates a new policy in Pomerium Zero.
func (r *PolicyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Initialize a new PolicyResourceModel
	var plan PolicyResourceModel

	// Get the plan data
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	log.Printf("[DEBUG] Creating policy with name: %s", plan.Name.ValueString())

	// Create a policy request from the plan
	policyReq := createPolicyRequest(plan)

	// Call the API to create the policy
	policy, err := r.createPolicy(ctx, policyReq)
	if err != nil {
		// If there's an error, add it to the diagnostics
		resp.Diagnostics.AddError("Error creating policy", err.Error())
		return
	}

	// Set the ID of the newly created policy in the plan
	plan.ID = types.StringValue(policy.ID)

	// Update the Terraform state with the complete plan
	diags = resp.State.Set(ctx, plan)

	// Append any diagnostics from setting the state
	resp.Diagnostics.Append(diags...)
}

// Read retrieves the current state of a PolicyResource from the API
func (r *PolicyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Initialize a PolicyResourceModel to hold the current state
	var state PolicyResourceModel

	// Retrieve the current state from Terraform
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Fetch the policy from the API using its ID
	policy, err := r.getPolicy(ctx, state.ID.ValueString())
	if err != nil {
		if strings.Contains(err.Error(), "policy not found") {
			// If the policy is not found in the API, remove it from Terraform state
			resp.State.RemoveResource(ctx)
			return
		}
		// If there's any other error, add it to the diagnostics
		resp.Diagnostics.AddError("Error reading policy", err.Error())
		return
	}

	// Update the state with the fetched policy data
	updatePolicyResourceModel(&state, policy)

	// Explicitly set the ID field
	state.ID = types.StringValue(policy.ID)

	// Log a message if the API returns a 'routes' field, which we're ignoring
	if len(policy.Routes) > 0 {
		log.Printf("[INFO] Ignoring 'routes' field returned by API for policy %s", state.ID.ValueString())
	}

	// Set the updated state in Terraform
	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
}

// Update handles the update operation for a PolicyResource
func (r *PolicyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Initialize a PolicyResourceModel to hold the planned changes
	var plan PolicyResourceModel
	// Get the planned changes from the request
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Retrieve the current state to get the ID
	var state PolicyResourceModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Extract the policy ID from the current state
	policyID := state.ID.ValueString()
	log.Printf("[DEBUG] Updating policy with ID: %s", policyID)

	// Create an update request from the planned changes
	policyReq := updatePolicyRequest(plan)
	// Call the API to update the policy
	policy, err := r.updatePolicy(ctx, policyID, policyReq)
	if err != nil {
		// If there's an error, add it to the diagnostics
		resp.Diagnostics.AddError("Error updating policy", err.Error())
		return
	}

	// Update the plan with the response from the API
	updatePolicyResourceModel(&plan, policy)
	// Set the updated plan as the new state
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

// Delete handles the deletion of a PolicyResource
func (r *PolicyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Initialize a PolicyResourceModel to hold the current state
	var state PolicyResourceModel

	// Retrieve the current state from the request
	diags := req.State.Get(ctx, &state)

	// Append any diagnostics to the response
	resp.Diagnostics.Append(diags...)

	// If there are any errors in the diagnostics, return early
	if resp.Diagnostics.HasError() {
		return
	}

	// Call the deletePolicy method to remove the policy from the API
	err := r.deletePolicy(ctx, state.ID.ValueString())

	// If there's an error during deletion, add it to the diagnostics
	if err != nil {
		resp.Diagnostics.AddError("Error deleting policy", err.Error())
		return
	}

	// If we reach here, the policy was successfully deleted
}

// ImportState handles the import of an existing policy into Terraform state
func (r *PolicyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Extract the policy ID from the import request
	policyID := req.ID

	// Set the ID attribute in the Terraform state
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), policyID)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Fetch the policy details from the API
	policy, err := r.getPolicy(ctx, policyID)
	if err != nil {
		// If there's an error fetching the policy, add it to the diagnostics
		resp.Diagnostics.AddError("Error importing policy", fmt.Sprintf("Unable to read policy %s, error: %s", policyID, err))
		return
	}

	// Create a new PolicyResourceModel to hold the imported state
	var state PolicyResourceModel
	// Populate the state with the fetched policy data
	updatePolicyResourceModel(&state, policy)

	// Set the full state in Terraform
	diags := resp.State.Set(ctx, &state)
	// Append any diagnostics from setting the state
	resp.Diagnostics.Append(diags...)
}

// API helper functions
// These functions interact with the Pomerium Zero API to create, read, update, and delete policies

// createPolicy creates a new policy in Pomerium Zero
func (r *PolicyResource) createPolicy(ctx context.Context, policy CreatePolicyRequest) (*Policy, error) {
	// Construct the URL for the API endpoint
	url := fmt.Sprintf("%s/organizations/%s/policies", apiBaseURL, r.organizationID)
	body, err := json.Marshal(policy)
	if err != nil {
		return nil, fmt.Errorf("error marshaling policy: %w", err)
	}

	log.Printf("[DEBUG] Create policy request body: %s", string(body))

	// Create a new HTTP POST request with the given context
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+r.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := r.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making request: %w", err)
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %w", err)
	}

	log.Printf("[DEBUG] Create policy response status: %d, body: %s", resp.StatusCode, string(responseBody))

	if resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("unexpected status code: %d. Response body: %s", resp.StatusCode, string(responseBody))
	}

	var createdPolicy Policy
	if err := json.Unmarshal(responseBody, &createdPolicy); err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %w", err)
	}

	return &createdPolicy, nil
}

// getPolicy retrieves a policy from Pomerium Zero by its ID
func (r *PolicyResource) getPolicy(ctx context.Context, policyID string) (*Policy, error) {
	// Construct the URL for the API endpoint
	url := fmt.Sprintf("%s/organizations/%s/policies/%s", apiBaseURL, r.organizationID, policyID)

	// Create a new HTTP GET request with the given context
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	// Set the necessary headers for authentication and content type
	req.Header.Set("Authorization", "Bearer "+r.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := r.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var policy Policy
	if err := json.NewDecoder(resp.Body).Decode(&policy); err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}

	return &policy, nil
}

// updatePolicy updates a policy in Pomerium Zero
func (r *PolicyResource) updatePolicy(ctx context.Context, policyID string, policy UpdatePolicyRequest) (*Policy, error) {
	log.Printf("[DEBUG] Updating policy with ID: %s", policyID)
	// Construct the URL for the API endpoint
	url := fmt.Sprintf("%s/organizations/%s/policies/%s", apiBaseURL, r.organizationID, policyID)

	// Marshal the policy data into a JSON body
	body, err := json.Marshal(policy)
	if err != nil {
		return nil, fmt.Errorf("error marshaling policy: %w", err)
	}

	// Create a new HTTP PUT request with the given context
	req, err := http.NewRequestWithContext(ctx, "PUT", url, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	// Set the necessary headers for authentication and content type
	req.Header.Set("Authorization", "Bearer "+r.token)
	req.Header.Set("Content-Type", "application/json")

	// Send the HTTP request
	resp, err := r.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making request: %w", err)
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %w", err)
	}

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("policy with ID %s not found. It may have been deleted outside of Terraform", policyID)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d. Response body: %s", resp.StatusCode, string(responseBody))
	}

	var updatedPolicy Policy
	if err := json.Unmarshal(responseBody, &updatedPolicy); err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %w", err)
	}

	return &updatedPolicy, nil
}

// deletePolicy removes a policy from Pomerium Zero
func (r *PolicyResource) deletePolicy(ctx context.Context, policyID string) error {
	// Construct the URL for the API endpoint
	url := fmt.Sprintf("%s/organizations/%s/policies/%s", apiBaseURL, r.organizationID, policyID)

	// Create a new HTTP DELETE request with the given context
	req, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return err
	}
	// Set the necessary headers for authentication
	req.Header.Set("Authorization", "Bearer "+r.token)

	// Send the HTTP request
	resp, err := r.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

// Helper functions for request/response mapping
// These functions convert between the Terraform model and the API request/response formats

// createPolicyRequest creates a CreatePolicyRequest from a PolicyResourceModel
func createPolicyRequest(model PolicyResourceModel) CreatePolicyRequest {
	// Declare a variable to hold the unmarshaled PPL data
	var ppl interface{}

	// Attempt to unmarshal the PPL string from the model into the ppl variable
	err := json.Unmarshal([]byte(model.PPL.ValueString()), &ppl)

	// Check if there was an error during unmarshaling
	if err != nil {
		// Log the error if unmarshaling fails
		// Note: Consider handling this error more robustly in production code
		log.Printf("[ERROR] Failed to unmarshal PPL: %v", err)
	}

	return CreatePolicyRequest{
		Name:        model.Name.ValueString(),
		Description: model.Description.ValueString(),
		Enforced:    model.Enforced.ValueBool(),
		Explanation: model.Explanation.ValueString(),
		NamespaceID: model.NamespaceID.ValueString(),
		PPL:         ppl,
		Remediation: model.Remediation.ValueString(),
	}
}

// updatePolicyRequest creates an UpdatePolicyRequest from a PolicyResourceModel
func updatePolicyRequest(model PolicyResourceModel) UpdatePolicyRequest {
	var ppl interface{}
	err := json.Unmarshal([]byte(model.PPL.ValueString()), &ppl)
	if err != nil {
		// Handle error (log it or return an error)
		log.Printf("[ERROR] Failed to unmarshal PPL: %v", err)
	}

	return UpdatePolicyRequest{
		NamespaceID: model.NamespaceID.ValueString(),
		Name:        model.Name.ValueString(),
		Enforced:    model.Enforced.ValueBool(),
		PPL:         ppl,
		Description: model.Description.ValueString(),
		Explanation: model.Explanation.ValueString(),
		Remediation: model.Remediation.ValueString(),
	}
}

// updatePolicyResourceModel updates a PolicyResourceModel with the data from a Policy
func updatePolicyResourceModel(model *PolicyResourceModel, policy *Policy) {
	// Use stringOrEmpty to convert potential null values to empty strings
	model.ID = types.StringValue(policy.ID)
	model.Name = types.StringValue(stringOrEmpty(policy.Name))
	model.Description = types.StringValue(stringOrEmpty(policy.Description))
	model.Enforced = types.BoolValue(policy.Enforced)
	model.Explanation = types.StringValue(stringOrEmpty(policy.Explanation))
	model.NamespaceID = types.StringValue(policy.NamespaceID)
	model.PPL = types.StringValue(string(policy.PPL))
	model.Remediation = types.StringValue(stringOrEmpty(policy.Remediation))
}

// stringOrEmpty is a helper function that converts null string values to empty strings
// This ensures that null values from the API are stored as empty strings in Terraform state
func stringOrEmpty(s string) string {
	if s == "null" {
		return ""
	}
	return s
}

// API data structures
// These structures represent the data exchanged with the Pomerium Zero API

// CreatePolicyRequest represents the request body for creating a policy
type CreatePolicyRequest struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Enforced    bool        `json:"enforced"`
	Explanation string      `json:"explanation"`
	NamespaceID string      `json:"namespaceId"`
	PPL         interface{} `json:"ppl"`
	Remediation string      `json:"remediation"`
}

// UpdatePolicyRequest represents the request body for updating a policy
type UpdatePolicyRequest struct {
	NamespaceID string      `json:"namespaceId"`
	Name        string      `json:"name"`
	Enforced    bool        `json:"enforced"`
	PPL         interface{} `json:"ppl"`
	Description string      `json:"description"`
	Explanation string      `json:"explanation"`
	Remediation string      `json:"remediation"`
}

// GetSchemaResourceData retrieves all policies and returns a JSON representation of their key attributes
func (r *PolicyResource) GetSchemaResourceData(ctx context.Context) ([]byte, error) {
	// Fetch all policies from the API
	policies, err := r.listPolicies(ctx)
	if err != nil {
		return nil, fmt.Errorf("error listing policies: %w", err)
	}

	// Create a slice to hold simplified policy data
	var resources []map[string]interface{}
	for _, policy := range policies {
		// For each policy, create a map with key attributes
		resources = append(resources, map[string]interface{}{
			"id":       policy.ID,
			"name":     policy.Name,
			"enforced": policy.Enforced,
		})
	}

	// Convert the slice of maps to a JSON string with indentation
	data, err := json.MarshalIndent(resources, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("error marshaling JSON: %w", err)
	}

	return data, nil
}

// listPolicies retrieves all policies from the Pomerium Zero API
func (r *PolicyResource) listPolicies(ctx context.Context) ([]*Policy, error) {
	// Construct the URL for the API endpoint
	url := fmt.Sprintf("%s/organizations/%s/policies", apiBaseURL, r.organizationID)

	// Create a new HTTP GET request with the given context
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	// Set the necessary headers for authentication and content type
	req.Header.Set("Authorization", "Bearer "+r.token)
	req.Header.Set("Content-Type", "application/json")

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

	// Decode the JSON response body into a slice of Policy structs
	var policies []*Policy
	if err := json.NewDecoder(resp.Body).Decode(&policies); err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}

	return policies, nil
}
