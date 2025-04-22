package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/juju/errors"
	branchV1beta1 "github.com/tidbcloud/tidbcloud-cli/pkg/tidbcloud/v1beta1/serverless/branch"
)

type serverlessBranchesDataSourceData struct {
	ClusterId types.String           `tfsdk:"cluster_id"`
	Branches  []serverlessBranchItem `tfsdk:"branches"`
}

type serverlessBranchItem struct {
	BranchId          types.String `tfsdk:"branch_id"`
	DisplayName       types.String `tfsdk:"display_name"`
	ParentId          types.String `tfsdk:"parent_id"`
	Endpoints         *endpoints   `tfsdk:"endpoints"`
	State             types.String `tfsdk:"state"`
	UserPrefix        types.String `tfsdk:"user_prefix"`
	CreatedBy         types.String `tfsdk:"created_by"`
	CreateTime        types.String `tfsdk:"create_time"`
	UpdateTime        types.String `tfsdk:"update_time"`
	ParentDisplayName types.String `tfsdk:"parent_display_name"`
	ParentTimestamp   types.String `tfsdk:"parent_timestamp"`
	Annotations       types.Map    `tfsdk:"annotations"`
}

var _ datasource.DataSource = &serverlessBranchesDataSource{}

type serverlessBranchesDataSource struct {
	provider *tidbcloudProvider
}

func NewServerlessBranchesDataSource() datasource.DataSource {
	return &serverlessBranchesDataSource{}
}

func (d *serverlessBranchesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_serverless_branches"
}

func (d *serverlessBranchesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	var ok bool
	if d.provider, ok = req.ProviderData.(*tidbcloudProvider); !ok {
		resp.Diagnostics.AddError("Internal provider error",
			fmt.Sprintf("Error in Configure: expected %T but got %T", tidbcloudProvider{}, req.ProviderData))
	}
}

func (d *serverlessBranchesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "serverless branches data source",
		Attributes: map[string]schema.Attribute{
			"cluster_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the cluster.",
				Required:            true,
			},
			"branches": schema.ListNestedAttribute{
				MarkdownDescription: "The branches of the cluster.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"branch_id": schema.StringAttribute{
							MarkdownDescription: "The ID of the branch.",
							Computed:            true,
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
											Optional:            true,
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
				},
			},
		},
	}
}

func (d *serverlessBranchesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data serverlessBranchesDataSourceData
	diags := req.Config.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Trace(ctx, "read serverless branches data source")
	branches, err := d.retrieveBranches(ctx, data.ClusterId.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Read Error", fmt.Sprintf("Unable to call ListBranches, got error: %s", err))
		return
	}
	var items []serverlessBranchItem
	for _, branch := range branches {
		var b serverlessBranchItem
		annotations, diag := types.MapValueFrom(ctx, types.StringType, *branch.Annotations)
		if diag.HasError() {
			diags.AddError("Read Error", "unable to convert annotations")
			return
		}
		b.DisplayName = types.StringValue(branch.DisplayName)

		e := branch.Endpoints
		var pe private
		if e.Private.Aws != nil {
			awsAvailabilityZone, diag := types.ListValueFrom(ctx, types.StringType, e.Private.Aws.AvailabilityZone)
			if diag.HasError() {
				diags.AddError("Read Error", "unable to convert aws availability zone")
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

		b.Endpoints = &endpoints{
			Public: &public{
				Host:     types.StringValue(*e.Public.Host),
				Port:     types.Int32Value(*e.Public.Port),
				Disabled: types.BoolValue(*e.Public.Disabled),
			},
			Private: &pe,
		}

		b.BranchId = types.StringValue(*branch.BranchId)
		b.ParentId = types.StringValue(*branch.ParentId)
		b.ParentDisplayName = types.StringValue(*branch.ParentDisplayName)
		b.ParentTimestamp = types.StringValue(branch.ParentTimestamp.Get().String())
		b.CreatedBy = types.StringValue(*branch.CreatedBy)
		b.CreateTime = types.StringValue(branch.CreateTime.String())
		b.UpdateTime = types.StringValue(branch.UpdateTime.String())
		b.UserPrefix = types.StringValue(*branch.UserPrefix.Get())
		b.State = types.StringValue(string(*branch.State))

		b.Annotations = annotations
		items = append(items, b)
	}

	data.Branches = items
	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (d serverlessBranchesDataSource) retrieveBranches(ctx context.Context, clusterId string) ([]branchV1beta1.Branch, error) {
	var items []branchV1beta1.Branch
	pageSizeInt32 := int32(DefaultPageSize)
	var pageToken *string
	for {

		branches, err := d.provider.ServerlessClient.ListBranches(ctx, clusterId, &pageSizeInt32, pageToken)
		if err != nil {
			return nil, errors.Trace(err)
		}
		items = append(items, branches.Branches...)

		pageToken = branches.NextPageToken
		if IsNilOrEmpty(pageToken) {
			break
		}
	}
	return items, nil
}
