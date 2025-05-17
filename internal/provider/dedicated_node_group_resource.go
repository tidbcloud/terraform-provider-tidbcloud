package provider

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int32planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"github.com/tidbcloud/terraform-provider-tidbcloud/tidbcloud"
	"github.com/tidbcloud/tidbcloud-cli/pkg/tidbcloud/v1beta1/dedicated"
)

type dedicatedNodeGroupResourceData struct {
	ClusterId             types.String           `tfsdk:"cluster_id"`
	NodeSpecKey           types.String           `tfsdk:"node_spec_key"`
	NodeCount             types.Int32            `tfsdk:"node_count"`
	NodeGroupId           types.String           `tfsdk:"node_group_id"`
	DisplayName           types.String           `tfsdk:"display_name"`
	NodeSpecDisplayName   types.String           `tfsdk:"node_spec_display_name"`
	IsDefaultGroup        types.Bool             `tfsdk:"is_default_group"`
	State                 types.String           `tfsdk:"state"`
	Endpoints             []endpoint             `tfsdk:"endpoints"`
	TiProxySetting        *tiProxySetting        `tfsdk:"tiproxy_setting"`
	PublicEndpointSetting *publicEndpointSetting `tfsdk:"public_endpoint_setting"`
}

type dedicatedNodeGroupResource struct {
	provider *tidbcloudProvider
}

func NewDedicatedNodeGroupResource() resource.Resource {
	return &dedicatedNodeGroupResource{}
}

func (r *dedicatedNodeGroupResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dedicated_node_group"
}

func (r *dedicatedNodeGroupResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *dedicatedNodeGroupResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "dedicated node group resource",
		Attributes: map[string]schema.Attribute{
			"cluster_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the cluster.",
				Required:            true,
			},
			"node_spec_key": schema.StringAttribute{
				MarkdownDescription: "The key of the node spec.",
				Computed:            true,
			},
			"node_count": schema.Int32Attribute{
				MarkdownDescription: "The count of the nodes in the group.",
				Required:            true,
			},
			"node_group_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the node group.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"display_name": schema.StringAttribute{
				MarkdownDescription: "The display name of the node group.",
				Required:            true,
			},
			"node_spec_display_name": schema.StringAttribute{
				MarkdownDescription: "The display name of the node spec.",
				Computed:            true,
			},
			"is_default_group": schema.BoolAttribute{
				MarkdownDescription: "Whether the node group is the default group.",
				Computed:            true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"state": schema.StringAttribute{
				MarkdownDescription: "The state of the node group.",
				Computed:            true,
			},
			"endpoints": schema.ListNestedAttribute{
				MarkdownDescription: "The endpoints of the node group.",
				Computed:            true,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.UseStateForUnknown(),
				},
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"host": schema.StringAttribute{
							MarkdownDescription: "The host of the endpoint.",
							Computed:            true,
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.UseStateForUnknown(),
							},
						},
						"port": schema.Int32Attribute{
							MarkdownDescription: "The port of the endpoint.",
							Computed:            true,
							PlanModifiers: []planmodifier.Int32{
								int32planmodifier.UseStateForUnknown(),
							},
						},
						"connection_type": schema.StringAttribute{
							MarkdownDescription: "The connection type of the endpoint.",
							Computed:            true,
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.UseStateForUnknown(),
							},
						},
					},
				},
			},
			"tiproxy_setting": schema.SingleNestedAttribute{
				MarkdownDescription: "Settings for TiProxy nodes.",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"type": schema.StringAttribute{
						MarkdownDescription: "The type of TiProxy nodes." +
							"- SMALL: Low performance instance with 2 vCPUs and 4 GiB memory. Max QPS: 30, Max Data Traffic: 90 MiB/s." +
							"- LARGE: High performance instance with 8 vCPUs and 16 GiB memory. Max QPS: 100, Max Data Traffic: 300 MiB/s.",
						Optional: true,
					},
					"node_count": schema.Int32Attribute{
						MarkdownDescription: "The number of TiProxy nodes.",
						Optional:            true,
					},
				},
			},
			"public_endpoint_setting": schema.SingleNestedAttribute{
				MarkdownDescription: "Settings for public endpoints.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.UseStateForUnknown(),
				},
				Attributes: map[string]schema.Attribute{
					"enabled": schema.BoolAttribute{
						MarkdownDescription: "Whether public endpoints are enabled.",
						Optional:            true,
						Computed:            true,
						PlanModifiers: []planmodifier.Bool{
							boolplanmodifier.UseStateForUnknown(),
						},
					},
					"ip_access_list": schema.ListAttribute{
						MarkdownDescription: "IP access list for the public endpoint.",
						Optional:            true,
						Computed:            true,
						PlanModifiers: []planmodifier.List{
							listplanmodifier.UseStateForUnknown(),
						},
						ElementType: types.ObjectType{AttrTypes: ipAccessListItemAttrTypes},
					},
				},
			},
		},
	}
}

func (r dedicatedNodeGroupResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if !r.provider.configured {
		resp.Diagnostics.AddError(
			"Provider not configured",
			"The provider hasn't been configured before apply, likely because it depends on an unknown value from another resource. This leads to weird stuff happening, so we'd prefer if you didn't do that. Thanks!",
		)
		return
	}

	// get data from config
	var data dedicatedNodeGroupResourceData
	diags := req.Config.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Trace(ctx, "create dedicated_node_group_resource")
	body := buildCreateDedicatedNodeGroupBody(data)
	nodeGroup, err := r.provider.DedicatedClient.CreateTiDBNodeGroup(ctx, data.ClusterId.ValueString(), &body)
	if err != nil {
		resp.Diagnostics.AddError("Create Error", fmt.Sprintf("Unable to call CreateTiDBNodeGroup, got error: %s", err))
		return
	}
	// set tidbNodeGroupId. other computed attributes are not returned by create, they will be set when refresh
	nodeGroupId := *nodeGroup.TidbNodeGroupId
	data.NodeGroupId = types.StringValue(nodeGroupId)
	tflog.Info(ctx, "wait dedicated node group ready")
	// it's a workaround, tidb node group state is active at the beginning, so we need to wait for it to be modifying
	time.Sleep(1 * time.Minute)

	nodeGroup, err = WaitDedicatedNodeGroupReady(ctx, clusterCreateTimeout, clusterCreateInterval, data.ClusterId.ValueString(), nodeGroupId, r.provider.DedicatedClient)
	if err != nil {
		resp.Diagnostics.AddError(
			"Node group creation failed",
			fmt.Sprintf("Node group is not ready, get error: %s", err),
		)
		return
	}

	refreshDedicatedNodeGroupResourceData(nodeGroup, &data)

	// using tidb node group api create public endpoint setting
	tflog.Debug(ctx, fmt.Sprintf("\n\n\n\n\n\n\ncreate dedicated_node_group_resource cluster_id: %s", data.PublicEndpointSetting))
	pes, err := updatePublicEndpointSetting(ctx, r.provider.DedicatedClient, data.ClusterId.ValueString(), data.NodeGroupId.ValueString(), data.PublicEndpointSetting)
	if err != nil {
		resp.Diagnostics.AddError("Update Error", fmt.Sprintf("Unable to call UpdatePublicEndpoint, got error: %s", err))
		return
	}
	data.PublicEndpointSetting = pes

	// save into the Terraform state.
	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (r dedicatedNodeGroupResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// get data from state
	var data dedicatedNodeGroupResourceData
	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Debug(ctx, fmt.Sprintf("read dedicated_node_group_resource cluster_id: %s", data.NodeGroupId.ValueString()))

	// call read api
	tflog.Trace(ctx, "read dedicated_node_group_resource")
	nodeGroup, err := r.provider.DedicatedClient.GetTiDBNodeGroup(ctx, data.ClusterId.ValueString(), data.NodeGroupId.ValueString())
	if err != nil {
		tflog.Error(ctx, fmt.Sprintf("Unable to call GetTiDBNodeGroup, error: %s", err))
		return
	}
	refreshDedicatedNodeGroupResourceData(nodeGroup, &data)

	// using tidb node group api get public endpoint setting
	publicEndpointSetting, err := r.provider.DedicatedClient.GetPublicEndpoint(ctx, data.ClusterId.ValueString(), data.NodeGroupId.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Read Error", fmt.Sprintf("Unable to call GetPublicEndpoint, got error: %s", err))
		return
	}
	data.PublicEndpointSetting = convertDedicatedPublicEndpointSetting(publicEndpointSetting)

	// save into the Terraform state.
	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (r dedicatedNodeGroupResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var clusterId string
	var nodeGroupId string

	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("cluster_id"), &clusterId)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("node_group_id"), &nodeGroupId)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Trace(ctx, "delete dedicated_node_group_resource")
	err := r.provider.DedicatedClient.DeleteTiDBNodeGroup(ctx, clusterId, nodeGroupId)
	if err != nil {
		resp.Diagnostics.AddError("Delete Error", fmt.Sprintf("Unable to call DeleteTiDBNodeGroup, got error: %s", err))
		return
	}
}

func (r dedicatedNodeGroupResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// get plan
	var plan dedicatedNodeGroupResourceData
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	// get state
	var state dedicatedNodeGroupResourceData
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	newDisplayName := plan.DisplayName.ValueString()
	newNodeCount := int32(plan.NodeCount.ValueInt32())
	body := dedicated.TidbNodeGroupServiceUpdateTidbNodeGroupRequest{
		DisplayName: &newDisplayName,
		NodeCount:   *dedicated.NewNullableInt32(&newNodeCount),
	}

	if plan.TiProxySetting != nil {
		tiProxySetting := dedicated.Dedicatedv1beta1TidbNodeGroupTiProxySetting{}
		tiProxyNodeCount := plan.TiProxySetting.NodeCount.ValueInt32()
		tiProxyType := dedicated.TidbNodeGroupTiProxyType(plan.TiProxySetting.Type.ValueString())
		tiProxySetting.NodeCount = *dedicated.NewNullableInt32(&tiProxyNodeCount)
		tiProxySetting.Type = &tiProxyType
		body.TiproxySetting = &tiProxySetting
	}

	// call update api
	tflog.Trace(ctx, "update dedicated_node_group_resource")
	_, err := r.provider.DedicatedClient.UpdateTiDBNodeGroup(ctx, state.ClusterId.ValueString(), plan.NodeGroupId.ValueString(), &body)
	if err != nil {
		resp.Diagnostics.AddError("Update Error", fmt.Sprintf("Unable to call UpdateTiDBNodeGroup, got error: %s", err))
		return
	}

	tflog.Info(ctx, "wait node group ready")
	nodeGroup, err := WaitDedicatedNodeGroupReady(ctx, clusterUpdateTimeout, clusterUpdateInterval, state.ClusterId.ValueString(), state.NodeGroupId.ValueString(), r.provider.DedicatedClient)
	if err != nil {
		resp.Diagnostics.AddError(
			"Node group update failed",
			fmt.Sprintf("Node Group is not ready, get error: %s", err),
		)
		return
	}

	refreshDedicatedNodeGroupResourceData(nodeGroup, &state)

	// using tidb node group api update public endpoint setting
	pes, err := updatePublicEndpointSetting(ctx, r.provider.DedicatedClient, state.ClusterId.ValueString(), state.NodeGroupId.ValueString(), plan.PublicEndpointSetting)
	if err != nil {
		resp.Diagnostics.AddError("Update Error", fmt.Sprintf("Unable to call UpdatePublicEndpoint, got error: %s", err))
		return
	}
	state.PublicEndpointSetting = pes

	// save into the Terraform state.
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)

}

func (r dedicatedNodeGroupResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	idParts := strings.Split(req.ID, ",")

	if len(idParts) != 2 || idParts[0] == "" || idParts[1] == "" {
		resp.Diagnostics.AddError(
			"Unexpected Import Identifier",
			fmt.Sprintf("Expected import identifier with format: cluster_id, node_group_id. Got: %q", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("cluster_id"), idParts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("node_group_id"), idParts[1])...)
}

func buildCreateDedicatedNodeGroupBody(data dedicatedNodeGroupResourceData) dedicated.Required {
	displayName := data.DisplayName.ValueString()
	nodeCount := int32(data.NodeCount.ValueInt32())
	nodeGroup := dedicated.Required{
		DisplayName: &displayName,
		NodeCount:   nodeCount,
	}

	if data.TiProxySetting != nil {
		tiProxySetting := dedicated.Dedicatedv1beta1TidbNodeGroupTiProxySetting{}
		tiProxyNodeCount := data.TiProxySetting.NodeCount.ValueInt32()
		tiProxyType := dedicated.TidbNodeGroupTiProxyType(data.TiProxySetting.Type.ValueString())
		tiProxySetting.NodeCount = *dedicated.NewNullableInt32(&tiProxyNodeCount)
		tiProxySetting.Type = &tiProxyType
		nodeGroup.TiproxySetting = &tiProxySetting
	}

	return nodeGroup
}

func refreshDedicatedNodeGroupResourceData(resp *dedicated.Dedicatedv1beta1TidbNodeGroup, data *dedicatedNodeGroupResourceData) {
	data.DisplayName = types.StringValue(*resp.DisplayName)
	data.NodeSpecDisplayName = types.StringValue(*resp.NodeSpecDisplayName)
	data.IsDefaultGroup = types.BoolValue(*resp.IsDefaultGroup)
	data.State = types.StringValue(string(*resp.State))
	data.NodeCount = types.Int32Value(resp.NodeCount)
	data.NodeSpecKey = types.StringValue(*resp.NodeSpecKey)
	var endpoints []endpoint
	for _, e := range resp.Endpoints {
		endpoints = append(endpoints, endpoint{
			Host:           types.StringValue(*e.Host),
			Port:           types.Int32Value(*e.Port),
			ConnectionType: types.StringValue(string(*e.ConnectionType)),
		})
	}
	data.Endpoints = endpoints
}

func WaitDedicatedNodeGroupReady(ctx context.Context, timeout time.Duration, interval time.Duration, clusterId string, nodeGroupId string,
	client tidbcloud.TiDBCloudDedicatedClient) (*dedicated.Dedicatedv1beta1TidbNodeGroup, error) {
	stateConf := &retry.StateChangeConf{
		Pending: []string{
			string(dedicated.DEDICATEDV1BETA1TIDBNODEGROUPSTATE_MODIFYING),
		},
		Target: []string{
			string(dedicated.DEDICATEDV1BETA1TIDBNODEGROUPSTATE_ACTIVE),
			string(dedicated.DEDICATEDV1BETA1TIDBNODEGROUPSTATE_PAUSED),
		},
		Timeout:      timeout,
		MinTimeout:   500 * time.Millisecond,
		PollInterval: interval,
		Refresh:      dedicatedNodeGroupStateRefreshFunc(ctx, clusterId, nodeGroupId, client),
	}

	outputRaw, err := stateConf.WaitForStateContext(ctx)

	if output, ok := outputRaw.(*dedicated.Dedicatedv1beta1TidbNodeGroup); ok {
		return output, err
	}
	return nil, err
}

func dedicatedNodeGroupStateRefreshFunc(ctx context.Context, clusterId string, nodeGroupId string,
	client tidbcloud.TiDBCloudDedicatedClient) retry.StateRefreshFunc {
	return func() (interface{}, string, error) {
		tflog.Trace(ctx, "Waiting for dedicated node group ready")
		nodeGroup, err := client.GetTiDBNodeGroup(ctx, clusterId, nodeGroupId)
		if err != nil {
			return nil, "", err
		}
		return nodeGroup, string(*nodeGroup.State), nil
	}
}
