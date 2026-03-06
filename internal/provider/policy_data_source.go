package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure PolicyDataSource satisfies the datasource.DataSource interface.
var _ datasource.DataSource = &PolicyDataSource{}

// PolicyDataSource defines the data source implementation.
type PolicyDataSource struct {
	client *apiClient
}

// PolicyDataSourceModel describes the data source data model.
type PolicyDataSourceModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	NamespaceID types.String `tfsdk:"namespace_id"`
}

// NewPolicyDataSource creates a new PolicyDataSource.
func NewPolicyDataSource() datasource.DataSource {
	return &PolicyDataSource{}
}

// Metadata sets the data source type name for the PolicyDataSource.
func (d *PolicyDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_policy"
}

// Schema defines the structure and attributes of the PolicyDataSource.
func (d *PolicyDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetch a policy by name.",
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Name of the policy to look up",
			},
			"namespace_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "ID of the namespace the policy belongs to",
			},
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "ID of the policy",
			},
		},
	}
}

// Configure sets up the PolicyDataSource with the provider's configuration.
func (d *PolicyDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*apiClient)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *apiClient, got: %T", req.ProviderData),
		)
		return
	}
	d.client = client
}

// Read retrieves a policy by name within the given namespace.
func (d *PolicyDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data PolicyDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	url := fmt.Sprintf("%s?namespaceId=%s&includeDescendants=true", d.client.policiesURL(), data.NamespaceID.ValueString())
	tflog.Debug(ctx, "Getting policies", map[string]interface{}{"url": url})

	var policies []Policy
	if err := d.client.get(ctx, url, &policies); err != nil {
		resp.Diagnostics.AddError("Error fetching policies", err.Error())
		return
	}

	tflog.Debug(ctx, "Got policies", map[string]interface{}{"count": len(policies)})

	for _, policy := range policies {
		if policy.Name == data.Name.ValueString() {
			data.ID = types.StringValue(policy.ID)
			resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
			return
		}
	}

	resp.Diagnostics.AddError("Policy Not Found", fmt.Sprintf("No policy found with name: %s", data.Name.ValueString()))
}
