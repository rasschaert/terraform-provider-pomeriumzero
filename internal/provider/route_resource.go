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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
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
	client *apiClient
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
	TLSDownstreamServerName                   types.String `tfsdk:"tls_downstream_server_name"`
	PolicyIDs                                 types.List   `tfsdk:"policy_ids"`
	Prefix                                    types.String `tfsdk:"prefix"`
	PrefixRewrite                             types.String `tfsdk:"prefix_rewrite"`
	KubernetesServiceAccountToken             types.String `tfsdk:"kubernetes_service_account_token"`
	EnforcedPolicyIDs                         types.List   `tfsdk:"enforced_policy_ids"`
	CreatedAt                                 types.String `tfsdk:"created_at"`
	UpdatedAt                                 types.String `tfsdk:"updated_at"`
	MCP                                       types.String `tfsdk:"mcp"`
}

// Metadata sets the resource type name for the RouteResource.
func (r *RouteResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_route"
}

// Schema defines the structure and attributes of the RouteResource.
func (r *RouteResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description:         "Route resource in Pomerium Zero.",
		MarkdownDescription: "Manages a route resource in Pomerium Zero.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The unique identifier of the route.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The name of the route. Must be unique within the namespace.",
			},
			"namespace_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The ID of the namespace where the route will be created.",
			},
			"from": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The source URL for the route. This is the URL that Pomerium will listen on.",
			},
			"to": schema.ListAttribute{
				ElementType:         types.StringType,
				Required:            true,
				MarkdownDescription: "A list of destination URLs for the route. These are the backend servers that Pomerium will forward requests to.",
			},
			"allow_spdy": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
				MarkdownDescription: "If set to `true`, allows the use of the SPDY protocol for this route.",
			},
			"allow_websockets": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
				MarkdownDescription: "If set to `true`, allows WebSocket connections for this route.",
			},
			"enable_google_cloud_serverless_authentication": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
				MarkdownDescription: "If set to `true`, enables Google Cloud Serverless Authentication for this route.",
			},
			"pass_identity_headers": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "If set to `true`, passes identity headers to the upstream service.",
			},
			"preserve_host_header": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
				MarkdownDescription: "If set to `true`, preserves the original host header when proxying requests.",
			},
			"show_error_details": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
				MarkdownDescription: "If set to `true`, shows detailed error messages when errors occur.",
			},
			"tls_skip_verify": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
				MarkdownDescription: "If set to `true`, skips TLS verification for upstream connections. Use with caution.",
			},
			"tls_upstream_allow_renegotiation": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
				MarkdownDescription: "If set to `true`, allows TLS renegotiation for upstream connections.",
			},
			"tls_downstream_server_name": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "TLS Downstream Server Name overrides the hostname specified in the from field.",
			},
			"policy_ids": schema.ListAttribute{
				ElementType:         types.StringType,
				Optional:            true,
				MarkdownDescription: "A list of policy IDs to associate with this route. These policies will be applied to requests matching this route.",
			},
			"prefix": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "The URL prefix for the route. If specified, only requests with this prefix will be matched.",
			},
			"prefix_rewrite": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "If specified, rewrites the URL prefix before forwarding the request to the upstream service.",
			},
			"kubernetes_service_account_token": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "The Kubernetes service account token to use for authentication.",
				Sensitive:           true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"enforced_policy_ids": schema.ListAttribute{
				ElementType:         types.StringType,
				Computed:            true,
				MarkdownDescription: "List of policy IDs that are enforced on this route at the namespace or organization level (read-only).",
				PlanModifiers: []planmodifier.List{
					listplanmodifier.UseStateForUnknown(),
				},
			},
			"created_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The timestamp when the route was created.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"updated_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The timestamp when the route was last updated.",
				PlanModifiers: []planmodifier.String{
					useStateUnlessUpdating{},
				},
			},
			"mcp": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "MCP (Model Context Protocol) configuration for this route, JSON-encoded. Null when not configured.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

// Configure sets up the RouteResource with the provider's configuration.
func (r *RouteResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// reconcileEmptyStrings fixes up optional string fields where the API treats ""
// and absent identically. When the API omits a field (mapper → null) but the
// reference model (plan or prior state) had an explicit empty string, we keep
// the empty string so Terraform never sees a plan/state inconsistency.
// This lets users write `prefix = ""` without triggering spurious diffs.
func reconcileEmptyStrings(newState *RouteResourceModel, ref *RouteResourceModel) {
	keep := func(mapped, reference types.String) types.String {
		if mapped.IsNull() && !reference.IsNull() && reference.ValueString() == "" {
			return reference
		}
		return mapped
	}
	newState.Prefix = keep(newState.Prefix, ref.Prefix)
	newState.PrefixRewrite = keep(newState.PrefixRewrite, ref.PrefixRewrite)
	newState.TLSDownstreamServerName = keep(newState.TLSDownstreamServerName, ref.TLSDownstreamServerName)
}

// Create handles the creation of a new RouteResource.
func (r *RouteResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan RouteResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var apiResponse map[string]interface{}
	if err := r.client.post(ctx, r.client.routesURL(), createRouteRequest(&plan), http.StatusCreated, &apiResponse); err != nil {
		resp.Diagnostics.AddError("Error creating route", fmt.Sprintf("Could not create route: %s", err))
		return
	}

	newState, err := mapRouteResponseToModel(ctx, apiResponse)
	if err != nil {
		resp.Diagnostics.AddError("Error reading route response", err.Error())
		return
	}
	reconcileEmptyStrings(&newState, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, newState)...)
}

// Read handles the reading of an existing RouteResource.
func (r *RouteResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state RouteResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var apiResponse map[string]interface{}
	if err := r.client.get(ctx, r.client.routeURL(state.ID.ValueString()), &apiResponse); err != nil {
		if errors.Is(err, errNotFound) {
			tflog.Warn(ctx, fmt.Sprintf("Route %s no longer exists in Pomerium Zero", state.ID.ValueString()))
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading route", fmt.Sprintf("Could not read route %s: %s", state.ID.ValueString(), err))
		return
	}

	newState, err := mapRouteResponseToModel(ctx, apiResponse)
	if err != nil {
		resp.Diagnostics.AddError("Error reading route response", err.Error())
		return
	}
	reconcileEmptyStrings(&newState, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, newState)...)
}

// Update handles the update operation for the RouteResource.
func (r *RouteResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan RouteResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var apiResponse map[string]interface{}
	if err := r.client.put(ctx, r.client.routeURL(plan.ID.ValueString()), createRouteRequest(&plan), &apiResponse); err != nil {
		resp.Diagnostics.AddError("Error updating route", fmt.Sprintf("Could not update route: %s", err))
		return
	}

	newState, err := mapRouteResponseToModel(ctx, apiResponse)
	if err != nil {
		resp.Diagnostics.AddError("Error reading route response", err.Error())
		return
	}
	reconcileEmptyStrings(&newState, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, newState)...)
}

// Delete handles the deletion of a RouteResource.
func (r *RouteResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state RouteResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.delete(ctx, r.client.routeURL(state.ID.ValueString())); err != nil {
		resp.Diagnostics.AddError("Error deleting route", fmt.Sprintf("Could not delete route: %s", err))
	}
}

// ImportState handles the importing of an existing RouteResource.
func (r *RouteResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// updateRouteRequest constructs the API request payload for updating a route.
// Update and create use the same payload shape.
func updateRouteRequest(model *RouteResourceModel) map[string]interface{} {
	return createRouteRequest(model)
}

// createRouteRequest constructs the API request payload for creating or updating a route.
func createRouteRequest(model *RouteResourceModel) map[string]interface{} {
	req := map[string]interface{}{
		"name":        model.Name.ValueString(),
		"namespaceId": model.NamespaceID.ValueString(),
		"from":        model.From.ValueString(),
		"allowSpdy":   model.AllowSpdy.ValueBool(),
		"enableGoogleCloudServerlessAuthentication": model.EnableGoogleCloudServerlessAuthentication.ValueBool(),
		"showErrorDetails":              model.ShowErrorDetails.ValueBool(),
		"tlsSkipVerify":                 model.TLSSkipVerify.ValueBool(),
		"tlsUpstreamAllowRenegotiation": model.TLSUpstreamAllowRenegotiation.ValueBool(),
	}

	if !model.To.IsNull() {
		var to []string
		model.To.ElementsAs(context.Background(), &to, false)
		req["to"] = to
	}
	if !model.AllowWebsockets.IsNull() {
		req["allowWebsockets"] = model.AllowWebsockets.ValueBool()
	}
	if !model.PassIdentityHeaders.IsNull() {
		req["passIdentityHeaders"] = model.PassIdentityHeaders.ValueBool()
	}
	if !model.PreserveHostHeader.IsNull() {
		req["preserveHostHeader"] = model.PreserveHostHeader.ValueBool()
	}
	if !model.PolicyIDs.IsNull() {
		var policyIDs []string
		model.PolicyIDs.ElementsAs(context.Background(), &policyIDs, false)
		req["policyIds"] = policyIDs
	}
	if !model.Prefix.IsNull() && model.Prefix.ValueString() != "" {
		req["prefix"] = model.Prefix.ValueString()
	}
	if !model.PrefixRewrite.IsNull() && model.PrefixRewrite.ValueString() != "" {
		req["prefixRewrite"] = model.PrefixRewrite.ValueString()
	}
	if !model.KubernetesServiceAccountToken.IsNull() {
		req["kubernetesServiceAccountToken"] = model.KubernetesServiceAccountToken.ValueString()
	}
	if !model.TLSDownstreamServerName.IsNull() {
		req["tlsDownstreamServerName"] = model.TLSDownstreamServerName.ValueString()
	}

	return req
}

// mapRouteResponseToModel converts the API response map to a RouteResourceModel.
// Returns an error if required fields are missing or have unexpected types.
func mapRouteResponseToModel(ctx context.Context, apiResponse map[string]interface{}) (RouteResourceModel, error) {
	getString := func(key string) (string, error) {
		v, ok := apiResponse[key]
		if !ok {
			return "", fmt.Errorf("route response missing required field %q", key)
		}
		s, ok := v.(string)
		if !ok {
			return "", fmt.Errorf("route response field %q has unexpected type %T", key, v)
		}
		return s, nil
	}

	id, err := getString("id")
	if err != nil {
		return RouteResourceModel{}, err
	}
	name, err := getString("name")
	if err != nil {
		return RouteResourceModel{}, err
	}
	namespaceID, err := getString("namespaceId")
	if err != nil {
		return RouteResourceModel{}, err
	}
	from, err := getString("from")
	if err != nil {
		return RouteResourceModel{}, err
	}

	model := RouteResourceModel{
		ID:          types.StringValue(id),
		Name:        types.StringValue(name),
		NamespaceID: types.StringValue(namespaceID),
		From:        types.StringValue(from),
	}

	// Handle the 'to' field
	if to, ok := apiResponse["to"].([]interface{}); ok {
		toList, _ := types.ListValueFrom(ctx, types.StringType, to)
		model.To = toList
	} else {
		model.To, _ = types.ListValueFrom(ctx, types.StringType, []string{})
	}

	// toBoolDefault extracts a bool from the API response, falling back to a
	// provided default when the field is absent or non-bool. Use this for
	// Optional+Computed+Default attributes so that the state value always
	// matches the schema default rather than going null on unexpected responses.
	toBoolDefault := func(v interface{}, def bool) types.Bool {
		if b, ok := v.(bool); ok {
			return types.BoolValue(b)
		}
		return types.BoolValue(def)
	}
	// toBoolNullable extracts a bool from the API response and returns null
	// when the field is absent. Use this for Optional+Computed attributes
	// that have no schema-level Default (null is a valid/meaningful state).
	toBoolNullable := func(v interface{}) types.Bool {
		if b, ok := v.(bool); ok {
			return types.BoolValue(b)
		}
		return types.BoolNull()
	}

	model.AllowSpdy = toBoolDefault(apiResponse["allowSpdy"], false)
	model.AllowWebsockets = toBoolNullable(apiResponse["allowWebsockets"])
	model.EnableGoogleCloudServerlessAuthentication = toBoolDefault(apiResponse["enableGoogleCloudServerlessAuthentication"], false)
	model.PassIdentityHeaders = toBoolNullable(apiResponse["passIdentityHeaders"])
	model.PreserveHostHeader = toBoolDefault(apiResponse["preserveHostHeader"], false)
	model.ShowErrorDetails = toBoolDefault(apiResponse["showErrorDetails"], true)
	model.TLSSkipVerify = toBoolDefault(apiResponse["tlsSkipVerify"], false)
	model.TLSUpstreamAllowRenegotiation = toBoolDefault(apiResponse["tlsUpstreamAllowRenegotiation"], false)

	// Handle policyIds — API may return either a flat string array or an array of objects
	if policyIDs, ok := apiResponse["policyIds"].([]interface{}); ok {
		policyIDsList, _ := types.ListValueFrom(ctx, types.StringType, policyIDs)
		model.PolicyIDs = policyIDsList
	} else if policies, ok := apiResponse["policies"].([]interface{}); ok {
		var ids []string
		for _, p := range policies {
			if obj, ok := p.(map[string]interface{}); ok {
				if id, ok := obj["id"].(string); ok {
					ids = append(ids, id)
				}
			}
		}
		if len(ids) > 0 {
			model.PolicyIDs, _ = types.ListValueFrom(ctx, types.StringType, ids)
		}
	}

	// Optional-only string fields: use StringValue when the API returns a
	// non-empty string, otherwise StringNull so that unset configs (null)
	// don't drift against an empty-string state.
	toOptionalString := func(key string) types.String {
		if v, ok := apiResponse[key].(string); ok && v != "" {
			return types.StringValue(v)
		}
		return types.StringNull()
	}
	model.Prefix = toOptionalString("prefix")
	model.PrefixRewrite = toOptionalString("prefixRewrite")
	model.TLSDownstreamServerName = toOptionalString("tlsDownstreamServerName")
	if v, ok := apiResponse["kubernetesServiceAccountToken"].(string); ok && v != "" {
		model.KubernetesServiceAccountToken = types.StringValue(v)
	}

	// enforced_policy_ids: read-only, set by namespace/org policies
	if ids, ok := apiResponse["enforcedPolicyIds"].([]interface{}); ok {
		model.EnforcedPolicyIDs, _ = types.ListValueFrom(ctx, types.StringType, ids)
	} else {
		model.EnforcedPolicyIDs, _ = types.ListValueFrom(ctx, types.StringType, []string{})
	}

	model.CreatedAt = toOptionalString("createdAt")
	model.UpdatedAt = toOptionalString("updatedAt")

	// mcp: JSON-encode non-null values; store null when absent or null
	if mcpVal := apiResponse["mcp"]; mcpVal != nil {
		if mcpJSON, err := json.Marshal(mcpVal); err == nil {
			model.MCP = types.StringValue(string(mcpJSON))
		} else {
			model.MCP = types.StringNull()
		}
	} else {
		model.MCP = types.StringNull()
	}

	return model, nil
}
