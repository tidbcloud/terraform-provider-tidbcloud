package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/juju/errors"
	"github.com/tidbcloud/tidbcloud-cli/pkg/tidbcloud/v1beta1/dedicated"
)

type dedicatedAuditLogFilterRulesDataSourceData struct {
	ClusterId           types.String                  `tfsdk:"cluster_id"`
	AuditLogFilterRules []dedicatedAuditLogFilterRule `tfsdk:"audit_log_filter_rules"`
}

type dedicatedAuditLogFilterRule struct {
	AuditLogFilterRuleId types.String `tfsdk:"audit_log_filter_rule_id"`
	UserExpr             types.String `tfsdk:"user_expr"`
	DBExpr               types.String `tfsdk:"db_expr"`
	TableExpr            types.String `tfsdk:"table_expr"`
	AccessTypeList       types.List   `tfsdk:"access_type_list"`
}

var _ datasource.DataSource = &dedicatedAuditLogFilterRulesDataSource{}

type dedicatedAuditLogFilterRulesDataSource struct {
	provider *tidbcloudProvider
}

func NewDedicatedAuditLogFilterRulesDataSource() datasource.DataSource {
	return &dedicatedAuditLogFilterRulesDataSource{}
}

func (d *dedicatedAuditLogFilterRulesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dedicated_audit_log_filter_rules"
}

func (d *dedicatedAuditLogFilterRulesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	var ok bool
	if d.provider, ok = req.ProviderData.(*tidbcloudProvider); !ok {
		resp.Diagnostics.AddError("Internal provider error",
			fmt.Sprintf("Error in Configure: expected %T but got %T", tidbcloudProvider{}, req.ProviderData))
	}
}

func (d *dedicatedAuditLogFilterRulesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "dedicated audit log filter rules data source",
		Attributes: map[string]schema.Attribute{
			"cluster_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the cluster",
				Optional:            true,
			},
			"audit_log_filter_rules": schema.ListNestedAttribute{
				MarkdownDescription: "The list of audit log filter rules",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"audit_log_filter_rule_id": schema.StringAttribute{
							MarkdownDescription: "The ID of the audit log filter rule.",
							Computed:            true,
						},
						"user_expr": schema.StringAttribute{
							MarkdownDescription: "The user expression.",
							Computed:            true,
						},
						"db_expr": schema.StringAttribute{
							MarkdownDescription: "The db expression.",
							Computed:            true,
						},
						"table_expr": schema.StringAttribute{
							MarkdownDescription: "The table expression.",
							Computed:            true,
						},
						"access_type_list": schema.ListAttribute{
							MarkdownDescription: "The access type list.",
							Computed:            true,
							ElementType:         types.StringType,
						},
					},
				},
			},
		},
	}
}

func (d *dedicatedAuditLogFilterRulesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data dedicatedAuditLogFilterRulesDataSourceData
	diags := req.Config.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Trace(ctx, "read dedicated audit log filter rules data source")
	auditLogFilterRules, err := d.retrieveAuditLogFilterRules(ctx, data.ClusterId.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Read Error", fmt.Sprintf("Unable to call ListAuditLogFilterRules, got error: %s", err))
		return
	}
	var items []dedicatedAuditLogFilterRule
	for _, r := range auditLogFilterRules {
		rule := dedicatedAuditLogFilterRule{}
		accessTypeList, diags := types.ListValueFrom(ctx, types.StringType, r.AccessTypeList)
		if resp.Diagnostics.HasError() {
			resp.Diagnostics.AddError("Read Error", fmt.Sprintf("Unable to convert AccessTypeList to list, got error: %s", diags))
			return
		}
		rule.UserExpr = types.StringValue(*r.UserExpr)
		rule.DBExpr = types.StringValue(*r.DbExpr)
		rule.TableExpr = types.StringValue(*r.TableExpr)
		rule.AuditLogFilterRuleId = types.StringValue(*r.AuditLogFilterRuleId)
		rule.AccessTypeList = accessTypeList

		items = append(items, rule)
	}
	data.AuditLogFilterRules = items

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (d dedicatedAuditLogFilterRulesDataSource) retrieveAuditLogFilterRules(ctx context.Context, clusterId string) ([]dedicated.V1beta1AuditLogFilterRule, error) {
	var items []dedicated.V1beta1AuditLogFilterRule
	pageSizeInt32 := int32(DefaultPageSize)
	var pageToken *string
	for {
		auditLogFilterRules, err := d.provider.DedicatedClient.ListAuditLogFilterRules(ctx, clusterId, &pageSizeInt32, pageToken)
		if err != nil {
			return nil, errors.Trace(err)
		}
		items = append(items, auditLogFilterRules.AuditLogFilterRules...)

		pageToken = auditLogFilterRules.NextPageToken
		if IsNilOrEmpty(pageToken) {
			break
		}
	}
	return items, nil
}
