package provider

import (
	"context"
	"fmt"
	"time"

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
	_ resource.Resource = &DedicatedNetworkContainerResource{}
)

type DedicatedNetworkContainerResource struct {
	provider *tidbcloudProvider
}

type DedicatedNetworkContainerResourceData struct {
	ProjectId          types.String `tfsdk:"project_id"`
	NetworkContainerId types.String `tfsdk:"network_container_id"`
	RegionId           types.String `tfsdk:"region_id"`
	CidrNotion         types.String `tfsdk:"cidr_notion"`
	State              types.String `tfsdk:"state"`
	CloudProvider      types.String `tfsdk:"cloud_provider"`
	RegionDisplayName  types.String `tfsdk:"region_display_name"`
	VpcId              types.String `tfsdk:"vpc_id"`
	Labels             types.Map    `tfsdk:"labels"`
}

func NewDedicatedNetworkContainerResource() resource.Resource {
	return &DedicatedNetworkContainerResource{}
}

func (r *DedicatedNetworkContainerResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dedicated_network_container"
}

func (r *DedicatedNetworkContainerResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "dedicated network container resource.",
		Attributes: map[string]schema.Attribute{
			"project_id": schema.StringAttribute{
				Description: "The project ID for the network container",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
			},
			"network_container_id": schema.StringAttribute{
				Description: "The ID of the network container",
				Computed:    true,
			},
			"region_id": schema.StringAttribute{
				Description: "The region ID for the network container",
				Required:    true,
			},
			"cidr_notion": schema.StringAttribute{
				Description: "CIDR notation for the network container",
				Required:    true,
			},
			"state": schema.StringAttribute{
				MarkdownDescription: "The state of the network container",
				Computed:            true,
			},
			"cloud_provider": schema.StringAttribute{
				MarkdownDescription: "The cloud provider for the network container",
				Computed:            true,
			},
			"region_display_name": schema.StringAttribute{
				MarkdownDescription: "The display name of the region",
				Computed:            true,
			},
			"vpc_id": schema.StringAttribute{
				MarkdownDescription: "The VPC ID for the network container",
				Computed:            true,
			},
			"labels": schema.MapAttribute{
				Description: "The labels for the network container",
				Computed:    true,
				ElementType: types.StringType,
			},
		},
	}
}

func (r *DedicatedNetworkContainerResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *DedicatedNetworkContainerResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if !r.provider.configured {
		resp.Diagnostics.AddError(
			"Provider not configured",
			"The provider hasn't been configured before apply, likely because it depends on an unknown value from another resource. This leads to weird stuff happening, so we'd prefer if you didn't do that. Thanks!",
		)
		return
	}

	var data DedicatedNetworkContainerResourceData
	diags := req.Plan.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Trace(ctx, "create dedicated_network_container_resource")
	body, err := buildCreateDedicatedNetworkContainerBody(data)
	if err != nil {
		resp.Diagnostics.AddError("Create Error", fmt.Sprintf("Unable to build create body, got error: %s", err))
		return
	}
	networkContainer, err := r.provider.DedicatedClient.CreateNetworkContainer(ctx, &body)
	if err != nil {
		resp.Diagnostics.AddError("Create Error", fmt.Sprintf("Unable to call CreateNetworkContainer, got error: %s", err))
		return
	}

	networkContainerId := *networkContainer.NetworkContainerId
	data.NetworkContainerId = types.StringValue(networkContainerId)
	tflog.Info(ctx, "wait dedicated network container ready")
	networkContainer, err = WaitDedicatedNetworkContainerReady(ctx, clusterCreateTimeout, clusterCreateInterval, networkContainerId, r.provider.DedicatedClient)
	if err != nil {
		resp.Diagnostics.AddError(
			"Dedicated network container creation failed",
			fmt.Sprintf("Dedicated network container is not ready, get error: %s", err),
		)
		return
	}
	refreshDedicatedNetworkContainerResourceData(ctx, networkContainer, &data)

	// save into the Terraform state.
	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (r *DedicatedNetworkContainerResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data DedicatedNetworkContainerResourceData
	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, fmt.Sprintf("read dedicated_network_container_resource network_container_id: %s", data.NetworkContainerId.ValueString()))

	// call read api
	tflog.Trace(ctx, "read dedicated_network_container_resource")
	networkContainer, err := r.provider.DedicatedClient.GetNetworkContainer(ctx, data.NetworkContainerId.ValueString())
	if err != nil {
		tflog.Error(ctx, fmt.Sprintf("Unable to call GetNetworkContainer, error: %s", err))
		return
	}
	refreshDedicatedNetworkContainerResourceData(ctx, networkContainer, &data)

	// save into the Terraform state.
	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (r *DedicatedNetworkContainerResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Update Error", "Update is not supported for dedicated network container")
	return
}

func (r *DedicatedNetworkContainerResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	resp.Diagnostics.AddError("Delete Error", "Network container can not be deleted")
	return
}

func buildCreateDedicatedNetworkContainerBody(data DedicatedNetworkContainerResourceData) (dedicated.V1beta1NetworkContainer, error) {
	regionId := data.RegionId.ValueString()
	cidrNotion := data.CidrNotion.ValueString()
	labels := make(map[string]string)
	if IsKnown(data.ProjectId) {
		labels[LabelsKeyProjectId] = data.ProjectId.ValueString()
	}
	return dedicated.V1beta1NetworkContainer{
		RegionId:   regionId,
		CidrNotion: &cidrNotion,
		Labels:     &labels,
	}, nil
}

func refreshDedicatedNetworkContainerResourceData(ctx context.Context, networkContainer *dedicated.V1beta1NetworkContainer, data *DedicatedNetworkContainerResourceData) {
	labels, diag := types.MapValueFrom(ctx, types.StringType, *networkContainer.Labels)
	if diag.HasError() {
		return
	}
	data.NetworkContainerId = types.StringValue(*networkContainer.NetworkContainerId)
	data.State = types.StringValue(string(*networkContainer.State))
	data.CloudProvider = types.StringValue(string(*networkContainer.CloudProvider))
	data.RegionDisplayName = types.StringValue(*networkContainer.RegionDisplayName)
	data.VpcId = types.StringValue(*networkContainer.VpcId)
	data.ProjectId = types.StringValue((*networkContainer.Labels)[LabelsKeyProjectId])
	data.Labels = labels
}

func WaitDedicatedNetworkContainerReady(ctx context.Context, timeout time.Duration, interval time.Duration, networkContainerId string,
	client tidbcloud.TiDBCloudDedicatedClient) (*dedicated.V1beta1NetworkContainer, error) {
	stateConf := &retry.StateChangeConf{
		Pending: []string{
			string(dedicated.V1BETA1NETWORKCONTAINERSTATE_INACTIVE),
		},
		Target: []string{
			string(dedicated.V1BETA1NETWORKCONTAINERSTATE_ACTIVE),
		},
		Timeout:      timeout,
		MinTimeout:   500 * time.Millisecond,
		PollInterval: interval,
		Refresh:      dedicatedNetworkContainerStateRefreshFunc(ctx, networkContainerId, client),
	}

	outputRaw, err := stateConf.WaitForStateContext(ctx)

	if output, ok := outputRaw.(*dedicated.V1beta1NetworkContainer); ok {
		return output, err
	}
	return nil, err
}

func dedicatedNetworkContainerStateRefreshFunc(ctx context.Context, networkContainerId string,
	client tidbcloud.TiDBCloudDedicatedClient) retry.StateRefreshFunc {
	return func() (interface{}, string, error) {
		tflog.Trace(ctx, "Waiting for dedicated network container ready")
		networkContainer, err := client.GetNetworkContainer(ctx, networkContainerId)
		if err != nil {
			return nil, "", err
		}
		return networkContainer, string(*networkContainer.State), nil
	}
}
