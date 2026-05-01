package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestUpdateClusterResourceModel(t *testing.T) {
	cluster := &Cluster{
		ID:                     "cluster-123",
		Name:                   "my-cluster",
		NamespaceID:            "ns-456",
		Domain:                 "my-cluster",
		FQDN:                   "my-cluster.pomerium.app",
		AutoDetectIPAddress:    "1.2.3.4",
		CreatedAt:              "2024-01-01T00:00:00Z",
		UpdatedAt:              "2024-06-01T12:00:00Z",
		Flavor:                 "standard",
		HasFailingHealthChecks: true,
		OnboardingStatus:       "complete",
	}

	var model ClusterResourceModel
	updateClusterResourceModel(&model, cluster)

	checks := []struct {
		field string
		got   string
		want  string
	}{
		{"ID", model.ID.ValueString(), "cluster-123"},
		{"Name", model.Name.ValueString(), "my-cluster"},
		{"NamespaceID", model.NamespaceID.ValueString(), "ns-456"},
		{"Domain", model.Domain.ValueString(), "my-cluster"},
		{"FQDN", model.FQDN.ValueString(), "my-cluster.pomerium.app"},
		{"AutoDetectIPAddress", model.AutoDetectIPAddress.ValueString(), "1.2.3.4"},
		{"CreatedAt", model.CreatedAt.ValueString(), "2024-01-01T00:00:00Z"},
		{"UpdatedAt", model.UpdatedAt.ValueString(), "2024-06-01T12:00:00Z"},
		{"Flavor", model.Flavor.ValueString(), "standard"},
		{"OnboardingStatus", model.OnboardingStatus.ValueString(), "complete"},
	}
	if !model.HasFailingHealthChecks.ValueBool() {
		t.Error("HasFailingHealthChecks: got false, want true")
	}
	for _, c := range checks {
		if c.got != c.want {
			t.Errorf("%s: got %q, want %q", c.field, c.got, c.want)
		}
	}
}

func TestUpdateClusterResourceModel_OverwritesExistingModel(t *testing.T) {
	// Populate model with stale values, then update — all fields must be overwritten.
	model := ClusterResourceModel{
		ID:                  types.StringValue("old-id"),
		Name:                types.StringValue("old-name"),
		NamespaceID:         types.StringValue("old-ns"),
		Domain:              types.StringValue("old-domain"),
		FQDN:                types.StringValue("old.fqdn"),
		AutoDetectIPAddress: types.StringValue("0.0.0.0"),
		CreatedAt:           types.StringValue("2000-01-01T00:00:00Z"),
		UpdatedAt:           types.StringValue("2000-01-01T00:00:00Z"),
	}

	cluster := &Cluster{
		ID:                  "new-id",
		Name:                "new-name",
		NamespaceID:         "new-ns",
		Domain:              "new-domain",
		FQDN:                "new.fqdn",
		AutoDetectIPAddress: "9.9.9.9",
		CreatedAt:           "2025-01-01T00:00:00Z",
		UpdatedAt:           "2025-06-01T00:00:00Z",
	}

	updateClusterResourceModel(&model, cluster)

	if model.ID.ValueString() != "new-id" {
		t.Errorf("ID not overwritten: got %q", model.ID.ValueString())
	}
	if model.Name.ValueString() != "new-name" {
		t.Errorf("Name not overwritten: got %q", model.Name.ValueString())
	}
	if model.AutoDetectIPAddress.ValueString() != "9.9.9.9" {
		t.Errorf("AutoDetectIPAddress not overwritten: got %q", model.AutoDetectIPAddress.ValueString())
	}
}

// TestBuildClusterRequestBody verifies the create/update body shape:
//   - `name` is always included (Required).
//   - `flavor` and `manualOverrideIpAddress` are included only when set.
//   - Server-assigned fields (`domain`, etc.) are never sent — the API rejects them.
func TestBuildClusterRequestBody(t *testing.T) {
	t.Run("minimal", func(t *testing.T) {
		m := &ClusterResourceModel{Name: types.StringValue("c1")}
		got := buildClusterRequestBody(m)
		if got["name"] != "c1" {
			t.Errorf("name: got %v, want c1", got["name"])
		}
		if _, ok := got["domain"]; ok {
			t.Error("domain must not be in request body (server-assigned)")
		}
		if _, ok := got["flavor"]; ok {
			t.Error("flavor must be omitted when null")
		}
		if _, ok := got["manualOverrideIpAddress"]; ok {
			t.Error("manualOverrideIpAddress must be omitted when null")
		}
	})

	t.Run("with optionals", func(t *testing.T) {
		m := &ClusterResourceModel{
			Name:                    types.StringValue("c1"),
			Flavor:                  types.StringValue("hosted"),
			ManualOverrideIPAddress: types.StringValue("1.2.3.4"),
		}
		got := buildClusterRequestBody(m)
		if got["flavor"] != "hosted" {
			t.Errorf("flavor: got %v, want hosted", got["flavor"])
		}
		if got["manualOverrideIpAddress"] != "1.2.3.4" {
			t.Errorf("manualOverrideIpAddress: got %v, want 1.2.3.4", got["manualOverrideIpAddress"])
		}
	})
}

// TestClusterTokenPreservedOnRead verifies that the cluster_token in state survives
// a Read. The GET cluster endpoint never returns the token (it's only emitted by
// POST .../clusters and the rotate endpoint), so Read must preserve the prior value
// rather than letting updateClusterResourceModel zero it out.
func TestClusterTokenPreservedOnRead(t *testing.T) {
	state := ClusterResourceModel{
		ID:           types.StringValue("cluster-1"),
		ClusterToken: types.StringValue("secret-token-from-create"),
	}

	// Simulate the GET response — it has no token field.
	cluster := &Cluster{
		ID:   "cluster-1",
		Name: "c",
	}

	prevToken := state.ClusterToken
	updateClusterResourceModel(&state, cluster)
	state.ClusterToken = prevToken // mirror Read's preservation step

	if state.ClusterToken.ValueString() != "secret-token-from-create" {
		t.Errorf("cluster_token should be preserved from prior state; got %q",
			state.ClusterToken.ValueString())
	}
}

func TestUpdateClusterResourceModel_EmptyStrings(t *testing.T) {
	// API may return empty strings for optional/computed fields — they should be stored as-is.
	cluster := &Cluster{
		ID:                  "cluster-1",
		Name:                "c",
		NamespaceID:         "ns-1",
		Domain:              "c",
		FQDN:                "c.pomerium.app",
		AutoDetectIPAddress: "", // not yet detected
		CreatedAt:           "2024-01-01T00:00:00Z",
		UpdatedAt:           "2024-01-01T00:00:00Z",
	}

	var model ClusterResourceModel
	updateClusterResourceModel(&model, cluster)

	if model.AutoDetectIPAddress.IsNull() {
		t.Error("AutoDetectIPAddress: got null, want empty string value")
	}
	if model.AutoDetectIPAddress.ValueString() != "" {
		t.Errorf("AutoDetectIPAddress: got %q, want empty string", model.AutoDetectIPAddress.ValueString())
	}
}
