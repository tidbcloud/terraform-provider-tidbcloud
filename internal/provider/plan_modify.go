package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

// clusterResourceStatusModifier implements the plan modifier.
type clusterResourceStatusModifier struct{}

func (m clusterResourceStatusModifier) Description(_ context.Context) string {
	return "The plan modifier for status attribute. It will apply useStateForUnknownModifier to all the nested attributes except the cluster_status attribute."
}

// MarkdownDescription returns a markdown description of the plan modifier.
func (m clusterResourceStatusModifier) MarkdownDescription(_ context.Context) string {
	return "The plan modifier for status attribute. It will apply useStateForUnknownModifier to all the nested attributes except the cluster_status attribute."
}

// PlanModifyObject implements the plan modification logic.
func (m clusterResourceStatusModifier) PlanModifyObject(ctx context.Context, req planmodifier.ObjectRequest, resp *planmodifier.ObjectResponse) {
	// Do nothing if there is no state value.
	if req.StateValue.IsNull() {
		return
	}

	// Do nothing if there is a known planned value.
	if !req.PlanValue.IsUnknown() {
		return
	}

	// Do nothing if there is an unknown configuration value, otherwise interpolation gets messed up.
	if req.ConfigValue.IsUnknown() {
		return
	}
	// Does not apply to cluster_status attribute
	attributes := req.StateValue.Attributes()
	attributes["cluster_status"] = types.StringUnknown()
	newStateValue, diag := basetypes.NewObjectValue(req.StateValue.AttributeTypes(ctx), attributes)

	resp.Diagnostics.Append(diag...)
	resp.PlanValue = newStateValue
}

func clusterResourceStatus() planmodifier.Object {
	return clusterResourceStatusModifier{}
}
