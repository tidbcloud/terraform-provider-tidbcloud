package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type dedicatedPrivateEndpointConnectionDataSourceData struct {
	ClusterId                   types.String `tfsdk:"cluster_id"`
	ClusterDisplayName          types.String `tfsdk:"cluster_display_name"`
	NodeGroupId                 types.String `tfsdk:"node_group_id"`
	PrivateEndpointConnectionId types.String `tfsdk:"private_endpoint_connection_id"`
	Message                     types.String `tfsdk:"message"`
	RegionId                    types.String `tfsdk:"region_id"`
	RegionDisplayName           types.String `tfsdk:"region_display_name"`
	CloudProvider               types.String `tfsdk:"cloud_provider"`
	PrivateLinkServiceName      types.String `tfsdk:"private_link_service_name"`
	PrivateLinkServiceState     types.String `tfsdk:"private_link_service_state"`
	Labels                      types.Map    `tfsdk:"labels"`
	EndpointId                  types.String `tfsdk:"endpoint_id"`
	PrivateIpAddress            types.String `tfsdk:"private_ip_address"`
	EndpointState               types.String `tfsdk:"endpoint_state"`
	NodeGroupDisplayName        types.String `tfsdk:"node_group_display_name"`
	AccountId                   types.String `tfsdk:"account_id"`
	Host                        types.String `tfsdk:"host"`
	Port                        types.Int64  `tfsdk:"port"`
}

var _ datasource.DataSource = &dedicatedPrivateEndpointConnectionDataSource{}

type dedicatedPrivateEndpointConnectionDataSource struct {
	provider *tidbcloudProvider
}

func NewDedicatedPrivateEndpointConnectionDataSource() datasource.DataSource {
	return &dedicatedPrivateEndpointConnectionDataSource{}
}

func (d *dedicatedPrivateEndpointConnectionDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dedicated_private_endpoint_connection"
}

func (d *dedicatedPrivateEndpointConnectionDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	var ok bool
	if d.provider, ok = req.ProviderData.(*tidbcloudProvider); !ok {
		resp.Diagnostics.AddError("Internal provider error",
			fmt.Sprintf("Error in Configure: expected %T but got %T", tidbcloudProvider{}, req.ProviderData))
	}
}

func (d *dedicatedPrivateEndpointConnectionDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "dedicated private endpoint connection data source",
		Attributes: map[string]schema.Attribute{
			"cluster_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the cluster.",
				Required:            true,
			},
			"cluster_display_name": schema.StringAttribute{
				MarkdownDescription: "The display name of the cluster.",
				Computed:            true,
			},
			"node_group_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the node group.",
				Required:            true,
			},
			"private_endpoint_connection_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the private endpoint connection.",
				Required:            true,
			},
			"message": schema.StringAttribute{
				MarkdownDescription: "The message of the private endpoint connection.",
				Computed:            true,
			},
			"region_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the region.",
				Computed:            true,
			},
			"region_display_name": schema.StringAttribute{
				MarkdownDescription: "The display name of the region.",
				Computed:            true,
			},
			"cloud_provider": schema.StringAttribute{
				MarkdownDescription: "The cloud provider.",
				Computed:            true,
			},
			"private_link_service_name": schema.StringAttribute{
				MarkdownDescription: "The name of the private link service.",
				Computed:            true,
			},
			"private_link_service_state": schema.StringAttribute{
				MarkdownDescription: "The state of the private link service.",
				Computed:            true,
			},
			"labels": schema.MapAttribute{
				MarkdownDescription: "The labels of the private endpoint connection.",
				Computed:            true,
				ElementType:         types.StringType,
			},
			"endpoint_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the endpoint.",
				Computed:            true,
			},
			"private_ip_address": schema.StringAttribute{
				MarkdownDescription: "The private IP address of the endpoint.",
				Computed:            true,
			},
			"endpoint_state": schema.StringAttribute{
				MarkdownDescription: "The state of the endpoint.",
				Computed:            true,
			},
			"node_group_display_name": schema.StringAttribute{
				MarkdownDescription: "The display name of the node group.",
				Computed:            true,
			},
			"account_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the account.",
				Computed:            true,
			},
			"host": schema.StringAttribute{
				MarkdownDescription: "The host of the private endpoint connection.",
				Computed:            true,
			},
			"port": schema.Int64Attribute{
				MarkdownDescription: "The port of the private endpoint connection.",
				Computed:            true,
			},
		},
	}
}

func (d *dedicatedPrivateEndpointConnectionDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data dedicatedPrivateEndpointConnectionDataSourceData
	diags := req.Config.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Trace(ctx, "read private endpoint connection data source")
	privateEndpointConnection, err := d.provider.DedicatedClient.GetPrivateEndpointConnection(ctx, data.ClusterId.ValueString(), data.NodeGroupId.ValueString(), data.PrivateEndpointConnectionId.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Read Error", fmt.Sprintf("Unable to call GetPrivateEndpointConnection, got error: %s", err))
		return
	}

	labels, diag := types.MapValueFrom(ctx, types.StringType, *privateEndpointConnection.Labels)
	if diag.HasError() {
		return
	}
	if privateEndpointConnection.PrivateIpAddress.IsSet() {
		data.PrivateIpAddress = types.StringValue(*privateEndpointConnection.PrivateIpAddress.Get())
	}
	if privateEndpointConnection.AccountId.IsSet() {
		data.AccountId = types.StringValue(*privateEndpointConnection.AccountId.Get())
	}
	data.ClusterDisplayName = types.StringValue(*privateEndpointConnection.ClusterDisplayName)
	data.RegionId = types.StringValue(*privateEndpointConnection.RegionId)
	data.RegionDisplayName = types.StringValue(*privateEndpointConnection.RegionDisplayName)
	data.CloudProvider = types.StringValue(string(*privateEndpointConnection.CloudProvider))
	data.PrivateLinkServiceName = types.StringValue(*privateEndpointConnection.PrivateLinkServiceName)
	data.PrivateLinkServiceState = types.StringValue(string(*privateEndpointConnection.PrivateLinkServiceState))
	data.Message = types.StringValue(*privateEndpointConnection.Message)
	data.Labels = labels
	data.EndpointId = types.StringValue(privateEndpointConnection.EndpointId)
	data.EndpointState = types.StringValue(string(*privateEndpointConnection.EndpointState))
	data.NodeGroupDisplayName = types.StringValue(*privateEndpointConnection.TidbNodeGroupDisplayName)
	data.Host = types.StringValue(*privateEndpointConnection.Host)
	data.Port = types.Int64Value(int64(*privateEndpointConnection.Port))

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}
