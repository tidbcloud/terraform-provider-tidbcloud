package provider

import (
	"context"
	"errors"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/tidbcloud/tidbcloud-cli/pkg/tidbcloud/v1beta1/dedicated"
)

var (
	_ resource.Resource = &DedicatedAuditLogFilterRuleResource{}
)

type DedicatedAuditLogFilterRuleResource struct {
	provider *tidbcloudProvider
}

type DedicatedAuditLogFilterRuleResourceData struct {
	AuditLogFilterRuleId types.String `tfsdk:"audit_log_filter_rule_id"`
	ClusterId            types.String `tfsdk:"cluster_id"`
	UserExpr             types.String `tfsdk:"user_expr"`
	DBExpr               types.String `tfsdk:"db_expr"`
	TableExpr            types.String `tfsdk:"table_expr"`
	AccessTypeList       types.List   `tfsdk:"access_type_list"`
}

func NewDedicatedAuditLogFilterRuleResource() resource.Resource {
	return &DedicatedAuditLogFilterRuleResource{}
}

func (r *DedicatedAuditLogFilterRuleResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dedicated_audit_log_filter_rule"
}

func (r *DedicatedAuditLogFilterRuleResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "dedicated audit log filter rule",
		Attributes: map[string]schema.Attribute{
			"audit_log_filter_rule_id": schema.StringAttribute{
				Description: "The ID of the audit log filter rule",
				Computed:    true,
			},
			"cluster_id": schema.StringAttribute{
				Description: "The ID of the cluster",
				Required:    true,
			},
			"user_expr": schema.StringAttribute{
				Description: "The user expression",
				Required:    true,
			},
			"db_expr": schema.StringAttribute{
				Description: "The db expression",
				Required:    true,
			},
			"table_expr": schema.StringAttribute{
				Description: "The table expression",
				Required:    true,
			},
			"access_type_list": schema.ListAttribute{
				Description: "The access type list",
				Required:    true,
				ElementType: types.StringType,
			},
		},
	}
}

func (r *DedicatedAuditLogFilterRuleResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	var ok bool
	r.provider, ok = req.ProviderData.(*tidbcloudProvider)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *tidbcloudProvider, got: %T", req.ProviderData),
		)
	}
}

func (r *DedicatedAuditLogFilterRuleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if !r.provider.configured {
		resp.Diagnostics.AddError(
			"Provider not configured",
			"The provider hasn't been configured before apply, likely because it depends on an unknown value from another resource. This leads to weird stuff happening, so we'd prefer if you didn't do that. Thanks!",
		)
		return
	}

	var data DedicatedAuditLogFilterRuleResourceData
	diags := req.Plan.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Trace(ctx, "create dedicated_audit_log_filter_rule_resource")
	body, err := buildCreateDedicatedAuditLogFilterRuleBody(ctx, data)
	if err != nil {
		resp.Diagnostics.AddError("Create Error", fmt.Sprintf("Unable to build create body, got error: %s", err))
		return
	}
	AuditLogFilterRule, err := r.provider.DedicatedClient.CreateAuditLogFilterRule(ctx, data.ClusterId.ValueString(), &body)
	if err != nil {
		resp.Diagnostics.AddError("Create Error", fmt.Sprintf("Unable to call CreateAuditLogFilterRule, got error: %s", err))
		return
	}

	data.AuditLogFilterRuleId = types.StringValue(*AuditLogFilterRule.AuditLogFilterRuleId)

	// save into the Terraform state.
	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (r *DedicatedAuditLogFilterRuleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data DedicatedAuditLogFilterRuleResourceData
	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, fmt.Sprintf("read dedicated_audit_log_filter_rule_resource audit_log_filter_rule_id: %s", data.AuditLogFilterRuleId.ValueString()))

	// call read api
	tflog.Trace(ctx, "read dedicated_audit_log_filter_rule_resource")
	auditLogFilterRule, err := r.provider.DedicatedClient.GetAuditLogFilterRule(ctx, data.ClusterId.ValueString(), data.AuditLogFilterRuleId.ValueString())
	if err != nil {
		tflog.Error(ctx, fmt.Sprintf("Unable to call GetAuditLogFilterRule, error: %s", err))
		return
	}

	data.AuditLogFilterRuleId = types.StringValue(*auditLogFilterRule.AuditLogFilterRuleId)

	// save into the Terraform state.
	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (r *DedicatedAuditLogFilterRuleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Update Error", "Update is not supported for dedicated audit log filter rule")
	return
}

func (r *DedicatedAuditLogFilterRuleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var AuditLogFilterRuleId string
	var ClusterId string
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("audit_log_filter_rule_id"), &AuditLogFilterRuleId)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("cluster_id"), &ClusterId)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Trace(ctx, "delete dedicated_audit_log_filter_rule_resource")
	err := r.provider.DedicatedClient.DeleteAuditLogFilterRule(ctx, ClusterId, AuditLogFilterRuleId)
	if err != nil {
		resp.Diagnostics.AddError("Delete Error", fmt.Sprintf("Unable to call DeleteAuditLogFilterRule, got error: %s", err))
		return
	}
}

func buildCreateDedicatedAuditLogFilterRuleBody(ctx context.Context, data DedicatedAuditLogFilterRuleResourceData) (dedicated.DatabaseAuditLogServiceCreateAuditLogFilterRuleRequest, error) {
	userExpr := data.UserExpr.ValueString()
	dbExpr := data.DBExpr.ValueString()
	tableExpr := data.TableExpr.ValueString()

	var accessTypeList []string
	diag := data.AccessTypeList.ElementsAs(ctx, &accessTypeList, false)
	if diag.HasError() {
		return dedicated.DatabaseAuditLogServiceCreateAuditLogFilterRuleRequest{}, errors.New("unable to get access type list")
	}

	return dedicated.DatabaseAuditLogServiceCreateAuditLogFilterRuleRequest{
		UserExpr:       &userExpr,
		DbExpr:         &dbExpr,
		TableExpr:      &tableExpr,
		AccessTypeList: accessTypeList,
	}, nil
}

