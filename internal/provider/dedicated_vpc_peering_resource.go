package provider

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"github.com/tidbcloud/terraform-provider-tidbcloud/tidbcloud"
	"github.com/tidbcloud/tidbcloud-cli/pkg/tidbcloud/v1beta1/dedicated"
)

var (
	_ resource.Resource = &DedicatedVpcPeeringResource{}
)

type DedicatedVpcPeeringResource struct {
	provider *tidbcloudProvider
}

type DedicatedVpcPeeringResourceData struct {
	ProjectId                 types.String `tfsdk:"project_id"`
	VpcPeeringId              types.String `tfsdk:"vpc_peering_id"`
	TiDBCloudRegionId         types.String `tfsdk:"tidb_cloud_region_id"`
	TiDBCloudCloudProvider    types.String `tfsdk:"tidb_cloud_cloud_provider"`
	TiDBCloudAccountId        types.String `tfsdk:"tidb_cloud_account_id"`
	TiDBCloudVpcId            types.String `tfsdk:"tidb_cloud_vpc_id"`
	TiDBCloudVpcCidr          types.String `tfsdk:"tidb_cloud_vpc_cidr"`
	CustomerRegionId          types.String `tfsdk:"customer_region_id"`
	CustomerAccountId         types.String `tfsdk:"customer_account_id"`
	CustomerVpcId             types.String `tfsdk:"customer_vpc_id"`
	CustomerVpcCidr           types.String `tfsdk:"customer_vpc_cidr"`
	State                     types.String `tfsdk:"state"`
	AWSVpcPeeringConnectionId types.String `tfsdk:"aws_vpc_peering_connection_id"`
	Labels                    types.Map    `tfsdk:"labels"`
}

func NewDedicatedVpcPeeringResource() resource.Resource {
	return &DedicatedVpcPeeringResource{}
}

func (r *DedicatedVpcPeeringResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dedicated_vpc_peering"
}

func (r *DedicatedVpcPeeringResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Resource for Dedicated VPC Peering.",
		Attributes: map[string]schema.Attribute{
			"project_id": schema.StringAttribute{
				Description: "The project ID for the VPC Peering",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
			},
			"vpc_peering_id": schema.StringAttribute{
				Description: "The ID of the VPC Peering",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"tidb_cloud_region_id": schema.StringAttribute{
				Description: "The region ID of the TiDB Cloud",
				Required:    true,
			},
			"tidb_cloud_cloud_provider": schema.StringAttribute{
				Description: "The cloud provider of the TiDB Cloud",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"tidb_cloud_account_id": schema.StringAttribute{
				Description: "The account ID of the TiDB Cloud",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"tidb_cloud_vpc_id": schema.StringAttribute{
				Description: "The VPC ID of the TiDB Cloud",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"tidb_cloud_vpc_cidr": schema.StringAttribute{
				Description: "The VPC CIDR of the TiDB Cloud",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"customer_region_id": schema.StringAttribute{
				Description: "The region ID of the AWS VPC",
				Required:    true,
			},
			"customer_account_id": schema.StringAttribute{
				Description: "The account ID of the AWS VPC",
				Required:    true,
			},
			"customer_vpc_id": schema.StringAttribute{
				Description: "The ID of the AWS VPC",
				Required:    true,
			},
			"customer_vpc_cidr": schema.StringAttribute{
				Description: "The VPC CIDR of the AWS VPC",
				Required:    true,
			},
			"state": schema.StringAttribute{
				Description: "The state of the VPC Peering",
				Computed:    true,
			},
			"aws_vpc_peering_connection_id": schema.StringAttribute{
				Description: "The ID of the AWS VPC Peering Connection",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"labels": schema.MapAttribute{
				Description: "The labels for the vpc peering",
				Computed:    true,
				ElementType: types.StringType,
			},
		},
	}
}

func (r *DedicatedVpcPeeringResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *DedicatedVpcPeeringResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if !r.provider.configured {
		resp.Diagnostics.AddError(
			"Provider not configured",
			"The provider hasn't been configured before apply, likely because it depends on an unknown value from another resource. This leads to weird stuff happening, so we'd prefer if you didn't do that. Thanks!",
		)
		return
	}

	var data DedicatedVpcPeeringResourceData
	diags := req.Plan.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Trace(ctx, "create dedicated_vpc_peering_resource")
	body := buildCreateDedicatedVpcPeeringBody(data)
	VpcPeering, err := r.provider.DedicatedClient.CreateVPCPeering(ctx, &body)
	if err != nil {
		resp.Diagnostics.AddError("Create Error", fmt.Sprintf("Unable to call CreateVpcPeering, got error: %s", err))
		return
	}

	VpcPeeringId := *VpcPeering.VpcPeeringId
	data.VpcPeeringId = types.StringValue(VpcPeeringId)
	tflog.Info(ctx, "wait dedicated vpc peering ready")
	VpcPeering, err = WaitDedicatedVpcPeeringReady(ctx, clusterCreateTimeout, clusterCreateInterval, VpcPeeringId, r.provider.DedicatedClient)
	if err != nil {
		resp.Diagnostics.AddError(
			"Dedicated vpc peering creation failed",
			fmt.Sprintf("Dedicated vpc peering is not ready, get error: %s", err),
		)
		return
	}
	refreshDedicatedVpcPeeringResourceData(ctx, VpcPeering, &data)

	// save into the Terraform state.
	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (r *DedicatedVpcPeeringResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data DedicatedVpcPeeringResourceData
	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, fmt.Sprintf("read dedicated_vpc_peering_resource vpc_peering_id: %s", data.VpcPeeringId.ValueString()))

	// call read api
	tflog.Trace(ctx, "read dedicated_vpc_peering_resource")
	VpcPeering, err := r.provider.DedicatedClient.GetVPCPeering(ctx, data.VpcPeeringId.ValueString())
	if err != nil {
		tflog.Error(ctx, fmt.Sprintf("Unable to call GetVpcPeering, error: %s", err))
		return
	}
	refreshDedicatedVpcPeeringResourceData(ctx, VpcPeering, &data)

	// save into the Terraform state.
	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (r *DedicatedVpcPeeringResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Update Error", "Update is not supported for dedicated vpc peering")
	return
}

func (r *DedicatedVpcPeeringResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var vpcPeeringId string
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("vpc_peering_id"), &vpcPeeringId)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Trace(ctx, "delete dedicated_vpc_peering_resource")
	err := r.provider.DedicatedClient.DeleteVPCPeering(ctx, vpcPeeringId)
	if err != nil {
		resp.Diagnostics.AddError("Delete Error", fmt.Sprintf("Unable to call DeleteVpcPeering, got error: %s", err))
		return
	}
}

func (r *DedicatedVpcPeeringResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("vpc_peering_id"), req, resp)
}

func buildCreateDedicatedVpcPeeringBody(data DedicatedVpcPeeringResourceData) dedicated.Dedicatedv1beta1VpcPeering {
	customerRegionId := data.CustomerRegionId.ValueString()
	labels := make(map[string]string)
	if IsKnown(data.ProjectId) {
		labels[LabelsKeyProjectId] = data.ProjectId.ValueString()
	}
	return dedicated.Dedicatedv1beta1VpcPeering{
		TidbCloudRegionId: data.TiDBCloudRegionId.ValueString(),
		CustomerRegionId:  &customerRegionId,
		CustomerAccountId: data.CustomerAccountId.ValueString(),
		CustomerVpcId:     data.CustomerVpcId.ValueString(),
		CustomerVpcCidr:   data.CustomerVpcCidr.ValueString(),
		Labels:            &labels,
	}
}

func refreshDedicatedVpcPeeringResourceData(ctx context.Context, vpcPeering *dedicated.Dedicatedv1beta1VpcPeering, data *DedicatedVpcPeeringResourceData) {
	data.VpcPeeringId = types.StringValue(*vpcPeering.VpcPeeringId)
	data.State = types.StringValue(string(*vpcPeering.State))
	data.TiDBCloudCloudProvider = types.StringValue(string(*vpcPeering.TidbCloudCloudProvider))
	data.TiDBCloudAccountId = types.StringValue(*vpcPeering.TidbCloudAccountId)
	data.TiDBCloudVpcId = types.StringValue(*vpcPeering.TidbCloudVpcId)
	data.TiDBCloudVpcCidr = types.StringValue(*vpcPeering.TidbCloudVpcCidr)
	data.ProjectId = types.StringValue((*vpcPeering.Labels)[LabelsKeyProjectId])
	if vpcPeering.AwsVpcPeeringConnectionId.IsSet() {
		data.AWSVpcPeeringConnectionId = types.StringValue(*vpcPeering.AwsVpcPeeringConnectionId.Get())
	}
	labels, diag := types.MapValueFrom(ctx, types.StringType, *vpcPeering.Labels)
	if diag.HasError() {
		return
	}
	data.Labels = labels
}

func WaitDedicatedVpcPeeringReady(ctx context.Context, timeout time.Duration, interval time.Duration, VpcPeeringId string,
	client tidbcloud.TiDBCloudDedicatedClient) (*dedicated.Dedicatedv1beta1VpcPeering, error) {
	stateConf := &retry.StateChangeConf{
		Pending: []string{
			string(dedicated.DEDICATEDV1BETA1VPCPEERINGSTATE_PENDING),
		},
		Target: []string{
			string(dedicated.DEDICATEDV1BETA1VPCPEERINGSTATE_FAILED),
			string(dedicated.DEDICATEDV1BETA1VPCPEERINGSTATE_ACTIVE),
		},
		Timeout:      timeout,
		MinTimeout:   500 * time.Millisecond,
		PollInterval: interval,
		Refresh:      dedicatedVpcPeeringStateRefreshFunc(ctx, VpcPeeringId, client),
	}

	outputRaw, err := stateConf.WaitForStateContext(ctx)

	if output, ok := outputRaw.(*dedicated.Dedicatedv1beta1VpcPeering); ok {
		return output, err
	}
	return nil, err
}

func dedicatedVpcPeeringStateRefreshFunc(ctx context.Context, VpcPeeringId string,
	client tidbcloud.TiDBCloudDedicatedClient) retry.StateRefreshFunc {
	return func() (interface{}, string, error) {
		tflog.Trace(ctx, "Waiting for dedicated vpc peering ready")
		VpcPeering, err := client.GetVPCPeering(ctx, VpcPeeringId)
		if err != nil {
			return nil, "", err
		}
		return VpcPeering, string(*VpcPeering.State), nil
	}
}
