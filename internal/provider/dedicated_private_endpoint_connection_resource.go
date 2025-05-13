package provider

import (
	"context"
	"fmt"
	"strings"
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

const (
	dedicatedPrivateEndpointConnectionCreateTimeout  = time.Hour
	dedicatedPrivateEndpointConnectionCreateInterval = 5 * time.Second
)

type dedicatedPrivateEndpointConnectionResourceData struct {
	ClusterId                   types.String `tfsdk:"cluster_id"`
	ClusterDisplayName          types.String `tfsdk:"cluster_display_name"`
	NodeGroupId                 types.String `tfsdk:"node_group_id"`
	PrivateEndpointConnectionId types.String `tfsdk:"private_endpoint_connection_id"`
	Labels                      types.Map    `tfsdk:"labels"`
	EndpointId                  types.String `tfsdk:"endpoint_id"`
	PrivateIpAddress            types.String `tfsdk:"private_ip_address"`
	EndpointStatus              types.String `tfsdk:"endpoint_status"`
	Message                     types.String `tfsdk:"message"`
	RegionId                    types.String `tfsdk:"region_id"`
	RegionDisplayName           types.String `tfsdk:"region_display_name"`
	CloudProvider               types.String `tfsdk:"cloud_provider"`
	PrivateLinkServiceName      types.String `tfsdk:"private_link_service_name"`
	AccountId                   types.String `tfsdk:"account_id"`
}

type dedicatedPrivateEndpointConnectionResource struct {
	provider *tidbcloudProvider
}

func NewDedicatedPrivateEndpointConnectionResource() resource.Resource {
	return &dedicatedPrivateEndpointConnectionResource{}
}

func (r *dedicatedPrivateEndpointConnectionResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dedicated_private_endpoint_connection"
}

func (r *dedicatedPrivateEndpointConnectionResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *dedicatedPrivateEndpointConnectionResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "dedicated private endpoint connection resource",
		Attributes: map[string]schema.Attribute{
			"cluster_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the cluster.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"cluster_display_name": schema.StringAttribute{
				MarkdownDescription: "The display name of the cluster.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"node_group_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the node group.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"private_endpoint_connection_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the private endpoint connection.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"endpoint_id": schema.StringAttribute{
				MarkdownDescription: "The endpoint ID of the private link connection.\n" +
					"For AWS, it's VPC endpoint ID.\n" +
					"For GCP, it's private service connect endpoint ID.\n" +
					"For Azure, it's private endpoint resource ID.",
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"private_ip_address": schema.StringAttribute{
				MarkdownDescription: "The private IP address of the private endpoint in the user's vNet.\n" +
					"TiDB Cloud will setup a public DNS record for this private IP address. So the user can use DNS address to connect to the cluster.\n" +
					"Only available for Azure clusters.",
				Optional: true,
			},
			"endpoint_status": schema.StringAttribute{
				MarkdownDescription: "The status of the endpoint.",
				Computed:            true,
			},
			"message": schema.StringAttribute{
				MarkdownDescription: "The message of the endpoint.",
				Computed:            true,
			},
			"region_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the region.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"region_display_name": schema.StringAttribute{
				MarkdownDescription: "The display name of the region.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"cloud_provider": schema.StringAttribute{
				MarkdownDescription: "The cloud provider of the region.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"private_link_service_name": schema.StringAttribute{
				MarkdownDescription: "The name of the private link service.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"account_id": schema.StringAttribute{
				MarkdownDescription: "Only for GCP private service connections. It's GCP project name.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"labels": schema.MapAttribute{
				MarkdownDescription: "The labels of the endpoint.",
				Computed:            true,
				ElementType:         types.StringType,
			},
		},
	}
}

func (r dedicatedPrivateEndpointConnectionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if !r.provider.configured {
		resp.Diagnostics.AddError(
			"Provider not configured",
			"The provider hasn't been configured before apply, likely because it depends on an unknown value from another resource. This leads to weird stuff happening, so we'd prefer if you didn't do that. Thanks!",
		)
		return
	}

	// get data from config
	var data dedicatedPrivateEndpointConnectionResourceData
	diags := req.Config.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Trace(ctx, "create dedicated_private_endpoint_connection_resource")
	body := buildCreateDedicatedPrivateEndpointConnectionBody(data)
	privateEndpointConnection, err := r.provider.DedicatedClient.CreatePrivateEndpointConnection(ctx, data.ClusterId.ValueString(), data.NodeGroupId.ValueString(), &body)
	if err != nil {
		resp.Diagnostics.AddError("Create Error", fmt.Sprintf("Unable to call CreatePrivateEndpointConnection, got error: %s", err))
		return
	}
	privateEndpointConnectionId := *privateEndpointConnection.PrivateEndpointConnectionId
	data.PrivateEndpointConnectionId = types.StringValue(privateEndpointConnectionId)
	tflog.Info(ctx, "wait dedicated private endpoint connection ready")
	privateEndpointConnection, err = WaitDedicatedPrivateEndpointConnectionReady(ctx, dedicatedPrivateEndpointConnectionCreateTimeout, dedicatedPrivateEndpointConnectionCreateInterval, data.ClusterId.ValueString(), data.NodeGroupId.ValueString(), privateEndpointConnectionId, r.provider.DedicatedClient)
	if err != nil {
		resp.Diagnostics.AddError(
			"Dedicated private endpoint connection creation failed",
			fmt.Sprintf("Dedicated private endpoint connection is not ready, get error: %s", err),
		)
		return
	}
	refreshDedicatedPrivateEndpointConnectionResourceData(privateEndpointConnection, &data)

	// save into the Terraform state.
	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (r dedicatedPrivateEndpointConnectionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// get data from state
	var data dedicatedPrivateEndpointConnectionResourceData
	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Debug(ctx, fmt.Sprintf("read dedicated_private_endpoint_connection_resource private_endpoint_connection_id: %s", data.PrivateEndpointConnectionId.ValueString()))

	// call read api
	tflog.Trace(ctx, "read dedicated_private_endpoint_connection_resource")
	privateEndpointConnection, err := r.provider.DedicatedClient.GetPrivateEndpointConnection(ctx, data.ClusterId.ValueString(), data.NodeGroupId.ValueString(), data.PrivateEndpointConnectionId.ValueString())
	if err != nil {
		tflog.Error(ctx, fmt.Sprintf("Unable to call GetPrivateEndpointConnection, error: %s", err))
		return
	}
	refreshDedicatedPrivateEndpointConnectionResourceData(privateEndpointConnection, &data)

	// save into the Terraform state.
	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (r dedicatedPrivateEndpointConnectionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var clusterId string
	var nodeGroupId string
	var privateEndpointConnectionId string

	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("cluster_id"), &clusterId)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("node_group_id"), &nodeGroupId)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("private_endpoint_connection_id"), &privateEndpointConnectionId)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Trace(ctx, "delete dedicated_private_endpoint_connection_resource")
	err := r.provider.DedicatedClient.DeletePrivateEndpointConnection(ctx, clusterId, nodeGroupId, privateEndpointConnectionId)
	if err != nil {
		resp.Diagnostics.AddError("Delete Error", fmt.Sprintf("Unable to call DeletePrivateEndpointConnection, got error: %s", err))
		return
	}
}

// NOTICE: update is not supported for dedicated private endpoint connection
func (r dedicatedPrivateEndpointConnectionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Update Error", "Update is not supported for dedicated private endpoint connection")
	return
}

func (r dedicatedPrivateEndpointConnectionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	idParts := strings.Split(req.ID, ",")

	if len(idParts) != 3 || idParts[0] == "" || idParts[1] == "" || idParts[2] == "" {
		resp.Diagnostics.AddError(
			"Unexpected Import Identifier",
			fmt.Sprintf("Expected import identifier with format: cluster_id, node_group_id, private_endpoint_connection_id. Got: %q", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("cluster_id"), idParts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("node_group_id"), idParts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("private_endpoint_connection_id"), idParts[2])...)
}

func buildCreateDedicatedPrivateEndpointConnectionBody(data dedicatedPrivateEndpointConnectionResourceData) dedicated.PrivateEndpointConnectionServiceCreatePrivateEndpointConnectionRequest {
	endpointId := data.EndpointId.ValueString()
	privateIpAddress := data.PrivateIpAddress.ValueString()

	return dedicated.PrivateEndpointConnectionServiceCreatePrivateEndpointConnectionRequest{
		EndpointId:       endpointId,
		PrivateIpAddress: *dedicated.NewNullableString(&privateIpAddress),
	}
}

func refreshDedicatedPrivateEndpointConnectionResourceData(resp *dedicated.V1beta1PrivateEndpointConnection, data *dedicatedPrivateEndpointConnectionResourceData) {
	data.EndpointId = types.StringValue(resp.EndpointId)
	if resp.PrivateIpAddress.IsSet() {
		data.PrivateIpAddress = types.StringValue(*resp.PrivateIpAddress.Get())
	}
	if resp.AccountId.IsSet() {
		data.AccountId = types.StringValue(*resp.AccountId.Get())
	}
	data.EndpointStatus = types.StringValue(string(*resp.EndpointState))
	data.Message = types.StringValue(*resp.Message)
	data.RegionId = types.StringValue(*resp.RegionId)
	data.RegionDisplayName = types.StringValue(*resp.RegionDisplayName)
	data.CloudProvider = types.StringValue(string(*resp.CloudProvider))
	data.PrivateLinkServiceName = types.StringValue(*resp.PrivateLinkServiceName)

}

func WaitDedicatedPrivateEndpointConnectionReady(ctx context.Context, timeout time.Duration, interval time.Duration, clusterId string, nodeGroupId string, privateEndpointConnectionId string,
	client tidbcloud.TiDBCloudDedicatedClient) (*dedicated.V1beta1PrivateEndpointConnection, error) {
	stateConf := &retry.StateChangeConf{
		Pending: []string{
			string(dedicated.PRIVATEENDPOINTCONNECTIONENDPOINTSTATE_PENDING),
		},
		Target: []string{
			string(dedicated.PRIVATEENDPOINTCONNECTIONENDPOINTSTATE_ACTIVE),
			string(dedicated.PRIVATEENDPOINTCONNECTIONENDPOINTSTATE_DELETING),
			string(dedicated.PRIVATEENDPOINTCONNECTIONENDPOINTSTATE_FAILED),
			string(dedicated.PRIVATEENDPOINTCONNECTIONENDPOINTSTATE_DISCOVERED),
		},
		Timeout:      timeout,
		MinTimeout:   500 * time.Millisecond,
		PollInterval: interval,
		Refresh:      dedicatedPrivateEndpointConnectionStateRefreshFunc(ctx, clusterId, nodeGroupId, privateEndpointConnectionId, client),
	}

	outputRaw, err := stateConf.WaitForStateContext(ctx)

	if output, ok := outputRaw.(*dedicated.V1beta1PrivateEndpointConnection); ok {
		return output, err
	}
	return nil, err
}

func dedicatedPrivateEndpointConnectionStateRefreshFunc(ctx context.Context, clusterId string, nodeGroupId string, privateEndpointConnectionId string,
	client tidbcloud.TiDBCloudDedicatedClient) retry.StateRefreshFunc {
	return func() (interface{}, string, error) {
		tflog.Trace(ctx, "Waiting for dedicated private endpoint connection ready")
		privateEndpointConnection, err := client.GetPrivateEndpointConnection(ctx, clusterId, nodeGroupId, privateEndpointConnectionId)
		if err != nil {
			return nil, "", err
		}
		tflog.Debug(ctx, fmt.Sprintf("dedicated private endpoint connection state: %s", *privateEndpointConnection.EndpointState))
		return privateEndpointConnection, string(*privateEndpointConnection.EndpointState), nil
	}
}
