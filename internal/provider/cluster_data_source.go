package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

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
	client         *http.Client
	token          string
	organizationID string
}

// ClusterDataSourceModel describes the data source data model.
type ClusterDataSourceModel struct {
	ID                  types.String `tfsdk:"id"`
	Name                types.String `tfsdk:"name"`
	NamespaceID         types.String `tfsdk:"namespace_id"`
	Domain              types.String `tfsdk:"domain"`
	FQDN                types.String `tfsdk:"fqdn"`
	AutoDetectIPAddress types.String `tfsdk:"auto_detect_ip_address"`
	CreatedAt           types.String `tfsdk:"created_at"`
	UpdatedAt           types.String `tfsdk:"updated_at"`
}

// Metadata sets the data source type name for the ClusterDataSource.
// It appends "_cluster" to the data source type name.
func (d *ClusterDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cluster"
}

// Schema defines the structure and attributes of the ClusterDataSource.
// It specifies the fields that can be used in the Terraform configuration
// to interact with the Pomerium Zero Cluster data source.
func (d *ClusterDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// Provides a description for the data source in Markdown format
		MarkdownDescription: "Pomerium Zero Cluster data source",

		// Defines the attributes of the data source
		Attributes: map[string]schema.Attribute{
			// Cluster identifier, automatically computed
			"id": schema.StringAttribute{
				MarkdownDescription: "Cluster identifier",
				Computed:            true,
			},
			// Cluster name, required input from the user
			"name": schema.StringAttribute{
				MarkdownDescription: "Cluster name",
				Required:            true,
			},
			// Namespace ID of the cluster, automatically computed
			"namespace_id": schema.StringAttribute{
				MarkdownDescription: "Cluster namespace ID",
				Computed:            true,
			},
			// Domain of the cluster, automatically computed
			"domain": schema.StringAttribute{
				MarkdownDescription: "Cluster domain",
				Computed:            true,
			},
			// Fully Qualified Domain Name of the cluster, automatically computed
			"fqdn": schema.StringAttribute{
				MarkdownDescription: "Cluster FQDN",
				Computed:            true,
			},
			// Auto-detected IP address of the cluster, automatically computed
			"auto_detect_ip_address": schema.StringAttribute{
				MarkdownDescription: "Auto-detected IP address",
				Computed:            true,
			},
			// Creation timestamp of the cluster, automatically computed
			"created_at": schema.StringAttribute{
				MarkdownDescription: "Creation timestamp",
				Computed:            true,
			},
			// Last update timestamp of the cluster, automatically computed
			"updated_at": schema.StringAttribute{
				MarkdownDescription: "Last update timestamp",
				Computed:            true,
			},
		},
	}
}

// Configure sets up the ClusterDataSource with the provider's configuration.
// It is called by the Terraform framework to initialize the data source.
func (d *ClusterDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	// Attempt to cast the provider data to the expected type
	provider, ok := req.ProviderData.(*pomeriumZeroProvider)
	if !ok {
		// If the cast fails, add an error to the diagnostics
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *pomeriumZeroProvider, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	// Set the ClusterDataSource fields with the provider's data
	d.client = provider.client
	d.token = provider.token
	d.organizationID = provider.organizationID
}

// Read retrieves information about a Pomerium Zero cluster.
//
// It performs the following steps:
// 1. Reads the Terraform configuration into the data model
// 2. Fetches all clusters from Pomerium Zero
// 3. Finds the cluster matching the provided name
// 4. Maps the cluster data to the data source model
// 5. Saves the data into Terraform state
//
// If any errors occur during this process, it adds them to the response diagnostics.
func (d *ClusterDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ClusterDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Fetch clusters from Pomerium Zero
	clusters, err := d.GetClusters(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Failed to fetch clusters", err.Error())
		return
	}

	// Find the cluster with the matching name
	var matchingCluster *Cluster
	for _, cluster := range clusters {
		if cluster.Name == data.Name.ValueString() {
			matchingCluster = &cluster
			break
		}
	}

	if matchingCluster == nil {
		resp.Diagnostics.AddError("Cluster not found", fmt.Sprintf("No cluster found with name: %s", data.Name.ValueString()))
		return
	}

	// Map the fetched cluster data to our ClusterDataSourceModel
	data.ID = types.StringValue(matchingCluster.ID)
	data.NamespaceID = types.StringValue(matchingCluster.NamespaceID)
	data.Domain = types.StringValue(matchingCluster.Domain)
	data.FQDN = types.StringValue(matchingCluster.FQDN)
	data.AutoDetectIPAddress = types.StringValue(matchingCluster.AutoDetectIPAddress)
	data.CreatedAt = types.StringValue(matchingCluster.CreatedAt)
	data.UpdatedAt = types.StringValue(matchingCluster.UpdatedAt)

	tflog.Trace(ctx, "read a cluster data source")

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// GetClusters fetches all clusters from Pomerium Zero.
func (d *ClusterDataSource) GetClusters(ctx context.Context) ([]Cluster, error) {
	url := fmt.Sprintf("https://console.pomerium.app/api/v0/organizations/%s/clusters", d.organizationID)

	// Create a new HTTP request
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	// Set the request headers
	req.Header.Set("Authorization", "Bearer "+d.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := d.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var clusters []Cluster
	if err := json.NewDecoder(resp.Body).Decode(&clusters); err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}

	return clusters, nil
}
