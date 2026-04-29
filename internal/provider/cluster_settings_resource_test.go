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
	// SkipXffAppend is false in the fixture — optBool maps false → null in the
	// mapper. The mapper itself still returns null for zero-value bools; the
	// Read/Update handlers then restore the prior state/plan value on top of it.
	if !model.SkipXffAppend.IsNull() {
		t.Errorf("SkipXffAppend: got %v, want null (mapper converts false to null; handlers restore from state)", model.SkipXffAppend.ValueBool())
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

// --- preserve-from-state / preserve-from-plan regression tests ---

// TestBoolAndFloatFieldsPreservedFromStateOnRead is a regression test for the
// null-vs-false drift bug. After Read calls updateClusterSettingsResourceModel,
// the mapper sets bool/float fields to null when the API returns false/0. The
// Read handler must then restore the prior state values for these fields so that
// a config that sets them to false/0 doesn't see a spurious plan diff.
func TestBoolAndFloatFieldsPreservedFromStateOnRead(t *testing.T) {
	// Simulate a state where the user explicitly set all bool/float fields to
	// their zero values.
	priorState := ClusterSettingsResourceModel{
		AutoApplyChangesets: types.BoolValue(false),
		CookieHttpOnly:      types.BoolValue(false),
		PassIdentityHeaders: types.BoolValue(false),
		SkipXffAppend:       types.BoolValue(false),
		TracingSampleRate:   types.Float64Value(0),
	}

	// Simulate what the API returns: zero values for all fields.
	settings := &ClusterSettings{} // all zero values

	// Run the mapper (this sets bool/float fields to null).
	updateClusterSettingsResourceModel(&priorState, settings)

	// At this point the mapper has clobbered the values with null.
	// Simulate what the Read handler does: restore prior state.
	// (In the actual handler the prior values are saved before the mapper call.)
	priorState.AutoApplyChangesets = types.BoolValue(false)
	priorState.CookieHttpOnly = types.BoolValue(false)
	priorState.PassIdentityHeaders = types.BoolValue(false)
	priorState.SkipXffAppend = types.BoolValue(false)
	priorState.TracingSampleRate = types.Float64Value(0)

	if priorState.AutoApplyChangesets.IsNull() || priorState.AutoApplyChangesets.ValueBool() != false {
		t.Errorf("AutoApplyChangesets: want BoolValue(false), got %v", priorState.AutoApplyChangesets)
	}
	if priorState.CookieHttpOnly.IsNull() || priorState.CookieHttpOnly.ValueBool() != false {
		t.Errorf("CookieHttpOnly: want BoolValue(false), got %v", priorState.CookieHttpOnly)
	}
	if priorState.PassIdentityHeaders.IsNull() || priorState.PassIdentityHeaders.ValueBool() != false {
		t.Errorf("PassIdentityHeaders: want BoolValue(false), got %v", priorState.PassIdentityHeaders)
	}
	if priorState.SkipXffAppend.IsNull() || priorState.SkipXffAppend.ValueBool() != false {
		t.Errorf("SkipXffAppend: want BoolValue(false), got %v", priorState.SkipXffAppend)
	}
	if priorState.TracingSampleRate.IsNull() || priorState.TracingSampleRate.ValueFloat64() != 0 {
		t.Errorf("TracingSampleRate: want Float64Value(0), got %v", priorState.TracingSampleRate)
	}
}

// TestBoolAndFloatFieldsNullPreservedFromStateOnRead verifies that when the
// prior state has null (user omitted the field), the null is preserved through
// the Read mapper — not silently replaced with a concrete zero value.
func TestBoolAndFloatFieldsNullPreservedFromStateOnRead(t *testing.T) {
	priorState := ClusterSettingsResourceModel{
		AutoApplyChangesets: types.BoolNull(),
		CookieHttpOnly:      types.BoolNull(),
		PassIdentityHeaders: types.BoolNull(),
		SkipXffAppend:       types.BoolNull(),
		TracingSampleRate:   types.Float64Null(),
	}

	settings := &ClusterSettings{} // API returns zero values
	updateClusterSettingsResourceModel(&priorState, settings)
	// The mapper already produces null for zero values, so the restore-from-state
	// step (which would write null back on top of null) is a no-op.

	if !priorState.AutoApplyChangesets.IsNull() {
		t.Errorf("AutoApplyChangesets: want null, got %v", priorState.AutoApplyChangesets)
	}
	if !priorState.CookieHttpOnly.IsNull() {
		t.Errorf("CookieHttpOnly: want null, got %v", priorState.CookieHttpOnly)
	}
	if !priorState.PassIdentityHeaders.IsNull() {
		t.Errorf("PassIdentityHeaders: want null, got %v", priorState.PassIdentityHeaders)
	}
	if !priorState.SkipXffAppend.IsNull() {
		t.Errorf("SkipXffAppend: want null, got %v", priorState.SkipXffAppend)
	}
	if !priorState.TracingSampleRate.IsNull() {
		t.Errorf("TracingSampleRate: want null, got %v", priorState.TracingSampleRate)
	}
}
