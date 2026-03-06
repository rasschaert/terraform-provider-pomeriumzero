package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

// helpers

func fullClusterSettings() *ClusterSettings {
	secret := "s3cr3t"
	return &ClusterSettings{
		ID:                           "cs-123",
		Address:                      ":443",
		AuthenticateServiceUrl:       "https://auth.example.com",
		AutoApplyChangesets:          true,
		CookieExpire:                 "14h",
		CookieHttpOnly:               true,
		CookieName:                   "_pomerium",
		DefaultUpstreamTimeout:       "30s",
		DNSLookupFamily:              "v4",
		IdentityProvider:             "google",
		IdentityProviderClientId:     "client-id",
		IdentityProviderClientSecret: &secret,
		IdentityProviderUrl:          "https://idp.example.com",
		LogLevel:                     "info",
		PassIdentityHeaders:          true,
		ProxyLogLevel:                "warn",
		SkipXffAppend:                false,
		TimeoutIdle:                  "5m",
		TimeoutRead:                  "10s",
		TimeoutWrite:                 "10s",
		TracingSampleRate:            0.5,
	}
}

func fullClusterSettingsModel() ClusterSettingsResourceModel {
	secret := "s3cr3t"
	return ClusterSettingsResourceModel{
		ID:                           types.StringValue("cluster-123"),
		Address:                      types.StringValue(":443"),
		AuthenticateServiceUrl:       types.StringValue("https://auth.example.com"),
		AutoApplyChangesets:          types.BoolValue(true),
		CookieExpire:                 types.StringValue("14h"),
		CookieHttpOnly:               types.BoolValue(true),
		CookieName:                   types.StringValue("_pomerium"),
		DefaultUpstreamTimeout:       types.StringValue("30s"),
		DNSLookupFamily:              types.StringValue("v4"),
		IdentityProvider:             types.StringValue("google"),
		IdentityProviderClientId:     types.StringValue("client-id"),
		IdentityProviderClientSecret: types.StringValue(secret),
		IdentityProviderUrl:          types.StringValue("https://idp.example.com"),
		LogLevel:                     types.StringValue("info"),
		PassIdentityHeaders:          types.BoolValue(true),
		ProxyLogLevel:                types.StringValue("warn"),
		SkipXffAppend:                types.BoolValue(false),
		TimeoutIdle:                  types.StringValue("5m"),
		TimeoutRead:                  types.StringValue("10s"),
		TimeoutWrite:                 types.StringValue("10s"),
		TracingSampleRate:            types.Float64Value(0.5),
		CodecType:                    types.StringValue("auto"),
	}
}

// --- updateClusterSettingsResourceModel ---

func TestUpdateClusterSettingsResourceModel_AllFields(t *testing.T) {
	settings := fullClusterSettings()
	var model ClusterSettingsResourceModel
	updateClusterSettingsResourceModel(&model, settings)

	checks := []struct {
		field string
		got   string
		want  string
	}{
		{"Address", model.Address.ValueString(), ":443"},
		{"AuthenticateServiceUrl", model.AuthenticateServiceUrl.ValueString(), "https://auth.example.com"},
		{"CookieExpire", model.CookieExpire.ValueString(), "14h"},
		{"CookieName", model.CookieName.ValueString(), "_pomerium"},
		{"DefaultUpstreamTimeout", model.DefaultUpstreamTimeout.ValueString(), "30s"},
		{"DNSLookupFamily", model.DNSLookupFamily.ValueString(), "v4"},
		{"IdentityProvider", model.IdentityProvider.ValueString(), "google"},
		{"IdentityProviderClientId", model.IdentityProviderClientId.ValueString(), "client-id"},
		{"IdentityProviderUrl", model.IdentityProviderUrl.ValueString(), "https://idp.example.com"},
		{"LogLevel", model.LogLevel.ValueString(), "info"},
		{"ProxyLogLevel", model.ProxyLogLevel.ValueString(), "warn"},
		{"TimeoutIdle", model.TimeoutIdle.ValueString(), "5m"},
		{"TimeoutRead", model.TimeoutRead.ValueString(), "10s"},
		{"TimeoutWrite", model.TimeoutWrite.ValueString(), "10s"},
	}
	for _, c := range checks {
		if c.got != c.want {
			t.Errorf("%s: got %q, want %q", c.field, c.got, c.want)
		}
	}

	if !model.AutoApplyChangesets.ValueBool() {
		t.Error("AutoApplyChangesets: got false, want true")
	}
	if !model.CookieHttpOnly.ValueBool() {
		t.Error("CookieHttpOnly: got false, want true")
	}
	if !model.PassIdentityHeaders.ValueBool() {
		t.Error("PassIdentityHeaders: got false, want true")
	}
	// SkipXffAppend is false in the fixture — optBool maps false → null for
	// Optional-only bools, so the field should be null rather than false.
	if !model.SkipXffAppend.IsNull() {
		t.Errorf("SkipXffAppend: got %v, want null (false is the zero value for Optional-only bool)", model.SkipXffAppend.ValueBool())
	}
	if model.TracingSampleRate.ValueFloat64() != 0.5 {
		t.Errorf("TracingSampleRate: got %v, want 0.5", model.TracingSampleRate.ValueFloat64())
	}

	// IdentityProviderClientSecret is a pointer — should be set
	if model.IdentityProviderClientSecret.IsNull() {
		t.Error("IdentityProviderClientSecret: got null, want s3cr3t")
	} else if model.IdentityProviderClientSecret.ValueString() != "s3cr3t" {
		t.Errorf("IdentityProviderClientSecret: got %q, want %q", model.IdentityProviderClientSecret.ValueString(), "s3cr3t")
	}
}

func TestUpdateClusterSettingsResourceModel_NilClientSecret(t *testing.T) {
	settings := fullClusterSettings()
	settings.IdentityProviderClientSecret = nil

	var model ClusterSettingsResourceModel
	updateClusterSettingsResourceModel(&model, settings)

	if !model.IdentityProviderClientSecret.IsNull() {
		t.Errorf("IdentityProviderClientSecret: expected null when API returns nil, got %q", model.IdentityProviderClientSecret.ValueString())
	}
}

func TestUpdateClusterSettingsResourceModel_ZeroValuesAreNull(t *testing.T) {
	// All Optional-only fields should be null when the API returns zero values,
	// so that configs omitting those fields don't drift against a concrete state.
	settings := &ClusterSettings{} // all zero values
	var model ClusterSettingsResourceModel
	updateClusterSettingsResourceModel(&model, settings)

	nullStrings := []struct {
		name string
		got  types.String
	}{
		{"Address", model.Address},
		{"AuthenticateServiceUrl", model.AuthenticateServiceUrl},
		{"CookieExpire", model.CookieExpire},
		{"CookieName", model.CookieName},
		{"DefaultUpstreamTimeout", model.DefaultUpstreamTimeout},
		{"DNSLookupFamily", model.DNSLookupFamily},
		{"IdentityProvider", model.IdentityProvider},
		{"IdentityProviderClientId", model.IdentityProviderClientId},
		{"IdentityProviderUrl", model.IdentityProviderUrl},
		{"LogLevel", model.LogLevel},
		{"ProxyLogLevel", model.ProxyLogLevel},
		{"TimeoutIdle", model.TimeoutIdle},
		{"TimeoutRead", model.TimeoutRead},
		{"TimeoutWrite", model.TimeoutWrite},
		{"CodecType", model.CodecType},
	}
	for _, c := range nullStrings {
		if !c.got.IsNull() {
			t.Errorf("%s: got %q, want null for empty API response", c.name, c.got.ValueString())
		}
	}

	nullBools := []struct {
		name string
		got  types.Bool
	}{
		{"AutoApplyChangesets", model.AutoApplyChangesets},
		{"CookieHttpOnly", model.CookieHttpOnly},
		{"PassIdentityHeaders", model.PassIdentityHeaders},
		{"SkipXffAppend", model.SkipXffAppend},
	}
	for _, c := range nullBools {
		if !c.got.IsNull() {
			t.Errorf("%s: got %v, want null for false API response", c.name, c.got.ValueBool())
		}
	}

	if !model.TracingSampleRate.IsNull() {
		t.Errorf("TracingSampleRate: got %v, want null for zero API response", model.TracingSampleRate.ValueFloat64())
	}
}

func TestUpdateClusterSettingsResourceModel_EmptyProxyLogLevelBecomesNull(t *testing.T) {
	settings := fullClusterSettings()
	settings.ProxyLogLevel = ""

	var model ClusterSettingsResourceModel
	updateClusterSettingsResourceModel(&model, settings)

	if !model.ProxyLogLevel.IsNull() {
		t.Errorf("ProxyLogLevel: expected null for empty string from API, got %q", model.ProxyLogLevel.ValueString())
	}
}

func TestUpdateClusterSettingsResourceModel_NonEmptyProxyLogLevel(t *testing.T) {
	settings := fullClusterSettings()
	settings.ProxyLogLevel = "debug"

	var model ClusterSettingsResourceModel
	updateClusterSettingsResourceModel(&model, settings)

	if model.ProxyLogLevel.IsNull() {
		t.Error("ProxyLogLevel: expected non-null for non-empty string")
	}
	if model.ProxyLogLevel.ValueString() != "debug" {
		t.Errorf("ProxyLogLevel: got %q, want %q", model.ProxyLogLevel.ValueString(), "debug")
	}
}

// --- createClusterSettingsRequest ---

func TestCreateClusterSettingsRequest_AllFields(t *testing.T) {
	model := fullClusterSettingsModel()
	req := createClusterSettingsRequest(model)

	if req.Address != ":443" {
		t.Errorf("Address: got %q, want %q", req.Address, ":443")
	}
	if req.AuthenticateServiceUrl != "https://auth.example.com" {
		t.Errorf("AuthenticateServiceUrl: got %q, want %q", req.AuthenticateServiceUrl, "https://auth.example.com")
	}
	if !req.AutoApplyChangesets {
		t.Error("AutoApplyChangesets: got false, want true")
	}
	if req.IdentityProvider != "google" {
		t.Errorf("IdentityProvider: got %q, want %q", req.IdentityProvider, "google")
	}
	if req.IdentityProviderClientSecret != "s3cr3t" {
		t.Errorf("IdentityProviderClientSecret: got %q, want %q", req.IdentityProviderClientSecret, "s3cr3t")
	}
	if req.ProxyLogLevel != "warn" {
		t.Errorf("ProxyLogLevel: got %q, want %q", req.ProxyLogLevel, "warn")
	}
	if req.TracingSampleRate != 0.5 {
		t.Errorf("TracingSampleRate: got %v, want 0.5", req.TracingSampleRate)
	}
	if req.CodecType != "auto" {
		t.Errorf("CodecType: got %q, want %q", req.CodecType, "auto")
	}
}

// --- updateClusterSettingsRequest ---

func TestUpdateClusterSettingsRequest_NonNullIDPFieldsIncluded(t *testing.T) {
	model := fullClusterSettingsModel()
	req := updateClusterSettingsRequest(model)

	if req.AuthenticateServiceUrl != "https://auth.example.com" {
		t.Errorf("AuthenticateServiceUrl: got %q, want %q", req.AuthenticateServiceUrl, "https://auth.example.com")
	}
	if req.IdentityProvider != "google" {
		t.Errorf("IdentityProvider: got %q, want %q", req.IdentityProvider, "google")
	}
	if req.IdentityProviderClientId != "client-id" {
		t.Errorf("IdentityProviderClientId: got %q, want %q", req.IdentityProviderClientId, "client-id")
	}
	if req.IdentityProviderClientSecret == nil {
		t.Error("IdentityProviderClientSecret: expected non-nil pointer")
	} else if *req.IdentityProviderClientSecret != "s3cr3t" {
		t.Errorf("IdentityProviderClientSecret: got %q, want %q", *req.IdentityProviderClientSecret, "s3cr3t")
	}
	if req.IdentityProviderUrl != "https://idp.example.com" {
		t.Errorf("IdentityProviderUrl: got %q, want %q", req.IdentityProviderUrl, "https://idp.example.com")
	}
}

func TestUpdateClusterSettingsRequest_NullIDPFieldsOmitted(t *testing.T) {
	model := fullClusterSettingsModel()
	model.AuthenticateServiceUrl = types.StringNull()
	model.IdentityProvider = types.StringNull()
	model.IdentityProviderClientId = types.StringNull()
	model.IdentityProviderClientSecret = types.StringNull()
	model.IdentityProviderUrl = types.StringNull()

	req := updateClusterSettingsRequest(model)

	if req.AuthenticateServiceUrl != "" {
		t.Errorf("AuthenticateServiceUrl: expected empty when null, got %q", req.AuthenticateServiceUrl)
	}
	if req.IdentityProvider != "" {
		t.Errorf("IdentityProvider: expected empty when null, got %q", req.IdentityProvider)
	}
	if req.IdentityProviderClientId != "" {
		t.Errorf("IdentityProviderClientId: expected empty when null, got %q", req.IdentityProviderClientId)
	}
	if req.IdentityProviderClientSecret != nil {
		t.Errorf("IdentityProviderClientSecret: expected nil when null, got %v", req.IdentityProviderClientSecret)
	}
	if req.IdentityProviderUrl != "" {
		t.Errorf("IdentityProviderUrl: expected empty when null, got %q", req.IdentityProviderUrl)
	}
}

func TestUpdateClusterSettingsRequest_NullProxyLogLevelOmitted(t *testing.T) {
	model := fullClusterSettingsModel()
	model.ProxyLogLevel = types.StringNull()

	req := updateClusterSettingsRequest(model)

	if req.ProxyLogLevel != "" {
		t.Errorf("ProxyLogLevel: expected empty when null, got %q", req.ProxyLogLevel)
	}
}

func TestUpdateClusterSettingsRequest_EmptyProxyLogLevelOmitted(t *testing.T) {
	model := fullClusterSettingsModel()
	model.ProxyLogLevel = types.StringValue("")

	req := updateClusterSettingsRequest(model)

	if req.ProxyLogLevel != "" {
		t.Errorf("ProxyLogLevel: expected empty when set to empty string, got %q", req.ProxyLogLevel)
	}
}

func TestUpdateClusterSettingsRequest_BoolFieldsAlwaysPresent(t *testing.T) {
	// PassIdentityHeaders and SkipXffAppend must always be sent (no omitempty).
	// We verify the request struct has the correct values even when false.
	model := fullClusterSettingsModel()
	model.PassIdentityHeaders = types.BoolValue(false)
	model.SkipXffAppend = types.BoolValue(false)

	req := updateClusterSettingsRequest(model)

	if req.PassIdentityHeaders {
		t.Error("PassIdentityHeaders: got true, want false")
	}
	if req.SkipXffAppend {
		t.Error("SkipXffAppend: got true, want false")
	}
}

func TestUpdateClusterSettingsRequest_CodecType(t *testing.T) {
	for _, codec := range []string{"", "auto", "http1", "http2", "http3"} {
		model := fullClusterSettingsModel()
		model.CodecType = types.StringValue(codec)
		req := updateClusterSettingsRequest(model)
		if req.CodecType != codec {
			t.Errorf("CodecType %q: got %q", codec, req.CodecType)
		}
	}
}
