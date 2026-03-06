package provider

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

// mustMapRoute calls mapRouteResponseToModel and fails the test on error.
func mustMapRoute(t *testing.T, ctx context.Context, apiResponse map[string]interface{}) RouteResourceModel {
	t.Helper()
	model, err := mapRouteResponseToModel(ctx, apiResponse)
	if err != nil {
		t.Fatalf("mapRouteResponseToModel returned unexpected error: %v", err)
	}
	return model
}

func TestMapRouteResponseToModel_RequiredFields(t *testing.T) {
	ctx := context.Background()
	apiResponse := map[string]interface{}{
		"id":          "route-123",
		"name":        "my-route",
		"namespaceId": "ns-456",
		"from":        "https://app.example.com",
		"to":          []interface{}{"https://backend.internal:8080"},
	}

	model := mustMapRoute(t, ctx, apiResponse)

	if model.ID.ValueString() != "route-123" {
		t.Errorf("ID: got %q, want %q", model.ID.ValueString(), "route-123")
	}
	if model.Name.ValueString() != "my-route" {
		t.Errorf("Name: got %q, want %q", model.Name.ValueString(), "my-route")
	}
	if model.NamespaceID.ValueString() != "ns-456" {
		t.Errorf("NamespaceID: got %q, want %q", model.NamespaceID.ValueString(), "ns-456")
	}
	if model.From.ValueString() != "https://app.example.com" {
		t.Errorf("From: got %q, want %q", model.From.ValueString(), "https://app.example.com")
	}
}

func TestMapRouteResponseToModel_MissingRequiredFieldReturnsError(t *testing.T) {
	ctx := context.Background()

	for _, missing := range []string{"id", "name", "namespaceId", "from"} {
		t.Run("missing_"+missing, func(t *testing.T) {
			apiResponse := map[string]interface{}{
				"id":          "route-1",
				"name":        "r",
				"namespaceId": "ns",
				"from":        "https://a.example.com",
			}
			delete(apiResponse, missing)

			_, err := mapRouteResponseToModel(ctx, apiResponse)
			if err == nil {
				t.Errorf("expected error when field %q is missing, got nil", missing)
			}
		})
	}
}

func TestMapRouteResponseToModel_ToField(t *testing.T) {
	ctx := context.Background()

	t.Run("single destination", func(t *testing.T) {
		apiResponse := map[string]interface{}{
			"id":          "route-1",
			"name":        "r",
			"namespaceId": "ns",
			"from":        "https://a.example.com",
			"to":          []interface{}{"https://backend:8080"},
		}
		model := mustMapRoute(t, ctx, apiResponse)
		elems := model.To.Elements()
		if len(elems) != 1 {
			t.Fatalf("To: got %d elements, want 1", len(elems))
		}
	})

	t.Run("multiple destinations", func(t *testing.T) {
		apiResponse := map[string]interface{}{
			"id":          "route-2",
			"name":        "r",
			"namespaceId": "ns",
			"from":        "https://a.example.com",
			"to":          []interface{}{"https://b1:8080", "https://b2:8080"},
		}
		model := mustMapRoute(t, ctx, apiResponse)
		if len(model.To.Elements()) != 2 {
			t.Errorf("To: got %d elements, want 2", len(model.To.Elements()))
		}
	})

	t.Run("missing to field produces empty list", func(t *testing.T) {
		apiResponse := map[string]interface{}{
			"id":          "route-3",
			"name":        "r",
			"namespaceId": "ns",
			"from":        "https://a.example.com",
		}
		model := mustMapRoute(t, ctx, apiResponse)
		if model.To.IsNull() {
			t.Error("To: should not be null when field is absent, expected empty list")
		}
		if len(model.To.Elements()) != 0 {
			t.Errorf("To: got %d elements, want 0", len(model.To.Elements()))
		}
	})
}

func TestMapRouteResponseToModel_BoolFields(t *testing.T) {
	ctx := context.Background()

	t.Run("all bools true", func(t *testing.T) {
		apiResponse := map[string]interface{}{
			"id":          "route-1",
			"name":        "r",
			"namespaceId": "ns",
			"from":        "https://a.example.com",
			"allowSpdy":                                true,
			"allowWebsockets":                          true,
			"enableGoogleCloudServerlessAuthentication": true,
			"passIdentityHeaders":                      true,
			"preserveHostHeader":                       true,
			"showErrorDetails":                         true,
			"tlsSkipVerify":                            true,
			"tlsUpstreamAllowRenegotiation":            true,
		}
		model := mustMapRoute(t, ctx, apiResponse)

		checks := []struct {
			name string
			got  types.Bool
			want bool
		}{
			{"AllowSpdy", model.AllowSpdy, true},
			{"AllowWebsockets", model.AllowWebsockets, true},
			{"EnableGoogleCloudServerlessAuthentication", model.EnableGoogleCloudServerlessAuthentication, true},
			{"PassIdentityHeaders", model.PassIdentityHeaders, true},
			{"PreserveHostHeader", model.PreserveHostHeader, true},
			{"ShowErrorDetails", model.ShowErrorDetails, true},
			{"TLSSkipVerify", model.TLSSkipVerify, true},
			{"TLSUpstreamAllowRenegotiation", model.TLSUpstreamAllowRenegotiation, true},
		}
		for _, c := range checks {
			if c.got.IsNull() {
				t.Errorf("%s: got null, want %v", c.name, c.want)
			} else if c.got.ValueBool() != c.want {
				t.Errorf("%s: got %v, want %v", c.name, c.got.ValueBool(), c.want)
			}
		}
	})

	t.Run("absent Default bools fall back to their schema default", func(t *testing.T) {
		// Attributes declared with Default(...) in the schema should never be
		// null in state — the mapper must apply the same default so that Read
		// doesn't produce spurious drift when the API omits the field.
		apiResponse := map[string]interface{}{
			"id":          "route-1",
			"name":        "r",
			"namespaceId": "ns",
			"from":        "https://a.example.com",
		}
		model := mustMapRoute(t, ctx, apiResponse)

		defaultFalse := []struct {
			name string
			got  types.Bool
		}{
			{"AllowSpdy", model.AllowSpdy},
			{"EnableGoogleCloudServerlessAuthentication", model.EnableGoogleCloudServerlessAuthentication},
			{"PreserveHostHeader", model.PreserveHostHeader},
			{"TLSSkipVerify", model.TLSSkipVerify},
			{"TLSUpstreamAllowRenegotiation", model.TLSUpstreamAllowRenegotiation},
		}
		for _, c := range defaultFalse {
			if c.got.IsNull() {
				t.Errorf("%s: got null, want false (schema default)", c.name)
			} else if c.got.ValueBool() {
				t.Errorf("%s: got true, want false (schema default)", c.name)
			}
		}

		if model.ShowErrorDetails.IsNull() {
			t.Error("ShowErrorDetails: got null, want true (schema default)")
		} else if !model.ShowErrorDetails.ValueBool() {
			t.Errorf("ShowErrorDetails: got false, want true (schema default)")
		}
	})

	t.Run("absent nullable bools are null", func(t *testing.T) {
		// Attributes with Optional+Computed but no Default should stay null
		// when absent from the API response, so the user can omit them.
		apiResponse := map[string]interface{}{
			"id":          "route-1",
			"name":        "r",
			"namespaceId": "ns",
			"from":        "https://a.example.com",
		}
		model := mustMapRoute(t, ctx, apiResponse)

		nullChecks := []struct {
			name string
			got  types.Bool
		}{
			{"AllowWebsockets", model.AllowWebsockets},
			{"PassIdentityHeaders", model.PassIdentityHeaders},
		}
		for _, c := range nullChecks {
			if !c.got.IsNull() {
				t.Errorf("%s: got %v, want null", c.name, c.got.ValueBool())
			}
		}
	})
}

func TestMapRouteResponseToModel_PolicyIDs(t *testing.T) {
	ctx := context.Background()
	base := map[string]interface{}{
		"id":          "route-1",
		"name":        "r",
		"namespaceId": "ns",
		"from":        "https://a.example.com",
	}

	t.Run("policyIds as string array", func(t *testing.T) {
		resp := copyMap(base)
		resp["policyIds"] = []interface{}{"pol-1", "pol-2"}
		model := mustMapRoute(t, ctx, resp)
		if len(model.PolicyIDs.Elements()) != 2 {
			t.Errorf("PolicyIDs: got %d elements, want 2", len(model.PolicyIDs.Elements()))
		}
	})

	t.Run("policies as object array", func(t *testing.T) {
		resp := copyMap(base)
		resp["policies"] = []interface{}{
			map[string]interface{}{"id": "pol-1", "name": "Policy One"},
			map[string]interface{}{"id": "pol-2", "name": "Policy Two"},
		}
		model := mustMapRoute(t, ctx, resp)
		if len(model.PolicyIDs.Elements()) != 2 {
			t.Errorf("PolicyIDs from objects: got %d elements, want 2", len(model.PolicyIDs.Elements()))
		}
	})

	t.Run("no policy fields produces null PolicyIDs", func(t *testing.T) {
		model := mustMapRoute(t, ctx, base)
		if !model.PolicyIDs.IsNull() {
			t.Errorf("PolicyIDs: expected null when absent, got %v", model.PolicyIDs)
		}
	})
}

func TestMapRouteResponseToModel_OptionalStrings(t *testing.T) {
	ctx := context.Background()
	apiResponse := map[string]interface{}{
		"id":                            "route-1",
		"name":                          "r",
		"namespaceId":                   "ns",
		"from":                          "https://a.example.com",
		"prefix":                        "/api",
		"prefixRewrite":                 "/v2",
		"kubernetesServiceAccountToken": "token-abc",
		"tlsDownstreamServerName":       "backend.internal",
	}
	model := mustMapRoute(t, ctx, apiResponse)

	if model.Prefix.ValueString() != "/api" {
		t.Errorf("Prefix: got %q, want %q", model.Prefix.ValueString(), "/api")
	}
	if model.PrefixRewrite.ValueString() != "/v2" {
		t.Errorf("PrefixRewrite: got %q, want %q", model.PrefixRewrite.ValueString(), "/v2")
	}
	if model.KubernetesServiceAccountToken.ValueString() != "token-abc" {
		t.Errorf("KubernetesServiceAccountToken: got %q, want %q", model.KubernetesServiceAccountToken.ValueString(), "token-abc")
	}
	if model.TLSDownstreamServerName.ValueString() != "backend.internal" {
		t.Errorf("TLSDownstreamServerName: got %q, want %q", model.TLSDownstreamServerName.ValueString(), "backend.internal")
	}
}

func TestCreateRouteRequest_RequiredFields(t *testing.T) {
	to, _ := types.ListValueFrom(context.Background(), types.StringType, []string{"https://backend:8080"})
	model := &RouteResourceModel{
		Name:        types.StringValue("my-route"),
		NamespaceID: types.StringValue("ns-123"),
		From:        types.StringValue("https://app.example.com"),
		To:          to,
	}

	req := createRouteRequest(model)

	if req["name"] != "my-route" {
		t.Errorf("name: got %v, want %q", req["name"], "my-route")
	}
	if req["namespaceId"] != "ns-123" {
		t.Errorf("namespaceId: got %v, want %q", req["namespaceId"], "ns-123")
	}
	if req["from"] != "https://app.example.com" {
		t.Errorf("from: got %v, want %q", req["from"], "https://app.example.com")
	}
	toSlice, ok := req["to"].([]string)
	if !ok || len(toSlice) != 1 || toSlice[0] != "https://backend:8080" {
		t.Errorf("to: got %v, want [https://backend:8080]", req["to"])
	}
}

func TestCreateRouteRequest_NullOptionalFieldsOmitted(t *testing.T) {
	to, _ := types.ListValueFrom(context.Background(), types.StringType, []string{"https://backend:8080"})
	model := &RouteResourceModel{
		Name:                    types.StringValue("r"),
		NamespaceID:             types.StringValue("ns"),
		From:                    types.StringValue("https://a.example.com"),
		To:                      to,
		AllowWebsockets:         types.BoolNull(),
		PassIdentityHeaders:     types.BoolNull(),
		PreserveHostHeader:      types.BoolNull(),
		PolicyIDs:               types.ListNull(types.StringType),
		Prefix:                  types.StringNull(),
		PrefixRewrite:           types.StringNull(),
		TLSDownstreamServerName: types.StringNull(),
	}

	req := createRouteRequest(model)

	for _, key := range []string{"allowWebsockets", "passIdentityHeaders", "preserveHostHeader", "policyIds", "prefix", "prefixRewrite", "tlsDownstreamServerName"} {
		if _, exists := req[key]; exists {
			t.Errorf("key %q should not be present when model field is null", key)
		}
	}
}

func TestCreateRouteRequest_NullKubernetesTokenAbsent(t *testing.T) {
	// When KubernetesServiceAccountToken is null, it must not appear in the request.
	to, _ := types.ListValueFrom(context.Background(), types.StringType, []string{"https://backend:8080"})
	model := &RouteResourceModel{
		Name:                          types.StringValue("r"),
		NamespaceID:                   types.StringValue("ns"),
		From:                          types.StringValue("https://a.example.com"),
		To:                            to,
		KubernetesServiceAccountToken: types.StringNull(),
	}

	req := createRouteRequest(model)

	if _, exists := req["kubernetesServiceAccountToken"]; exists {
		t.Error("kubernetesServiceAccountToken should not be present in request when model field is null")
	}
}

func TestUpdateRouteRequest_DelegatesToCreate(t *testing.T) {
	to, _ := types.ListValueFrom(context.Background(), types.StringType, []string{"https://backend:8080"})
	model := &RouteResourceModel{
		Name:        types.StringValue("r"),
		NamespaceID: types.StringValue("ns"),
		From:        types.StringValue("https://a.example.com"),
		To:          to,
	}

	createReq := createRouteRequest(model)
	updateReq := updateRouteRequest(model)

	if len(createReq) != len(updateReq) {
		t.Errorf("updateRouteRequest differs from createRouteRequest: create has %d keys, update has %d keys", len(createReq), len(updateReq))
	}
	// Use fmt.Sprintf for comparison because map values may contain slices,
	// which are not comparable with ==.
	for k, cv := range createReq {
		uv, ok := updateReq[k]
		if !ok {
			t.Errorf("updateRouteRequest missing key %q", k)
		} else if fmt.Sprintf("%v", cv) != fmt.Sprintf("%v", uv) {
			t.Errorf("key %q: create=%v update=%v", k, cv, uv)
		}
	}
}

// copyMap creates a shallow copy of a map[string]interface{}.
func copyMap(m map[string]interface{}) map[string]interface{} {
	out := make(map[string]interface{}, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}
