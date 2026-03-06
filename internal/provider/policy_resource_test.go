package provider

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

// --- stringOrEmpty ---

func TestStringOrEmpty(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"hello", "hello"},
		{"", ""},
		{"null", ""},     // literal string "null" is treated as empty
		{"NULL", "NULL"}, // case-sensitive: not treated as null
		{"  ", "  "},
	}
	for _, tt := range tests {
		got := stringOrEmpty(tt.input)
		if got != tt.want {
			t.Errorf("stringOrEmpty(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

// --- updatePolicyResourceModel ---

func TestUpdatePolicyResourceModel(t *testing.T) {
	ppl := json.RawMessage(`{"allow":{"and":[{"email":{"is":"user@example.com"}}]}}`)
	policy := &Policy{
		ID:          "pol-123",
		Name:        "Test Policy",
		Description: "A test policy",
		Enforced:    true,
		Explanation: "For testing",
		NamespaceID: "ns-456",
		PPL:         ppl,
		Remediation: "Contact admin",
	}

	var model PolicyResourceModel
	updatePolicyResourceModel(&model, policy)

	if model.ID.ValueString() != "pol-123" {
		t.Errorf("ID: got %q, want %q", model.ID.ValueString(), "pol-123")
	}
	if model.Name.ValueString() != "Test Policy" {
		t.Errorf("Name: got %q, want %q", model.Name.ValueString(), "Test Policy")
	}
	if model.Description.ValueString() != "A test policy" {
		t.Errorf("Description: got %q, want %q", model.Description.ValueString(), "A test policy")
	}
	if !model.Enforced.ValueBool() {
		t.Error("Enforced: got false, want true")
	}
	if model.Explanation.ValueString() != "For testing" {
		t.Errorf("Explanation: got %q, want %q", model.Explanation.ValueString(), "For testing")
	}
	if model.NamespaceID.ValueString() != "ns-456" {
		t.Errorf("NamespaceID: got %q, want %q", model.NamespaceID.ValueString(), "ns-456")
	}
	if model.Remediation.ValueString() != "Contact admin" {
		t.Errorf("Remediation: got %q, want %q", model.Remediation.ValueString(), "Contact admin")
	}
	// PPL is stored as the raw JSON string
	if model.PPL.ValueString() != string(ppl) {
		t.Errorf("PPL: got %q, want %q", model.PPL.ValueString(), string(ppl))
	}
}

func TestUpdatePolicyResourceModel_NullStringFieldsViaStringOrEmpty(t *testing.T) {
	// The API may return "null" as a string for unset fields.
	// stringOrEmpty converts the literal "null" to "".
	policy := &Policy{
		ID:          "pol-1",
		Name:        "null",
		Description: "null",
		Explanation: "null",
		Remediation: "null",
		NamespaceID: "ns-1",
		PPL:         json.RawMessage(`{}`),
	}

	var model PolicyResourceModel
	updatePolicyResourceModel(&model, policy)

	for field, got := range map[string]string{
		"Name":        model.Name.ValueString(),
		"Description": model.Description.ValueString(),
		"Explanation": model.Explanation.ValueString(),
		"Remediation": model.Remediation.ValueString(),
	} {
		if got != "" {
			t.Errorf("%s: got %q, want empty string (stringOrEmpty should convert literal null)", field, got)
		}
	}
}

// --- createPolicyRequest ---

func TestCreatePolicyRequest(t *testing.T) {
	ctx := context.Background()
	model := PolicyResourceModel{
		Name:        types.StringValue("My Policy"),
		Description: types.StringValue("desc"),
		Enforced:    types.BoolValue(true),
		Explanation: types.StringValue("explanation"),
		NamespaceID: types.StringValue("ns-1"),
		PPL:         types.StringValue(`{"allow":{"and":[{"email":{"is":"user@example.com"}}]}}`),
		Remediation: types.StringValue("fix it"),
	}

	req := createPolicyRequest(ctx, model)

	if req.Name != "My Policy" {
		t.Errorf("Name: got %q, want %q", req.Name, "My Policy")
	}
	if req.Description != "desc" {
		t.Errorf("Description: got %q, want %q", req.Description, "desc")
	}
	if !req.Enforced {
		t.Error("Enforced: got false, want true")
	}
	if req.NamespaceID != "ns-1" {
		t.Errorf("NamespaceID: got %q, want %q", req.NamespaceID, "ns-1")
	}
	if req.PPL == nil {
		t.Error("PPL: got nil, want parsed JSON object")
	}
}

func TestCreatePolicyRequest_InvalidPPLIsNil(t *testing.T) {
	ctx := context.Background()
	// Invalid JSON in PPL results in PPL being nil (not panicking).
	model := PolicyResourceModel{
		Name:        types.StringValue("p"),
		Description: types.StringValue(""),
		Enforced:    types.BoolValue(false),
		Explanation: types.StringValue(""),
		NamespaceID: types.StringValue("ns"),
		PPL:         types.StringValue(`not valid json`),
		Remediation: types.StringValue(""),
	}

	req := createPolicyRequest(ctx, model)

	if req.PPL != nil {
		t.Errorf("PPL: expected nil for invalid JSON, got %v", req.PPL)
	}
}

// --- updatePolicyRequest ---

func TestUpdatePolicyRequest(t *testing.T) {
	ctx := context.Background()
	model := PolicyResourceModel{
		ID:          types.StringValue("pol-1"),
		Name:        types.StringValue("Updated Policy"),
		Description: types.StringValue("new desc"),
		Enforced:    types.BoolValue(false),
		Explanation: types.StringValue("new explanation"),
		NamespaceID: types.StringValue("ns-2"),
		PPL:         types.StringValue(`{"deny":{"or":[{"groups":{"has":"admin"}}]}}`),
		Remediation: types.StringValue("new fix"),
	}

	req := updatePolicyRequest(ctx, model)

	if req.Name != "Updated Policy" {
		t.Errorf("Name: got %q, want %q", req.Name, "Updated Policy")
	}
	if req.NamespaceID != "ns-2" {
		t.Errorf("NamespaceID: got %q, want %q", req.NamespaceID, "ns-2")
	}
	if req.Enforced {
		t.Error("Enforced: got true, want false")
	}
	if req.PPL == nil {
		t.Error("PPL: got nil, want parsed JSON object")
	}
}
