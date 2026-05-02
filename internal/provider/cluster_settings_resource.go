package provider

import (
	"context"
	"errors"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	resource_schema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	resource_schema_planmodifier "github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	resource_schema_boolplanmodifier "github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	resource_schema_float64planmodifier "github.com/hashicorp/terraform-plugin-framework/resource/schema/float64planmodifier"
	resource_schema_stringplanmodifier "github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &ClusterSettingsResource{}
var _ resource.ResourceWithImportState = &ClusterSettingsResource{}

// NewClusterSettingsResource creates a new ClusterSettingsResource.
func NewClusterSettingsResource() resource.Resource {
	return &ClusterSettingsResource{}
}

// ClusterSettingsResource defines the resource implementation.
type ClusterSettingsResource struct {
	client *apiClient
}

// ClusterSettingsResourceModel describes the resource data model.
type ClusterSettingsResourceModel struct {
	ID                           types.String  `tfsdk:"id"`
	ClusterID                    types.String  `tfsdk:"cluster_id"`
	Address                      types.String  `tfsdk:"address"`
	AuthenticateServiceUrl       types.String  `tfsdk:"authenticate_service_url"`
	AutoApplyChangesets          types.Bool    `tfsdk:"auto_apply_changesets"`
	CookieExpire                 types.String  `tfsdk:"cookie_expire"`
	CookieHttpOnly               types.Bool    `tfsdk:"cookie_http_only"`
	CookieName                   types.String  `tfsdk:"cookie_name"`
	DefaultUpstreamTimeout       types.String  `tfsdk:"default_upstream_timeout"`
	DNSLookupFamily              types.String  `tfsdk:"dns_lookup_family"`
	IdentityProvider             types.String  `tfsdk:"identity_provider"`
	IdentityProviderClientId     types.String  `tfsdk:"identity_provider_client_id"`
	IdentityProviderClientSecret types.String  `tfsdk:"identity_provider_client_secret"`
	IdentityProviderUrl          types.String  `tfsdk:"identity_provider_url"`
	LogLevel                     types.String  `tfsdk:"log_level"`
	PassIdentityHeaders          types.Bool    `tfsdk:"pass_identity_headers"`
	ProxyLogLevel                types.String  `tfsdk:"proxy_log_level"`
	SkipXffAppend                types.Bool    `tfsdk:"skip_xff_append"`
	TimeoutIdle                  types.String  `tfsdk:"timeout_idle"`
	TimeoutRead                  types.String  `tfsdk:"timeout_read"`
	TimeoutWrite                 types.String  `tfsdk:"timeout_write"`
	TracingSampleRate            types.Float64 `tfsdk:"tracing_sample_rate"`
	CodecType                    types.String  `tfsdk:"codec_type"`
}

// Metadata sets the resource type name for the ClusterSettingsResource.
func (r *ClusterSettingsResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cluster_settings"
}

// Schema defines the structure and attributes of the ClusterSettingsResource.
func (r *ClusterSettingsResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = resource_schema.Schema{
		MarkdownDescription: "Manages settings for a Pomerium Zero Cluster. This resource allows you to configure various aspects of your cluster, including authentication, timeouts, and logging.",
		Attributes: map[string]resource_schema.Attribute{
			"id": resource_schema.StringAttribute{
				MarkdownDescription: "The unique identifier of the cluster settings. This is set to the cluster ID.",
				Computed:            true,
				PlanModifiers: []resource_schema_planmodifier.String{
					resource_schema_stringplanmodifier.UseStateForUnknown(),
				},
			},
			"cluster_id": resource_schema.StringAttribute{
				MarkdownDescription: "The ID of the cluster these settings apply to. The Pomerium Zero settings endpoints are nested under the cluster (`/clusters/{cluster_id}/settings`), so this must be set to the target cluster's ID — typically `pomeriumzero_cluster.<name>.id`.",
				Required:            true,
				PlanModifiers: []resource_schema_planmodifier.String{
					resource_schema_stringplanmodifier.RequiresReplace(),
				},
			},
			"address": resource_schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "The address of the Pomerium Zero cluster. Typically set to ':443' for HTTPS traffic.",
			},
			"auto_apply_changesets": resource_schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Whether to automatically apply changesets.",
				PlanModifiers: []resource_schema_planmodifier.Bool{
					resource_schema_boolplanmodifier.UseStateForUnknown(),
				},
			},
			"cookie_expire": resource_schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "The expiration time for cookies.",
			},
			"cookie_http_only": resource_schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Whether cookies should be HTTP only.",
				PlanModifiers: []resource_schema_planmodifier.Bool{
					resource_schema_boolplanmodifier.UseStateForUnknown(),
				},
			},
			"cookie_name": resource_schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "The name of the cookie used for authentication.",
			},
			"default_upstream_timeout": resource_schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "The default timeout for upstream requests.",
			},
			"dns_lookup_family": resource_schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "The DNS lookup family to use (e.g., 'v4', 'v6').",
			},
			"identity_provider": resource_schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "The identity provider to use for authentication. If not set, Hosted Authenticate will be used.",
			},
			"identity_provider_client_id": resource_schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "The client ID for the identity provider (required if using custom IDP).",
			},
			"identity_provider_client_secret": resource_schema.StringAttribute{
				Optional:            true,
				Sensitive:           true,
				MarkdownDescription: "The client secret for the identity provider (required if using custom IDP).",
			},
			"identity_provider_url": resource_schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "The URL of the identity provider (required if using custom IDP).",
			},
			"authenticate_service_url": resource_schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "The URL of the authentication service (required if using custom IDP).",
			},
			"log_level": resource_schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "The log level for the Pomerium Zero cluster.",
			},
			"pass_identity_headers": resource_schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Whether to pass identity headers to upstream services.",
				PlanModifiers: []resource_schema_planmodifier.Bool{
					resource_schema_boolplanmodifier.UseStateForUnknown(),
				},
			},
			"proxy_log_level": resource_schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "The log level for the proxy component.",
			},
			"skip_xff_append": resource_schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Whether to skip appending X-Forwarded-For headers.",
				PlanModifiers: []resource_schema_planmodifier.Bool{
					resource_schema_boolplanmodifier.UseStateForUnknown(),
				},
			},
			"timeout_idle": resource_schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "The idle timeout for connections.",
			},
			"timeout_read": resource_schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "The read timeout for connections.",
			},
			"timeout_write": resource_schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "The write timeout for connections.",
			},
			"tracing_sample_rate": resource_schema.Float64Attribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "The sampling rate for tracing.",
				PlanModifiers: []resource_schema_planmodifier.Float64{
					resource_schema_float64planmodifier.UseStateForUnknown(),
				},
			},
			"codec_type": resource_schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "The codec type to use. Valid values are '', 'auto', 'http1', 'http2', 'http3'.",
			},
		},
	}
}

// ValidateConfig checks that identity provider fields are all set or all unset.
func (r *ClusterSettingsResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var data ClusterSettingsResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	idpFieldsSet := !data.IdentityProvider.IsNull() ||
		!data.IdentityProviderClientId.IsNull() ||
		!data.IdentityProviderClientSecret.IsNull() ||
		!data.IdentityProviderUrl.IsNull() ||
		!data.AuthenticateServiceUrl.IsNull()

	if idpFieldsSet {
		if data.IdentityProvider.IsNull() ||
			data.IdentityProviderClientId.IsNull() ||
			data.IdentityProviderClientSecret.IsNull() ||
			data.IdentityProviderUrl.IsNull() ||
			data.AuthenticateServiceUrl.IsNull() {
			resp.Diagnostics.AddError(
				"Invalid Identity Provider Configuration",
				"When configuring a custom identity provider, all related fields (identity_provider, "+
					"identity_provider_client_id, identity_provider_client_secret, identity_provider_url, authenticate_service_url) must be provided.",
			)
		}
	}
}

// Configure prepares the ClusterSettingsResource with provider data.
func (r *ClusterSettingsResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// Create handles the creation of a new ClusterSettingsResource.
//
// The Pomerium Zero API auto-creates a default settings record when a cluster
// is created. There is no POST endpoint for cluster settings — the OpenAPI spec
// defines only GET, PUT, and PATCH on /organizations/{org}/clusters/{cluster}/settings.
// Therefore "create" from Terraform's perspective is really "PUT the desired
// state onto the auto-created record" — the same operation as Update.
func (r *ClusterSettingsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ClusterSettingsResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Normalize ProxyLogLevel (mirrors Update).
	if !plan.ProxyLogLevel.IsNull() && plan.ProxyLogLevel.ValueString() == "" {
		plan.ProxyLogLevel = types.StringNull()
	}

	clusterID := plan.ClusterID.ValueString()
	settingsReq := updateClusterSettingsRequest(plan)
	var settings ClusterSettings
	if err := r.client.put(ctx, r.client.clusterSettingsURL(clusterID), settingsReq, &settings); err != nil {
		resp.Diagnostics.AddError("Error creating cluster settings", err.Error())
		return
	}

	// At Create time, Optional+Computed fields the user didn't set are Unknown
	// in the plan. We must replace them with concrete values before saving state
	// (Terraform fails the apply if any plan value is still Unknown afterward).
	// For fields that are Unknown in the plan we keep what the mapper produced
	// (null, since the API zero values map to null). For fields the user did
	// set, we restore the user's value — the mapper would otherwise clobber
	// false/0 to null, which would mismatch the plan and trip Terraform's
	// "inconsistent result after apply" check.
	resolveBool := func(planVal types.Bool) types.Bool {
		if planVal.IsUnknown() {
			return types.BoolNull()
		}
		return planVal
	}
	resolveFloat := func(planVal types.Float64) types.Float64 {
		if planVal.IsUnknown() {
			return types.Float64Null()
		}
		return planVal
	}
	priorAutoApplyChangesets := resolveBool(plan.AutoApplyChangesets)
	priorCookieHttpOnly := resolveBool(plan.CookieHttpOnly)
	priorPassIdentityHeaders := resolveBool(plan.PassIdentityHeaders)
	priorSkipXffAppend := resolveBool(plan.SkipXffAppend)
	priorTracingSampleRate := resolveFloat(plan.TracingSampleRate)

	updateClusterSettingsResourceModel(&plan, &settings)

	// Use cluster_id as both `id` and `cluster_id` in state (the cluster ID is
	// what every CRUD URL is keyed on).
	plan.ID = types.StringValue(clusterID)
	plan.AutoApplyChangesets = priorAutoApplyChangesets
	plan.CookieHttpOnly = priorCookieHttpOnly
	plan.PassIdentityHeaders = priorPassIdentityHeaders
	plan.SkipXffAppend = priorSkipXffAppend
	plan.TracingSampleRate = priorTracingSampleRate

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

// Read retrieves the current state of the ClusterSettingsResource.
func (r *ClusterSettingsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ClusterSettingsResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	clusterID := state.ID.ValueString()
	if !state.ClusterID.IsNull() && state.ClusterID.ValueString() != "" {
		clusterID = state.ClusterID.ValueString()
	}
	var settings ClusterSettings
	if err := r.client.get(ctx, r.client.clusterSettingsURL(clusterID), &settings); err != nil {
		// See cluster_resource.go Read for the rationale on treating 403 as gone.
		if errors.Is(err, errNotFound) || errors.Is(err, errForbidden) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading cluster settings", err.Error())
		return
	}
	// The API returns a settings-specific ID; we use cluster_id as both `id` and
	// `cluster_id` in state (the cluster ID is what every CRUD URL is keyed on).
	settings.ID = clusterID

	// Save the prior state values for bool/float fields the API doesn't echo
	// back when their value is the zero value (false / 0.0). The mapper converts
	// those to null, which would cause null-vs-false drift on the next plan.
	// We restore these after the mapper runs so the prior state is preserved.
	priorAutoApplyChangesets := state.AutoApplyChangesets
	priorCookieHttpOnly := state.CookieHttpOnly
	priorPassIdentityHeaders := state.PassIdentityHeaders
	priorSkipXffAppend := state.SkipXffAppend
	priorTracingSampleRate := state.TracingSampleRate

	updateClusterSettingsResourceModel(&state, &settings)

	state.ID = types.StringValue(clusterID)
	state.ClusterID = types.StringValue(clusterID)
	state.AutoApplyChangesets = priorAutoApplyChangesets
	state.CookieHttpOnly = priorCookieHttpOnly
	state.PassIdentityHeaders = priorPassIdentityHeaders
	state.SkipXffAppend = priorSkipXffAppend
	state.TracingSampleRate = priorTracingSampleRate

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update handles the update operation for the ClusterSettingsResource.
func (r *ClusterSettingsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ClusterSettingsResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Normalize ProxyLogLevel
	if !plan.ProxyLogLevel.IsNull() && plan.ProxyLogLevel.ValueString() == "" {
		plan.ProxyLogLevel = types.StringNull()
	}

	clusterID := plan.ClusterID.ValueString()
	settingsReq := updateClusterSettingsRequest(plan)
	var settings ClusterSettings
	if err := r.client.put(ctx, r.client.clusterSettingsURL(clusterID), settingsReq, &settings); err != nil {
		resp.Diagnostics.AddError("Error updating cluster settings", err.Error())
		return
	}
	// Preserve the cluster ID (the API may return a different settings ID).
	settings.ID = clusterID

	// Save plan values for bool/float fields the API doesn't echo back when
	// zero (false / 0.0). The mapper converts them to null, which would cause
	// "inconsistent result after apply" since the plan had a known value.
	// Restore the plan values so the post-apply state matches the plan.
	priorAutoApplyChangesets := plan.AutoApplyChangesets
	priorCookieHttpOnly := plan.CookieHttpOnly
	priorPassIdentityHeaders := plan.PassIdentityHeaders
	priorSkipXffAppend := plan.SkipXffAppend
	priorTracingSampleRate := plan.TracingSampleRate

	updateClusterSettingsResourceModel(&plan, &settings)

	plan.ID = types.StringValue(clusterID)
	plan.AutoApplyChangesets = priorAutoApplyChangesets
	plan.CookieHttpOnly = priorCookieHttpOnly
	plan.PassIdentityHeaders = priorPassIdentityHeaders
	plan.SkipXffAppend = priorSkipXffAppend
	plan.TracingSampleRate = priorTracingSampleRate

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

// Delete handles the deletion of a ClusterSettingsResource.
//
// The OpenAPI spec defines no DELETE on /organizations/{org}/clusters/{cluster}/settings.
// Settings are auto-created with the cluster and deleted when the cluster is deleted —
// they have no independent lifecycle. So Delete is a no-op: Terraform removes the
// resource from state automatically when this function returns without error.
func (r *ClusterSettingsResource) Delete(_ context.Context, _ resource.DeleteRequest, _ *resource.DeleteResponse) {
}

// ImportState fetches the current state of the resource from the API.
// The import ID is the cluster ID (the settings resource is keyed by cluster).
func (r *ClusterSettingsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	clusterID := req.ID

	var settings ClusterSettings
	if err := r.client.get(ctx, r.client.clusterSettingsURL(clusterID), &settings); err != nil {
		resp.Diagnostics.AddError("Error importing cluster settings", fmt.Sprintf("Unable to read cluster settings for %s: %s", clusterID, err))
		return
	}

	var state ClusterSettingsResourceModel
	updateClusterSettingsResourceModel(&state, &settings)
	state.ID = types.StringValue(clusterID)
	state.ClusterID = types.StringValue(clusterID)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// updateClusterSettingsResourceModel maps a ClusterSettings API response onto the Terraform model.
// Note: ID is NOT updated from the response — the cluster ID is used as the stable identifier.
//
// All attributes in this resource are Optional-only (no Computed, no Default). The mapper must
// therefore set null when the API returns an empty/zero value, so that configs that omit a field
// (null in Terraform) don't drift against a concrete empty-string or false state.
func updateClusterSettingsResourceModel(model *ClusterSettingsResourceModel, settings *ClusterSettings) {
	// Optional-only string: null when empty, value otherwise.
	optStr := func(val string) types.String {
		if val != "" {
			return types.StringValue(val)
		}
		return types.StringNull()
	}
	// Optional-only bool: null when false (zero value), value otherwise.
	// The API always returns a concrete bool, but if a user omits the field
	// we store null so the next plan doesn't show false -> null drift.
	optBool := func(val bool) types.Bool {
		if val {
			return types.BoolValue(true)
		}
		return types.BoolNull()
	}
	// Optional-only float64: null when zero.
	optFloat := func(val float64) types.Float64 {
		if val != 0 {
			return types.Float64Value(val)
		}
		return types.Float64Null()
	}

	model.Address = optStr(settings.Address)
	model.AutoApplyChangesets = optBool(settings.AutoApplyChangesets)
	model.CookieExpire = optStr(settings.CookieExpire)
	model.CookieHttpOnly = optBool(settings.CookieHttpOnly)
	model.CookieName = optStr(settings.CookieName)
	model.DefaultUpstreamTimeout = optStr(settings.DefaultUpstreamTimeout)
	model.DNSLookupFamily = optStr(settings.DNSLookupFamily)
	model.LogLevel = optStr(settings.LogLevel)
	model.PassIdentityHeaders = optBool(settings.PassIdentityHeaders)
	model.ProxyLogLevel = optStr(settings.ProxyLogLevel)
	model.SkipXffAppend = optBool(settings.SkipXffAppend)
	model.TimeoutIdle = optStr(settings.TimeoutIdle)
	model.TimeoutRead = optStr(settings.TimeoutRead)
	model.TimeoutWrite = optStr(settings.TimeoutWrite)
	model.TracingSampleRate = optFloat(settings.TracingSampleRate)
	model.CodecType = optStr(settings.CodecType)
	model.AuthenticateServiceUrl = optStr(settings.AuthenticateServiceUrl)
	model.IdentityProvider = optStr(settings.IdentityProvider)
	model.IdentityProviderClientId = optStr(settings.IdentityProviderClientId)
	model.IdentityProviderUrl = optStr(settings.IdentityProviderUrl)

	// IdentityProviderClientSecret: pointer in the API struct (sensitive); null when absent.
	if settings.IdentityProviderClientSecret != nil {
		model.IdentityProviderClientSecret = types.StringValue(*settings.IdentityProviderClientSecret)
	} else {
		model.IdentityProviderClientSecret = types.StringNull()
	}
}

// updateClusterSettingsRequest builds an UpdateClusterSettingsRequest from the model,
// omitting nullable fields that are null in state.
func updateClusterSettingsRequest(model ClusterSettingsResourceModel) UpdateClusterSettingsRequest {
	req := UpdateClusterSettingsRequest{
		Address:                model.Address.ValueString(),
		AutoApplyChangesets:    model.AutoApplyChangesets.ValueBool(),
		CookieExpire:           model.CookieExpire.ValueString(),
		CookieHttpOnly:         model.CookieHttpOnly.ValueBool(),
		CookieName:             model.CookieName.ValueString(),
		DefaultUpstreamTimeout: model.DefaultUpstreamTimeout.ValueString(),
		DNSLookupFamily:        model.DNSLookupFamily.ValueString(),
		LogLevel:               model.LogLevel.ValueString(),
		PassIdentityHeaders:    model.PassIdentityHeaders.ValueBool(),
		SkipXffAppend:          model.SkipXffAppend.ValueBool(),
		TimeoutIdle:            model.TimeoutIdle.ValueString(),
		TimeoutRead:            model.TimeoutRead.ValueString(),
		TimeoutWrite:           model.TimeoutWrite.ValueString(),
		TracingSampleRate:      model.TracingSampleRate.ValueFloat64(),
		CodecType:              model.CodecType.ValueString(),
	}

	if !model.AuthenticateServiceUrl.IsNull() {
		req.AuthenticateServiceUrl = model.AuthenticateServiceUrl.ValueString()
	}
	if !model.IdentityProvider.IsNull() {
		req.IdentityProvider = model.IdentityProvider.ValueString()
	}
	if !model.IdentityProviderClientId.IsNull() {
		req.IdentityProviderClientId = model.IdentityProviderClientId.ValueString()
	}
	if !model.IdentityProviderClientSecret.IsNull() {
		v := model.IdentityProviderClientSecret.ValueString()
		req.IdentityProviderClientSecret = &v
	}
	if !model.IdentityProviderUrl.IsNull() {
		req.IdentityProviderUrl = model.IdentityProviderUrl.ValueString()
	}
	if !model.ProxyLogLevel.IsNull() && model.ProxyLogLevel.ValueString() != "" {
		req.ProxyLogLevel = model.ProxyLogLevel.ValueString()
	}

	return req
}

// API data structures

// UpdateClusterSettingsRequest is used to update existing cluster settings.
// Per the OpenAPI spec, this is the body for both PUT (Create + Update) and
// PATCH on /organizations/{org}/clusters/{cluster}/settings.
type UpdateClusterSettingsRequest struct {
	Address                      string  `json:"address,omitempty"`
	AuthenticateServiceUrl       string  `json:"authenticateServiceUrl,omitempty"`
	AutoApplyChangesets          bool    `json:"autoApplyChangesets,omitempty"`
	CookieExpire                 string  `json:"cookieExpire,omitempty"`
	CookieHttpOnly               bool    `json:"cookieHttpOnly,omitempty"`
	CookieName                   string  `json:"cookieName,omitempty"`
	DefaultUpstreamTimeout       string  `json:"defaultUpstreamTimeout,omitempty"`
	DNSLookupFamily              string  `json:"dnsLookupFamily,omitempty"`
	IdentityProvider             string  `json:"identityProvider,omitempty"`
	IdentityProviderClientId     string  `json:"identityProviderClientId,omitempty"`
	IdentityProviderClientSecret *string `json:"identityProviderClientSecret,omitempty"`
	IdentityProviderUrl          string  `json:"identityProviderUrl,omitempty"`
	LogLevel                     string  `json:"logLevel,omitempty"`
	PassIdentityHeaders          bool    `json:"passIdentityHeaders"`
	ProxyLogLevel                string  `json:"proxyLogLevel,omitempty"`
	SkipXffAppend                bool    `json:"skipXffAppend"`
	TimeoutIdle                  string  `json:"timeoutIdle,omitempty"`
	TimeoutRead                  string  `json:"timeoutRead,omitempty"`
	TimeoutWrite                 string  `json:"timeoutWrite,omitempty"`
	TracingSampleRate            float64 `json:"tracingSampleRate,omitempty"`
	CodecType                    string  `json:"codecType"`
}

// ClusterSettings represents the cluster settings data returned by the API.
type ClusterSettings struct {
	ID                           string  `json:"id"`
	Address                      string  `json:"address"`
	AuthenticateServiceUrl       string  `json:"authenticateServiceUrl"`
	AutoApplyChangesets          bool    `json:"autoApplyChangesets"`
	CookieExpire                 string  `json:"cookieExpire"`
	CookieHttpOnly               bool    `json:"cookieHttpOnly"`
	CookieName                   string  `json:"cookieName"`
	DefaultUpstreamTimeout       string  `json:"defaultUpstreamTimeout"`
	DNSLookupFamily              string  `json:"dnsLookupFamily"`
	IdentityProvider             string  `json:"identityProvider"`
	IdentityProviderClientId     string  `json:"identityProviderClientId"`
	IdentityProviderClientSecret *string `json:"identityProviderClientSecret"`
	IdentityProviderUrl          string  `json:"identityProviderUrl"`
	LogLevel                     string  `json:"logLevel"`
	PassIdentityHeaders          bool    `json:"passIdentityHeaders"`
	ProxyLogLevel                string  `json:"proxyLogLevel"`
	SkipXffAppend                bool    `json:"skipXffAppend"`
	TimeoutIdle                  string  `json:"timeoutIdle"`
	TimeoutRead                  string  `json:"timeoutRead"`
	TimeoutWrite                 string  `json:"timeoutWrite"`
	TracingSampleRate            float64 `json:"tracingSampleRate"`
	CodecType                    string  `json:"codecType"`
}
