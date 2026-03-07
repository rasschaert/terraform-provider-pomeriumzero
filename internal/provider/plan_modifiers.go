package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// useStateUnlessUpdating is a plan modifier for Computed-only timestamp attributes
// (like updated_at) that resolves the tension between two conflicting requirements:
//
//  1. UseStateForUnknown would prevent phantom diffs (Computed attr Unknown in plan ≠
//     known in state causes Terraform to trigger unnecessary updates). BUT it also causes
//     "Provider produced inconsistent result after apply" when a real update fires,
//     because the plan says "old timestamp" while the API returns a new one.
//
//  2. No modifier → Computed attr is always Unknown in plan → phantom update on every run.
//
// This modifier uses the state value only when the plan has no user-driven changes,
// and leaves the attribute Unknown when an actual update is occurring.
type useStateUnlessUpdating struct{}

var _ planmodifier.String = useStateUnlessUpdating{}

func (m useStateUnlessUpdating) Description(_ context.Context) string {
	return "Uses prior state value when no changes are planned; remains (known after apply) during updates."
}

func (m useStateUnlessUpdating) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m useStateUnlessUpdating) PlanModifyString(_ context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	if !req.PlanValue.IsUnknown() {
		return
	}
	if req.StateValue.IsNull() || req.StateValue.IsUnknown() {
		return
	}

	var planAttrs, stateAttrs map[string]tftypes.Value
	if err := req.Plan.Raw.As(&planAttrs); err != nil {
		return // can't compare — leave Unknown (safe default)
	}
	if err := req.State.Raw.As(&stateAttrs); err != nil {
		return
	}

	// If any known (non-Unknown) plan attribute differs from its state counterpart,
	// a real update is occurring. Leave updated_at Unknown so the new API timestamp
	// passes Terraform's consistency check.
	for k, pv := range planAttrs {
		if !pv.IsKnown() {
			continue
		}
		sv, ok := stateAttrs[k]
		if !ok || !pv.Equal(sv) {
			return
		}
	}

	// No user-driven changes detected — use state value to prevent phantom diff.
	resp.PlanValue = req.StateValue
}
