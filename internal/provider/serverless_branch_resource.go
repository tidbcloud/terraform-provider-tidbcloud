package provider

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"github.com/tidbcloud/terraform-provider-tidbcloud/tidbcloud"
	branchV1beta1 "github.com/tidbcloud/tidbcloud-cli/pkg/tidbcloud/v1beta1/serverless/branch"
)

const (
	serverlessBranchCreateTimeout  = 600 * time.Second
	serverlessBranchCreateInterval = 10 * time.Second
)

type serverlessBranchResourceData struct {
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

type serverlessBranchResource struct {
	provider *tidbcloudProvider
}

func NewServerlessBranchResource() resource.Resource {
	return &serverlessBranchResource{}
}

func (r *serverlessBranchResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_serverless_branch"
}

func (r *serverlessBranchResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	var ok bool
	if r.provider, ok = req.ProviderData.(*tidbcloudProvider); !ok {
		resp.Diagnostics.AddError("Internal provider error",
			fmt.Sprintf("Error in Configure: expected %T but got %T", tidbcloudProvider{}, req.ProviderData))
	}
}

func (r *serverlessBranchResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "serverless cluster resource",
		Attributes: map[string]schema.Attribute{
			"cluster_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the cluster.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"branch_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the branch.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"display_name": schema.StringAttribute{
				MarkdownDescription: "The display name of the cluster.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"parent_id": schema.StringAttribute{
				MarkdownDescription: "The parent ID of the branch.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
			},
			"endpoints": schema.SingleNestedAttribute{
				MarkdownDescription: "The endpoints for connecting to the branch.",
				Computed:            true,
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.UseStateForUnknown(),
				},
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
						PlanModifiers: []planmodifier.Object{
							objectplanmodifier.UseStateForUnknown(),
						},
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
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"create_time": schema.StringAttribute{
				MarkdownDescription: "The time the branch was created.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"update_time": schema.StringAttribute{
				MarkdownDescription: "The time the branch was last updated.",
				Computed:            true,
			},
			"user_prefix": schema.StringAttribute{
				MarkdownDescription: "The unique prefix in SQL user name.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"state": schema.StringAttribute{
				MarkdownDescription: "The state of the branch.",
				Computed:            true,
			},
			"usage": schema.SingleNestedAttribute{
				MarkdownDescription: "The usage of the branch.",
				Computed:            true,
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.UseStateForUnknown(),
				},
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
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"parent_timestamp": schema.StringAttribute{
				MarkdownDescription: "The timestamp of the parent. (RFC3339 format, e.g., 2024-01-01T00:00:00Z)",
				Computed:            true,
				Optional:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
			},
			"annotations": schema.MapAttribute{
				MarkdownDescription: "The annotations of the branch.",
				Computed:            true,
				ElementType:         types.StringType,
				PlanModifiers: []planmodifier.Map{
					mapplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r serverlessBranchResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if !r.provider.configured {
		resp.Diagnostics.AddError(
			"Provider not configured",
			"The provider hasn't been configured before apply, likely because it depends on an unknown value from another resource. This leads to weird stuff happening, so we'd prefer if you didn't do that. Thanks!",
		)
		return
	}

	// get data from config
	var data serverlessBranchResourceData
	diags := req.Config.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Trace(ctx, "create serverless_branch_resource")
	body, err := buildCreateServerlessBranchBody(data)
	if err != nil {
		resp.Diagnostics.AddError("Create Error", fmt.Sprintf("Unable to build CreateBranch body, got error: %s", err))
		return
	}
	branch, err := r.provider.ServerlessClient.CreateBranch(ctx, data.ClusterId.ValueString(), &body)
	if err != nil {
		resp.Diagnostics.AddError("Create Error", fmt.Sprintf("Unable to call CreateBranch, got error: %s", err))
		return
	}

	branchId := *branch.BranchId
	tflog.Info(ctx, "wait serverless branch ready")
	_, err = WaitServerlessBranchReady(ctx, serverlessBranchCreateTimeout, serverlessBranchCreateInterval, data.ClusterId.ValueString(), branchId, r.provider.ServerlessClient)
	if err != nil {
		resp.Diagnostics.AddError(
			"Branch creation failed",
			fmt.Sprintf("Branch is not ready, get error: %s", err),
		)
		return
	}
	branch, err = r.provider.ServerlessClient.GetBranch(ctx, data.ClusterId.ValueString(), branchId, branchV1beta1.BRANCHSERVICEGETBRANCHVIEWPARAMETER_FULL)
	if err != nil {
		tflog.Error(ctx, fmt.Sprintf("Unable to call GetBranch, error: %s", err))
		return
	}
	data.BranchId = types.StringValue(branchId)
	err = refreshServerlessBranchResourceData(ctx, branch, &data)
	if err != nil {
		resp.Diagnostics.AddError("Refresh Error", fmt.Sprintf("Unable to refresh serverless cluster resource data, got error: %s", err))
		return
	}

	// save into the Terraform state.
	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (r serverlessBranchResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// get data from state
	var data serverlessBranchResourceData
	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Debug(ctx, fmt.Sprintf("read serverless_branch_resource clusterid: %s", data.ClusterId.ValueString()))

	// call read api
	tflog.Trace(ctx, "read serverless_branch_resource")
	branch, err := r.provider.ServerlessClient.GetBranch(ctx, data.ClusterId.ValueString(), data.BranchId.ValueString(), branchV1beta1.BRANCHSERVICEGETBRANCHVIEWPARAMETER_FULL)
	if err != nil {
		tflog.Error(ctx, fmt.Sprintf("Unable to call GetBranch, error: %s", err))
		return
	}
	err = refreshServerlessBranchResourceData(ctx, branch, &data)
	if err != nil {
		resp.Diagnostics.AddError("Refresh Error", fmt.Sprintf("Unable to refresh serverless branch resource data, got error: %s", err))
		return
	}

	// save into the Terraform state.
	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (r serverlessBranchResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var clusterId string
	var branchId string

	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("cluster_id"), &clusterId)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("branch_id"), &branchId)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Trace(ctx, "delete serverless_branch_resource")
	_, err := r.provider.ServerlessClient.DeleteBranch(ctx, clusterId, branchId)
	if err != nil {
		resp.Diagnostics.AddError("Delete Error", fmt.Sprintf("Unable to call DeleteBranch, got error: %s", err))
		return
	}
}

func (r serverlessBranchResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
}

func (r serverlessBranchResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	idParts := strings.Split(req.ID, ",")

	if len(idParts) != 2 || idParts[0] == "" || idParts[1] == "" {
		resp.Diagnostics.AddError(
			"Unexpected Import Identifier",
			fmt.Sprintf("Expected import identifier with format: cluster_id, branch_id. Got: %q", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("cluster_id"), idParts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("branch_id"), idParts[1])...)
}

func buildCreateServerlessBranchBody(data serverlessBranchResourceData) (branchV1beta1.Branch, error) {
	displayName := data.DisplayName.ValueString()
	parentId := data.ParentId.ValueString()
	body := branchV1beta1.Branch{
		DisplayName: displayName,
		ParentId:    &parentId,
	}

	if data.ParentTimestamp.ValueString() != "" {
		parentTimestampStr := data.ParentTimestamp.ValueString()
		parentTimestamp, err := time.Parse(time.RFC3339, parentTimestampStr)
		if err != nil {
			return branchV1beta1.Branch{}, err
		}
		body.ParentTimestamp = *branchV1beta1.NewNullableTime(&parentTimestamp)
	}

	return body, nil
}

func refreshServerlessBranchResourceData(ctx context.Context, resp *branchV1beta1.Branch, data *serverlessBranchResourceData) error {
	annotations, diags := types.MapValueFrom(ctx, types.StringType, *resp.Annotations)
	if diags.HasError() {
		return errors.New("unable to convert annotations")
	}

	data.DisplayName = types.StringValue(resp.DisplayName)
	data.ParentTimestamp = types.StringValue(resp.ParentTimestamp.Get().Format(time.RFC3339))
	data.ParentDisplayName = types.StringValue(*resp.ParentDisplayName)
	data.ParentId = types.StringValue(*resp.ParentId)

	e := resp.Endpoints
	var pe private
	if e.Private.Aws != nil {
		awsAvailabilityZone, diag := types.ListValueFrom(ctx, types.StringType, e.Private.Aws.AvailabilityZone)
		if diag.HasError() {
			return errors.New("unable to convert aws availability zone")
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

	data.CreatedBy = types.StringValue(*resp.CreatedBy)
	data.CreateTime = types.StringValue(resp.CreateTime.String())
	data.UpdateTime = types.StringValue(resp.UpdateTime.String())
	data.UserPrefix = types.StringValue(*resp.UserPrefix.Get())
	data.State = types.StringValue(string(*resp.State))

	u := resp.Usage
	data.Usage = &usage{
		RequestUnit:     types.StringValue(*u.RequestUnit),
		RowBasedStorage: types.Int64Value(int64(*u.RowStorage)),
		ColumnarStorage: types.Int64Value(int64(*u.ColumnarStorage)),
	}

	data.Annotations = annotations
	return nil
}

func WaitServerlessBranchReady(ctx context.Context, timeout time.Duration, interval time.Duration, clusterId string, branchId string,
	client tidbcloud.TiDBCloudServerlessClient) (*branchV1beta1.Branch, error) {
	stateConf := &retry.StateChangeConf{
		Pending: []string{
			string(branchV1beta1.BRANCHSTATE_CREATING),
			string(branchV1beta1.BRANCHSTATE_RESTORING),
		},
		Target: []string{
			string(branchV1beta1.BRANCHSTATE_ACTIVE),
			string(branchV1beta1.BRANCHSTATE_DELETED),
			string(branchV1beta1.BRANCHSTATE_MAINTENANCE),
		},
		Timeout:      timeout,
		MinTimeout:   500 * time.Millisecond,
		PollInterval: interval,
		Refresh:      serverlessBranchStateRefreshFunc(ctx, clusterId, branchId, client),
	}

	outputRaw, err := stateConf.WaitForStateContext(ctx)

	if output, ok := outputRaw.(*branchV1beta1.Branch); ok {
		return output, err
	}
	return nil, err
}

func serverlessBranchStateRefreshFunc(ctx context.Context, clusterId string, branchId string,
	client tidbcloud.TiDBCloudServerlessClient) retry.StateRefreshFunc {
	return func() (interface{}, string, error) {
		tflog.Trace(ctx, "Waiting for serverless cluster ready")
		branch, err := client.GetBranch(ctx, clusterId, branchId, branchV1beta1.BRANCHSERVICEGETBRANCHVIEWPARAMETER_BASIC)
		if err != nil {
			return nil, "", err
		}
		return branch, string(*branch.State), nil
	}
}
