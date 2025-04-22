package provider

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	branchV1beta1 "github.com/tidbcloud/tidbcloud-cli/pkg/tidbcloud/v1beta1/serverless/branch"
)

type serverlessBranchDataSourceData struct {
	ClusterId         types.String `tfsdk:"cluster_id"`
	BranchId          types.String `tfsdk:"branch_id"`
	DisplayName       types.String `tfsdk:"display_name"`
	ParentId          types.String `tfsdk:"parent_id"`
	Endpoints         *endpoints   `tfsdk:"endpoints"`
	State             types.String `tfsdk:"state"`
	UserPrefix        types.String `tfsdk:"user_prefix"`
	Usage             *usage       `tfsdk:"usage"`
	CreatedBy         types.String `tfsdk:"created_by"`
	CreateTime        types.String `tfsdk:"create_time"`
	UpdateTime        types.String `tfsdk:"update_time"`
	ParentDisplayName types.String `tfsdk:"parent_display_name"`
	ParentTimestamp   types.String `tfsdk:"parent_timestamp"`
	Annotations       types.Map    `tfsdk:"annotations"`
}

var _ datasource.DataSource = &serverlessBranchDataSource{}

type serverlessBranchDataSource struct {
	provider *tidbcloudProvider
}

func NewServerlessBranchDataSource() datasource.DataSource {
	return &serverlessBranchDataSource{}
}

func (d *serverlessBranchDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_serverless_branch"
}

func (d *serverlessBranchDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	var ok bool
	if d.provider, ok = req.ProviderData.(*tidbcloudProvider); !ok {
		resp.Diagnostics.AddError("Internal provider error",
			fmt.Sprintf("Error in Configure: expected %T but got %T", tidbcloudProvider{}, req.ProviderData))
	}
}

func (d *serverlessBranchDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "serverless branch data source",
		Attributes: map[string]schema.Attribute{
			"cluster_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the cluster.",
				Required:            true,
			},
			"branch_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the branch.",
				Required:            true,
			},
			"display_name": schema.StringAttribute{
				MarkdownDescription: "The display name of the branch.",
				Computed:            true,
			},
			"parent_id": schema.StringAttribute{
				MarkdownDescription: "The parent ID of the branch.",
				Computed:            true,
			},
			"endpoints": schema.SingleNestedAttribute{
				MarkdownDescription: "The endpoints for connecting to the branch.",
				Computed:            true,
				Attributes: map[string]schema.Attribute{
					"public": schema.SingleNestedAttribute{
						MarkdownDescription: "The public endpoint for connecting to the branch.",
						Computed:            true,
						Attributes: map[string]schema.Attribute{
							"host": schema.StringAttribute{
								MarkdownDescription: "The host of the public endpoint.",
								Computed:            true,
							},
							"port": schema.Int32Attribute{
								MarkdownDescription: "The port of the public endpoint.",
								Computed:            true,
							},
							"disabled": schema.BoolAttribute{
								MarkdownDescription: "Whether the public endpoint is disabled.",
								Computed:            true,
							},
						},
					},
					"private": schema.SingleNestedAttribute{
						MarkdownDescription: "The private endpoint for connecting to the branch.",
						Computed:            true,
						Attributes: map[string]schema.Attribute{
							"host": schema.StringAttribute{
								MarkdownDescription: "The host of the private endpoint.",
								Computed:            true,
							},
							"port": schema.Int32Attribute{
								MarkdownDescription: "The port of the private endpoint.",
								Computed:            true,
							},
							"aws": schema.SingleNestedAttribute{
								MarkdownDescription: "Message for AWS PrivateLink information.",
								Computed:            true,
								Attributes: map[string]schema.Attribute{
									"service_name": schema.StringAttribute{
										MarkdownDescription: "The AWS service name for private access.",
										Computed:            true,
									},
									"availability_zone": schema.ListAttribute{
										MarkdownDescription: "The availability zones that the service is available in.",
										Computed:            true,
										ElementType:         types.StringType,
									},
								},
							},
						},
					},
				},
			},
			"created_by": schema.StringAttribute{
				MarkdownDescription: "The email of the creator of the branch.",
				Computed:            true,
			},
			"create_time": schema.StringAttribute{
				MarkdownDescription: "The time the branch was created.",
				Computed:            true,
			},
			"update_time": schema.StringAttribute{
				MarkdownDescription: "The time the branch was last updated.",
				Computed:            true,
			},
			"user_prefix": schema.StringAttribute{
				MarkdownDescription: "The unique prefix in SQL user name.",
				Computed:            true,
			},
			"state": schema.StringAttribute{
				MarkdownDescription: "The state of the branch.",
				Computed:            true,
			},
			"usage": schema.SingleNestedAttribute{
				MarkdownDescription: "The usage of the branch.",
				Computed:            true,
				Attributes: map[string]schema.Attribute{
					"request_unit": schema.StringAttribute{
						MarkdownDescription: "The request unit of the branch.",
						Computed:            true,
					},
					"row_based_storage": schema.Int64Attribute{
						MarkdownDescription: "The row-based storage of the branch.",
						Computed:            true,
					},
					"columnar_storage": schema.Int64Attribute{
						MarkdownDescription: "The columnar storage of the branch.",
						Computed:            true,
					},
				},
			},
			"parent_display_name": schema.StringAttribute{
				MarkdownDescription: "The display name of the parent.",
				Computed:            true,
			},
			"parent_timestamp": schema.StringAttribute{
				MarkdownDescription: "The timestamp of the parent. (RFC3339 format, e.g., 2024-01-01T00:00:00Z)",
				Computed:            true,
			},
			"annotations": schema.MapAttribute{
				MarkdownDescription: "The annotations of the branch.",
				Computed:            true,
				ElementType:         types.StringType,
			},
		},
	}
}

func (d *serverlessBranchDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data serverlessBranchDataSourceData
	diags := req.Config.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Trace(ctx, "read serverless branch data source")
	branch, err := d.provider.ServerlessClient.GetBranch(ctx, data.ClusterId.ValueString(), data.BranchId.ValueString(), branchV1beta1.BRANCHSERVICEGETBRANCHVIEWPARAMETER_FULL)
	if err != nil {
		resp.Diagnostics.AddError("Read Error", fmt.Sprintf("Unable to call GetBranch, got error: %s", err))
		return
	}

	annotations, diag := types.MapValueFrom(ctx, types.StringType, *branch.Annotations)
	if diag.HasError() {
		diags.AddError("Read Error", "unable to convert annotations")
		return
	}
	data.DisplayName = types.StringValue(branch.DisplayName)
	data.ParentId = types.StringValue(*branch.ParentId)
	data.ParentDisplayName = types.StringValue(*branch.ParentDisplayName)
	data.ParentTimestamp = types.StringValue(branch.ParentTimestamp.Get().Format(time.RFC3339))

	e := branch.Endpoints
	var pe private
	if e.Private.Aws != nil {
		awsAvailabilityZone, diags := types.ListValueFrom(ctx, types.StringType, e.Private.Aws.AvailabilityZone)
		if diags.HasError() {
			return
		}
		pe = private{
			Host: types.StringValue(*e.Private.Host),
			Port: types.Int32Value(*e.Private.Port),
			AWS: &aws{
				ServiceName:      types.StringValue(*e.Private.Aws.ServiceName),
				AvailabilityZone: awsAvailabilityZone,
			},
		}
	}

	data.Endpoints = &endpoints{
		Public: &public{
			Host:     types.StringValue(*e.Public.Host),
			Port:     types.Int32Value(*e.Public.Port),
			Disabled: types.BoolValue(*e.Public.Disabled),
		},
		Private: &pe,
	}

	data.CreatedBy = types.StringValue(*branch.CreatedBy)
	data.CreateTime = types.StringValue(branch.CreateTime.String())
	data.UpdateTime = types.StringValue(branch.UpdateTime.String())
	data.UserPrefix = types.StringValue(*branch.UserPrefix.Get())
	data.State = types.StringValue(string(*branch.State))

	u := branch.Usage
	data.Usage = &usage{
		RequestUnit:     types.StringValue(*u.RequestUnit),
		RowBasedStorage: types.Int64Value(int64(*u.RowStorage)),
		ColumnarStorage: types.Int64Value(int64(*u.ColumnarStorage)),
	}

	data.Annotations = annotations

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}
