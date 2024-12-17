package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure PolicyDataSource satisfies the datasource.DataSource interface.
var _ datasource.DataSource = &PolicyDataSource{}

// NewPolicyDataSource creates a new PolicyDataSource.
func NewPolicyDataSource() datasource.DataSource {
	return &PolicyDataSource{}
}

// PolicyDataSource defines the data source implementation.
type PolicyDataSource struct {
	client         *http.Client
	token          string
	organizationID string
}

// PolicyDataSourceModel describes the data source data model.
type PolicyDataSourceModel struct {
	ID   types.String `tfsdk:"id"`
	Name types.String `tfsdk:"name"`
}

// Metadata sets the data source type name for the PolicyDataSource.
// It appends "_policy" to the data source type name.
func (d *PolicyDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_policy"
}

// Schema defines the structure and attributes of the PolicyDataSource.
func (d *PolicyDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Name of the policy.",
			},
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "ID of the policy.",
			},
		},
	}
}

// Read retrieves the policy ID based on the provided name.
func (d *PolicyDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data PolicyDataSourceModel

	// Read the configuration into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Fetch policies from the API
	policies, err := d.getPolicies(ctx)
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

// Configure sets up the data source with provider-specific data.
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

// Policy represents a Pomerium Zero policy
type Policy struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Enforced    bool            `json:"enforced"`
	Explanation string          `json:"explanation"`
	NamespaceID string          `json:"namespaceId"`
	PPL         json.RawMessage `json:"ppl"`
	Remediation string          `json:"remediation"`
	CreatedAt   string          `json:"createdAt"`
	UpdatedAt   string          `json:"updatedAt"`
	Routes      []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"routes"`
}

// getPolicies fetches all policies from the API.
func (d *PolicyDataSource) getPolicies(ctx context.Context) ([]Policy, error) {
	url := fmt.Sprintf("%s/organizations/%s/policies", apiBaseURL, d.organizationID)

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

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var policies []Policy
	if err := json.NewDecoder(resp.Body).Decode(&policies); err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}

	return policies, nil
}
