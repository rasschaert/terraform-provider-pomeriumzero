package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure PolicyDataSource satisfies the datasource.DataSource interface.
var _ datasource.DataSource = &PolicyDataSource{}

// PolicyDataSource defines the data source implementation.
type PolicyDataSource struct {
	client         *http.Client
	token          string
	organizationID string
}

// PolicyDataSourceModel describes the data source data model.
type PolicyDataSourceModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	NamespaceID types.String `tfsdk:"namespace_id"`
}

func NewPolicyDataSource() datasource.DataSource {
	return &PolicyDataSource{}
}

func (d *PolicyDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_policy"
}

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

func (d *PolicyDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	provider, ok := req.ProviderData.(*pomeriumZeroProvider)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *pomeriumZeroProvider, got: %T", req.ProviderData),
		)
		return
	}

	d.client = provider.client
	d.token = provider.token
	d.organizationID = provider.organizationID
}

func (d *PolicyDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data PolicyDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Pass namespaceID to getPolicies
	policies, err := d.getPolicies(ctx, data.NamespaceID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error fetching policies", err.Error())
		return
	}

	// Find the policy by name
	var foundPolicy *Policy
	for _, policy := range policies {
		if policy.Name == data.Name.ValueString() {
			foundPolicy = &policy
			break
		}
	}

	if foundPolicy == nil {
		resp.Diagnostics.AddError("Policy Not Found", fmt.Sprintf("No policy found with name: %s", data.Name.ValueString()))
		return
	}

	// Set the ID in the data model
	data.ID = types.StringValue(foundPolicy.ID)

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (d *PolicyDataSource) getPolicies(ctx context.Context, namespaceID string) ([]Policy, error) {
	url := fmt.Sprintf("%s/organizations/%s/policies?namespaceId=%s&includeDescendants=true",
		apiBaseURL,
		d.organizationID,
		namespaceID,
	)

	tflog.Debug(ctx, "Getting policies", map[string]interface{}{
		"url": url,
	})

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+d.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := d.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(body))
	}

	var policies []Policy
	if err := json.Unmarshal(body, &policies); err != nil {
		return nil, fmt.Errorf("error parsing response: %w", err)
	}

	tflog.Debug(ctx, "Got policies", map[string]interface{}{
		"count": len(policies),
	})

	return policies, nil
}
