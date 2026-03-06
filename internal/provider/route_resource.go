package provider

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
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
	if !model.Prefix.IsNull() {
		req["prefix"] = model.Prefix.ValueString()
	}
	if !model.PrefixRewrite.IsNull() {
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

	// Helper to safely convert interface{} to types.Bool
	toBool := func(v interface{}) types.Bool {
		if v == nil {
			return types.BoolNull()
		}
		if b, ok := v.(bool); ok {
			return types.BoolValue(b)
		}
		return types.BoolNull()
	}

	model.AllowSpdy = toBool(apiResponse["allowSpdy"])
	model.AllowWebsockets = toBool(apiResponse["allowWebsockets"])
	model.EnableGoogleCloudServerlessAuthentication = toBool(apiResponse["enableGoogleCloudServerlessAuthentication"])
	model.PassIdentityHeaders = toBool(apiResponse["passIdentityHeaders"])
	model.PreserveHostHeader = toBool(apiResponse["preserveHostHeader"])
	model.ShowErrorDetails = toBool(apiResponse["showErrorDetails"])
	model.TLSSkipVerify = toBool(apiResponse["tlsSkipVerify"])
	model.TLSUpstreamAllowRenegotiation = toBool(apiResponse["tlsUpstreamAllowRenegotiation"])

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

	// Optional string fields
	if v, ok := apiResponse["prefix"].(string); ok {
		model.Prefix = types.StringValue(v)
	}
	if v, ok := apiResponse["prefixRewrite"].(string); ok {
		model.PrefixRewrite = types.StringValue(v)
	}
	if v, ok := apiResponse["kubernetesServiceAccountToken"].(string); ok {
		model.KubernetesServiceAccountToken = types.StringValue(v)
	}
	if v, ok := apiResponse["tlsDownstreamServerName"].(string); ok {
		model.TLSDownstreamServerName = types.StringValue(v)
	}

	return model, nil
}
