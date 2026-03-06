package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

// --- updateServiceAccountResourceModel ---

func TestUpdateServiceAccountResourceModel(t *testing.T) {
	sa := &ServiceAccount{
		ID:          "sa-123",
		CreatedAt:   "2024-01-01T00:00:00Z",
		UpdatedAt:   "2024-06-01T00:00:00Z",
		ExpiresAt:   "2025-01-01T00:00:00Z",
		Description: "test account",
		UserID:      "user@example.com",
	}

	var model ServiceAccountResourceModel
	updateServiceAccountResourceModel(&model, sa, "mytoken")

	if model.ID.ValueString() != "sa-123" {
		t.Errorf("ID: got %q, want %q", model.ID.ValueString(), "sa-123")
	}
	if model.Description.ValueString() != "test account" {
		t.Errorf("Description: got %q, want %q", model.Description.ValueString(), "test account")
	}
	if model.UserID.ValueString() != "user@example.com" {
		t.Errorf("UserID: got %q, want %q", model.UserID.ValueString(), "user@example.com")
	}
	if model.CreatedAt.ValueString() != "2024-01-01T00:00:00Z" {
		t.Errorf("CreatedAt: got %q, want %q", model.CreatedAt.ValueString(), "2024-01-01T00:00:00Z")
	}
	if model.UpdatedAt.ValueString() != "2024-06-01T00:00:00Z" {
		t.Errorf("UpdatedAt: got %q, want %q", model.UpdatedAt.ValueString(), "2024-06-01T00:00:00Z")
	}
	if model.ExpiresAt.ValueString() != "2025-01-01T00:00:00Z" {
		t.Errorf("ExpiresAt: got %q, want %q", model.ExpiresAt.ValueString(), "2025-01-01T00:00:00Z")
	}
	if model.Token.ValueString() != "mytoken" {
		t.Errorf("Token: got %q, want %q", model.Token.ValueString(), "mytoken")
	}
}

func TestUpdateServiceAccountResourceModel_EmptyExpiresAt(t *testing.T) {
	sa := &ServiceAccount{
		ID:          "sa-456",
		CreatedAt:   "2024-01-01T00:00:00Z",
		UpdatedAt:   "2024-01-01T00:00:00Z",
		ExpiresAt:   "",
		Description: "no expiry",
		UserID:      "user@example.com",
	}

	var model ServiceAccountResourceModel
	updateServiceAccountResourceModel(&model, sa, "")

	if !model.ExpiresAt.IsNull() {
		t.Errorf("ExpiresAt: expected null for empty string, got %q", model.ExpiresAt.ValueString())
	}
}

func TestUpdateServiceAccountResourceModel_TokenPreservedWhenEmpty(t *testing.T) {
	sa := &ServiceAccount{
		ID:          "sa-789",
		CreatedAt:   "2024-01-01T00:00:00Z",
		UpdatedAt:   "2024-01-01T00:00:00Z",
		Description: "existing",
		UserID:      "user@example.com",
	}

	// Pre-populate token in state (as would happen after import or re-read).
	model := ServiceAccountResourceModel{
		Token: types.StringValue("existing-token"),
	}
	// Calling with empty token should NOT overwrite the existing token.
	updateServiceAccountResourceModel(&model, sa, "")

	if model.Token.ValueString() != "existing-token" {
		t.Errorf("Token: expected preserved %q, got %q", "existing-token", model.Token.ValueString())
	}
}

// --- URL helpers ---

func TestServiceAccountURLHelpers(t *testing.T) {
	c := &apiClient{organizationID: "org-1"}

	got := c.serviceAccountsURL("cluster-1")
	want := "https://console.pomerium.app/api/v0/organizations/org-1/clusters/cluster-1/serviceAccounts"
	if got != want {
		t.Errorf("serviceAccountsURL: got %q, want %q", got, want)
	}

	got = c.serviceAccountURL("cluster-1", "sa-1")
	want = "https://console.pomerium.app/api/v0/organizations/org-1/clusters/cluster-1/serviceAccounts/sa-1"
	if got != want {
		t.Errorf("serviceAccountURL: got %q, want %q", got, want)
	}

	got = c.serviceAccountTokenURL("cluster-1", "sa-1")
	want = "https://console.pomerium.app/api/v0/organizations/org-1/clusters/cluster-1/serviceAccounts/sa-1/token"
	if got != want {
		t.Errorf("serviceAccountTokenURL: got %q, want %q", got, want)
	}
}
