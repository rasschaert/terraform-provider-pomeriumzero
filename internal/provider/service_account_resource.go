package provider

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &ServiceAccountResource{}
var _ resource.ResourceWithImportState = &ServiceAccountResource{}

// NewServiceAccountResource creates a new ServiceAccountResource.
func NewServiceAccountResource() resource.Resource {
	return &ServiceAccountResource{}
}

// ServiceAccountResource defines the resource implementation.
type ServiceAccountResource struct {
	client *apiClient
}

// ServiceAccountResourceModel describes the resource data model.
type ServiceAccountResourceModel struct {
	ID          types.String `tfsdk:"id"`
	ClusterID   types.String `tfsdk:"cluster_id"`
	Description types.String `tfsdk:"description"`
	UserID      types.String `tfsdk:"user_id"`
	ExpiresAt   types.String `tfsdk:"expires_at"`
	CreatedAt   types.String `tfsdk:"created_at"`
	UpdatedAt   types.String `tfsdk:"updated_at"`
	Token       types.String `tfsdk:"token"`
}

// Metadata sets the resource type name for the ServiceAccountResource.
func (r *ServiceAccountResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_service_account"
}

// Schema defines the structure and attributes of the ServiceAccountResource.
func (r *ServiceAccountResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Pomerium Zero Service Account. Service accounts are cluster-scoped and cannot be updated in place; any change to a non-computed field will recreate the resource.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The unique identifier of the service account.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"cluster_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The ID of the cluster this service account belongs to.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"description": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "A human-readable description for the service account.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"user_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The user ID to associate with this service account.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"expires_at": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "The expiration time of the service account (RFC3339 format). If omitted the service account does not expire.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"created_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The time the service account was created.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"updated_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The time the service account was last updated.",
			},
			"token": schema.StringAttribute{
				Computed:            true,
				Sensitive:           true,
				MarkdownDescription: "The bearer token for this service account. Only available after creation; use `nonsensitive(resource.token)` to print it.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

// Configure prepares the ServiceAccountResource with provider data.
func (r *ServiceAccountResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// Create creates a new service account in Pomerium Zero.
func (r *ServiceAccountResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ServiceAccountResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := CreateServiceAccountRequest{
		Description: plan.Description.ValueString(),
		UserID:      plan.UserID.ValueString(),
	}
	if !plan.ExpiresAt.IsNull() && !plan.ExpiresAt.IsUnknown() {
		v := plan.ExpiresAt.ValueString()
		body.ExpiresAt = &v
	}

	var sa ServiceAccountCreateResponse
	if err := r.client.post(ctx, r.client.serviceAccountsURL(plan.ClusterID.ValueString()), body, http.StatusCreated, &sa); err != nil {
		resp.Diagnostics.AddError("Error creating service account", err.Error())
		return
	}

	updateServiceAccountResourceModel(&plan, &sa.ServiceAccount, sa.Token)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

// Read retrieves the current state of a ServiceAccountResource from the API.
func (r *ServiceAccountResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ServiceAccountResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var sa ServiceAccount
	if err := r.client.get(ctx, r.client.serviceAccountURL(state.ClusterID.ValueString(), state.ID.ValueString()), &sa); err != nil {
		// See cluster_resource.go Read for the rationale on treating 403 as gone.
		if errors.Is(err, errNotFound) || errors.Is(err, errForbidden) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading service account", err.Error())
		return
	}

	// Preserve the token from state — the read endpoint does not return it.
	updateServiceAccountResourceModel(&state, &sa, state.Token.ValueString())
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

// Update is not supported; all mutable fields use RequiresReplace.
func (r *ServiceAccountResource) Update(_ context.Context, _ resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Update not supported", "Service accounts cannot be updated in place. This is a bug in the provider.")
}

// Delete handles the deletion of a ServiceAccountResource.
func (r *ServiceAccountResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ServiceAccountResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.delete(ctx, r.client.serviceAccountURL(state.ClusterID.ValueString(), state.ID.ValueString())); err != nil {
		resp.Diagnostics.AddError("Error deleting service account", err.Error())
	}
}

// ImportState imports an existing service account by "clusterID/serviceAccountID".
func (r *ServiceAccountResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import ID format: "<cluster_id>/<service_account_id>"
	var clusterID, serviceAccountID string
	if n, _ := fmt.Sscanf(req.ID, "%s", &clusterID); n != 1 {
		resp.Diagnostics.AddError(
			"Invalid import ID",
			`Expected format: "<cluster_id>/<service_account_id>"`,
		)
		return
	}
	// Split manually since Sscanf on a single %s won't handle slashes.
	for i, c := range req.ID {
		if c == '/' {
			clusterID = req.ID[:i]
			serviceAccountID = req.ID[i+1:]
			break
		}
	}
	if clusterID == "" || serviceAccountID == "" {
		resp.Diagnostics.AddError(
			"Invalid import ID",
			`Expected format: "<cluster_id>/<service_account_id>"`,
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("cluster_id"), clusterID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), serviceAccountID)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var sa ServiceAccount
	if err := r.client.get(ctx, r.client.serviceAccountURL(clusterID, serviceAccountID), &sa); err != nil {
		resp.Diagnostics.AddError("Error importing service account", fmt.Sprintf("Unable to read service account %s: %s", serviceAccountID, err))
		return
	}

	// Token is not available on import; it must be retrieved separately if needed.
	var state ServiceAccountResourceModel
	state.ClusterID = types.StringValue(clusterID)
	updateServiceAccountResourceModel(&state, &sa, "")
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// updateServiceAccountResourceModel populates a ServiceAccountResourceModel from API data.
// token should be the bearer token (only available from create response or token endpoint).
func updateServiceAccountResourceModel(model *ServiceAccountResourceModel, sa *ServiceAccount, token string) {
	model.ID = types.StringValue(sa.ID)
	model.Description = types.StringValue(sa.Description)
	model.UserID = types.StringValue(sa.UserID)
	model.CreatedAt = types.StringValue(sa.CreatedAt)
	model.UpdatedAt = types.StringValue(sa.UpdatedAt)
	if sa.ExpiresAt != "" {
		model.ExpiresAt = types.StringValue(sa.ExpiresAt)
	} else {
		model.ExpiresAt = types.StringNull()
	}
	if token != "" {
		model.Token = types.StringValue(token)
	}
}

// API data structures

// ServiceAccount represents a service account returned by the API.
type ServiceAccount struct {
	ID          string `json:"id"`
	CreatedAt   string `json:"createdAt"`
	UpdatedAt   string `json:"updatedAt"`
	ExpiresAt   string `json:"expiresAt"`
	Description string `json:"description"`
	UserID      string `json:"userId"`
}

// ServiceAccountCreateResponse is the response body from POST .../serviceAccounts.
// It embeds ServiceAccount and adds a one-time Token field.
type ServiceAccountCreateResponse struct {
	ServiceAccount
	Token string `json:"token"`
}

// CreateServiceAccountRequest is the request body for creating a service account.
type CreateServiceAccountRequest struct {
	ExpiresAt   *string `json:"expiresAt,omitempty"`
	Description string  `json:"description"`
	UserID      string  `json:"userId"`
}
