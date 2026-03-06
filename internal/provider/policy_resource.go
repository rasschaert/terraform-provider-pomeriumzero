package provider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
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
	client *apiClient
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
func (r *PolicyResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_policy"
}

// Schema defines the structure and attributes of the PolicyResource.
func (r *PolicyResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Pomerium Zero Policy.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The unique identifier of the policy.",
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The name of the policy.",
			},
			"description": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "A description of the policy.",
			},
			"enforced": schema.BoolAttribute{
				Required:            true,
				MarkdownDescription: "Whether the policy is enforced or not.",
			},
			"explanation": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "An explanation of the policy.",
			},
			"namespace_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The ID of the namespace this policy belongs to.",
			},
			"ppl": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The Pomerium Policy Language (PPL) definition for this policy.",
			},
			"remediation": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Instructions for remediating policy violations.",
			},
		},
	}
}

// Configure prepares the PolicyResource with provider data.
func (r *PolicyResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*apiClient)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *apiClient, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}
	r.client = client
}

// Create creates a new policy in Pomerium Zero.
func (r *PolicyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan PolicyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var policy Policy
	if err := r.client.post(ctx, r.client.policiesURL(), createPolicyRequest(ctx, plan), http.StatusCreated, &policy); err != nil {
		resp.Diagnostics.AddError("Error creating policy", err.Error())
		return
	}

	updatePolicyResourceModel(&plan, &policy)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

// Read retrieves the current state of a PolicyResource from the API.
func (r *PolicyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state PolicyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var policy Policy
	if err := r.client.get(ctx, r.client.policyURL(state.ID.ValueString()), &policy); err != nil {
		if errors.Is(err, errNotFound) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading policy", err.Error())
		return
	}

	updatePolicyResourceModel(&state, &policy)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

// Update handles the update operation for a PolicyResource.
func (r *PolicyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan PolicyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state PolicyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var policy Policy
	if err := r.client.put(ctx, r.client.policyURL(state.ID.ValueString()), updatePolicyRequest(ctx, plan), &policy); err != nil {
		if errors.Is(err, errNotFound) {
			resp.Diagnostics.AddError("Error updating policy",
				fmt.Sprintf("Policy %s not found. It may have been deleted outside of Terraform.", state.ID.ValueString()))
			return
		}
		resp.Diagnostics.AddError("Error updating policy", err.Error())
		return
	}

	updatePolicyResourceModel(&plan, &policy)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

// Delete handles the deletion of a PolicyResource.
func (r *PolicyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state PolicyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.delete(ctx, r.client.policyURL(state.ID.ValueString())); err != nil {
		resp.Diagnostics.AddError("Error deleting policy", err.Error())
	}
}

// ImportState handles the import of an existing policy into Terraform state.
func (r *PolicyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var policy Policy
	if err := r.client.get(ctx, r.client.policyURL(req.ID), &policy); err != nil {
		resp.Diagnostics.AddError("Error importing policy", fmt.Sprintf("Unable to read policy %s: %s", req.ID, err))
		return
	}

	var state PolicyResourceModel
	updatePolicyResourceModel(&state, &policy)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Helper functions for request/response mapping.

// createPolicyRequest builds a CreatePolicyRequest from a PolicyResourceModel.
func createPolicyRequest(ctx context.Context, model PolicyResourceModel) CreatePolicyRequest {
	var ppl interface{}
	if err := json.Unmarshal([]byte(model.PPL.ValueString()), &ppl); err != nil {
		tflog.Error(ctx, "Failed to unmarshal PPL", map[string]interface{}{"error": err.Error()})
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

// updatePolicyRequest builds an UpdatePolicyRequest from a PolicyResourceModel.
func updatePolicyRequest(ctx context.Context, model PolicyResourceModel) UpdatePolicyRequest {
	var ppl interface{}
	if err := json.Unmarshal([]byte(model.PPL.ValueString()), &ppl); err != nil {
		tflog.Error(ctx, "Failed to unmarshal PPL", map[string]interface{}{"error": err.Error()})
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

// updatePolicyResourceModel updates a PolicyResourceModel with data from a Policy.
func updatePolicyResourceModel(model *PolicyResourceModel, policy *Policy) {
	model.ID = types.StringValue(policy.ID)
	model.Name = types.StringValue(stringOrEmpty(policy.Name))
	model.Description = types.StringValue(stringOrEmpty(policy.Description))
	model.Enforced = types.BoolValue(policy.Enforced)
	model.Explanation = types.StringValue(stringOrEmpty(policy.Explanation))
	model.NamespaceID = types.StringValue(policy.NamespaceID)
	model.PPL = types.StringValue(normalizePPL(policy.PPL))
	model.Remediation = types.StringValue(stringOrEmpty(policy.Remediation))
}

// normalizePPL round-trips the PPL JSON through interface{} to produce a canonical
// representation with sorted keys. This prevents perpetual diffs caused by key ordering
// or whitespace differences between user input and the API response.
func normalizePPL(raw json.RawMessage) string {
	var v interface{}
	if err := json.Unmarshal(raw, &v); err != nil {
		return string(raw)
	}
	normalized, err := json.Marshal(v)
	if err != nil {
		return string(raw)
	}
	return string(normalized)
}

// stringOrEmpty converts the literal string "null" (as returned by some API fields) to empty string.
func stringOrEmpty(s string) string {
	if s == "null" {
		return ""
	}
	return s
}

// API data structures

// CreatePolicyRequest represents the request body for creating a policy.
type CreatePolicyRequest struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Enforced    bool        `json:"enforced"`
	Explanation string      `json:"explanation"`
	NamespaceID string      `json:"namespaceId"`
	PPL         interface{} `json:"ppl"`
	Remediation string      `json:"remediation"`
}

// UpdatePolicyRequest represents the request body for updating a policy.
type UpdatePolicyRequest struct {
	NamespaceID string      `json:"namespaceId"`
	Name        string      `json:"name"`
	Enforced    bool        `json:"enforced"`
	PPL         interface{} `json:"ppl"`
	Description string      `json:"description"`
	Explanation string      `json:"explanation"`
	Remediation string      `json:"remediation"`
}
