package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &ClusterDataSource{}

// NewClusterDataSource creates a new ClusterDataSource.
func NewClusterDataSource() datasource.DataSource {
	return &ClusterDataSource{}
}

// ClusterDataSource defines the data source implementation.
type ClusterDataSource struct {
	client *apiClient
}

// ClusterDataSourceModel describes the data source data model.
type ClusterDataSourceModel struct {
	ID                     types.String `tfsdk:"id"`
	Name                   types.String `tfsdk:"name"`
	NamespaceID            types.String `tfsdk:"namespace_id"`
	Domain                 types.String `tfsdk:"domain"`
	FQDN                   types.String `tfsdk:"fqdn"`
	AutoDetectIPAddress    types.String `tfsdk:"auto_detect_ip_address"`
	CreatedAt              types.String `tfsdk:"created_at"`
	UpdatedAt              types.String `tfsdk:"updated_at"`
	Flavor                 types.String `tfsdk:"flavor"`
	HasFailingHealthChecks types.Bool   `tfsdk:"has_failing_health_checks"`
	OnboardingStatus       types.String `tfsdk:"onboarding_status"`
}

// Metadata sets the data source type name for the ClusterDataSource.
func (d *ClusterDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cluster"
}

// Schema defines the structure and attributes of the ClusterDataSource.
func (d *ClusterDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Pomerium Zero Cluster data source",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Cluster identifier",
				Computed:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Cluster name",
				Required:            true,
			},
			"namespace_id": schema.StringAttribute{
				MarkdownDescription: "Cluster namespace ID",
				Computed:            true,
			},
			"domain": schema.StringAttribute{
				MarkdownDescription: "Cluster domain",
				Computed:            true,
			},
			"fqdn": schema.StringAttribute{
				MarkdownDescription: "Cluster FQDN",
				Computed:            true,
			},
			"auto_detect_ip_address": schema.StringAttribute{
				MarkdownDescription: "Auto-detected IP address",
				Computed:            true,
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "Creation timestamp",
				Computed:            true,
			},
			"updated_at": schema.StringAttribute{
				MarkdownDescription: "Last update timestamp",
				Computed:            true,
			},
			"flavor": schema.StringAttribute{
				MarkdownDescription: "The cluster flavor (e.g. `standard`).",
				Computed:            true,
			},
			"has_failing_health_checks": schema.BoolAttribute{
				MarkdownDescription: "Whether the cluster currently has failing health checks.",
				Computed:            true,
			},
			"onboarding_status": schema.StringAttribute{
				MarkdownDescription: "The onboarding status of the cluster (e.g. `in_progress`, `complete`).",
				Computed:            true,
			},
		},
	}
}

// Configure sets up the ClusterDataSource with the provider's configuration.
func (d *ClusterDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*apiClient)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *apiClient, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}
	d.client = client
}

// Read retrieves information about a Pomerium Zero cluster by name.
func (d *ClusterDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ClusterDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var clusters []Cluster
	if err := d.client.get(ctx, d.client.clustersURL(), &clusters); err != nil {
		resp.Diagnostics.AddError("Failed to fetch clusters", err.Error())
		return
	}

	for _, cluster := range clusters {
		if cluster.Name == data.Name.ValueString() {
			data.ID = types.StringValue(cluster.ID)
			data.NamespaceID = types.StringValue(cluster.NamespaceID)
			data.Domain = types.StringValue(cluster.Domain)
			data.FQDN = types.StringValue(cluster.FQDN)
			data.AutoDetectIPAddress = types.StringValue(cluster.AutoDetectIPAddress)
			data.CreatedAt = types.StringValue(cluster.CreatedAt)
			data.UpdatedAt = types.StringValue(cluster.UpdatedAt)
			data.Flavor = types.StringValue(cluster.Flavor)
			data.HasFailingHealthChecks = types.BoolValue(cluster.HasFailingHealthChecks)
			data.OnboardingStatus = types.StringValue(cluster.OnboardingStatus)

			tflog.Trace(ctx, "read a cluster data source")
			resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
			return
		}
	}

	resp.Diagnostics.AddError("Cluster not found", fmt.Sprintf("No cluster found with name: %s", data.Name.ValueString()))
}
