package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type dedicatedAuditLogFilterRuleDataSourceData struct {
	AuditLogFilterRuleId types.String `tfsdk:"audit_log_filter_rule_id"`
	ClusterId            types.String `tfsdk:"cluster_id"`
	UserExpr             types.String `tfsdk:"user_expr"`
	DBExpr               types.String `tfsdk:"db_expr"`
	TableExpr            types.String `tfsdk:"table_expr"`
	AccessTypeList       types.List   `tfsdk:"access_type_list"`
}

var _ datasource.DataSource = &dedicatedAuditLogFilterRuleDataSource{}

type dedicatedAuditLogFilterRuleDataSource struct {
	provider *tidbcloudProvider
}

func NewDedicatedAuditLogFilterRuleDataSource() datasource.DataSource {
	return &dedicatedAuditLogFilterRuleDataSource{}
}

func (d *dedicatedAuditLogFilterRuleDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dedicated_audit_log_filter_rule"
}

func (d *dedicatedAuditLogFilterRuleDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	var ok bool
	if d.provider, ok = req.ProviderData.(*tidbcloudProvider); !ok {
		resp.Diagnostics.AddError("Internal provider error",
			fmt.Sprintf("Error in Configure: expected %T but got %T", tidbcloudProvider{}, req.ProviderData))
	}
}

func (d *dedicatedAuditLogFilterRuleDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "dedicated region data source",
		Attributes: map[string]schema.Attribute{
			"audit_log_filter_rule_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the audit log filter rule",
				Required: 		  true,
			},
			"cluster_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the cluster",
				Required: 		  true,
			},
			"user_expr": schema.StringAttribute{
				MarkdownDescription: "The user expression",
				Computed: 		  true,
			},
			"db_expr": schema.StringAttribute{
				MarkdownDescription: "The db expression",
				Computed: 		  true,
			},
			"table_expr": schema.StringAttribute{
				MarkdownDescription: "The table expression",
				Computed: 		  true,
			},
			"access_type_list": schema.ListAttribute{
				MarkdownDescription: "The access type list",
				Computed: 		  true,
			},
		},
	}
}

func (d *dedicatedAuditLogFilterRuleDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data dedicatedAuditLogFilterRuleDataSourceData
	diags := req.Config.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Trace(ctx, "read audit log filter rule data source")

	auditLogFilterRule, err := d.provider.DedicatedClient.GetAuditLogFilterRule(ctx, data.ClusterId.ValueString(), data.AuditLogFilterRuleId.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Read Error", fmt.Sprintf("Unable to call GetAuditLogFilterRule, got error: %s", err))
		return
	}
	data.UserExpr = types.StringValue(*auditLogFilterRule.UserExpr)
	data.DBExpr = types.StringValue(*auditLogFilterRule.DbExpr)
	data.TableExpr = types.StringValue(*auditLogFilterRule.TableExpr)
	diags = data.AccessTypeList.ElementsAs(ctx, &auditLogFilterRule.AccessTypeList, false)
	if resp.Diagnostics.HasError() {
		return
	}
	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}
