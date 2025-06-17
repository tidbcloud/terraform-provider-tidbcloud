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

type dedicatedPrivateEndpointConnectionsDataSourceData struct {
	ClusterId                  types.String                    `tfsdk:"cluster_id"`
	NodeGroupId                types.String                    `tfsdk:"node_group_id"`
	PrivateEndpointConnections []privateEndpointConnectionItem `tfsdk:"private_endpoint_connections"`
}

type privateEndpointConnectionItem struct {
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
	Port                        types.Int32  `tfsdk:"port"`
}

var _ datasource.DataSource = &dedicatedPrivateEndpointConnectionsDataSource{}

type dedicatedPrivateEndpointConnectionsDataSource struct {
	provider *tidbcloudProvider
}

func NewDedicatedPrivateEndpointConnectionsDataSource() datasource.DataSource {
	return &dedicatedPrivateEndpointConnectionsDataSource{}
}

func (d *dedicatedPrivateEndpointConnectionsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dedicated_private_endpoint_connections"
}

func (d *dedicatedPrivateEndpointConnectionsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	var ok bool
	if d.provider, ok = req.ProviderData.(*tidbcloudProvider); !ok {
		resp.Diagnostics.AddError("Internal provider error",
			fmt.Sprintf("Error in Configure: expected %T but got %T", tidbcloudProvider{}, req.ProviderData))
	}
}

func (d *dedicatedPrivateEndpointConnectionsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "dedicated private endpoint connections data source",
		Attributes: map[string]schema.Attribute{
			"cluster_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the cluster.",
				Required:            true,
			},
			"node_group_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the node group.",
				Required:            true,
			},
			"private_endpoint_connections": schema.ListNestedAttribute{
				MarkdownDescription: "The private endpoint connections.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"private_endpoint_connection_id": schema.StringAttribute{
							MarkdownDescription: "The ID of the private endpoint connection.",
							Computed:            true,
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
						"port": schema.Int32Attribute{
							MarkdownDescription: "The port of the private endpoint connection.",
							Computed:            true,
						},
					},
				},
			},
		},
	}
}

func (d *dedicatedPrivateEndpointConnectionsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data dedicatedPrivateEndpointConnectionsDataSourceData
	diags := req.Config.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Trace(ctx, "read private endpoint connections data source")
	privateEndpointConnections, err := d.retrievePrivateEndpointConnections(ctx, data.ClusterId.ValueString(), data.NodeGroupId.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Read Error", fmt.Sprintf("Unable to call ListPrivateEndpointConnection, got error: %s", err))
		return
	}

	var items []privateEndpointConnectionItem

	for _, privateEndpointConnection := range privateEndpointConnections {
		labels, diag := types.MapValueFrom(ctx, types.StringType, *privateEndpointConnection.Labels)
		if diag.HasError() {
			return
		}
		p := privateEndpointConnectionItem{
			PrivateEndpointConnectionId: types.StringValue(*privateEndpointConnection.PrivateEndpointConnectionId),
			Message:                     types.StringValue(*privateEndpointConnection.Message),
			RegionId:                    types.StringValue(*privateEndpointConnection.RegionId),
			RegionDisplayName:           types.StringValue(*privateEndpointConnection.RegionDisplayName),
			CloudProvider:               types.StringValue(string(*privateEndpointConnection.CloudProvider)),
			PrivateLinkServiceName:      types.StringValue(*privateEndpointConnection.PrivateLinkServiceName),
			PrivateLinkServiceState:     types.StringValue(string(*privateEndpointConnection.PrivateLinkServiceState)),
			Labels:                      labels,
			EndpointId:                  types.StringValue(privateEndpointConnection.EndpointId),
			EndpointState:               types.StringValue(string(*privateEndpointConnection.EndpointState)),
			NodeGroupDisplayName:        types.StringValue(*privateEndpointConnection.TidbNodeGroupDisplayName),
			Host:                        types.StringValue(*privateEndpointConnection.Host),
			Port:                        types.Int32Value(*privateEndpointConnection.Port),
		}
		if privateEndpointConnection.PrivateIpAddress.IsSet() {
			p.PrivateIpAddress = types.StringValue(*privateEndpointConnection.PrivateIpAddress.Get())
		}
		if privateEndpointConnection.AccountId.IsSet() {
			p.AccountId = types.StringValue(*privateEndpointConnection.AccountId.Get())
		}
		items = append(items, p)
	}
	data.PrivateEndpointConnections = items

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (d dedicatedPrivateEndpointConnectionsDataSource) retrievePrivateEndpointConnections(ctx context.Context, clusterId, nodeGroupId string) ([]dedicated.Dedicatedv1beta1PrivateEndpointConnection, error) {
	var items []dedicated.Dedicatedv1beta1PrivateEndpointConnection
	pageSizeInt32 := int32(DefaultPageSize)
	var pageToken *string
	for {
		privateEndpointConnections, err := d.provider.DedicatedClient.ListPrivateEndpointConnections(ctx, clusterId, nodeGroupId, &pageSizeInt32, pageToken)
		if err != nil {
			return nil, errors.Trace(err)
		}
		items = append(items, privateEndpointConnections.PrivateEndpointConnections...)

		pageToken = privateEndpointConnections.NextPageToken
		if IsNilOrEmpty(pageToken) {
			break
		}
	}
	return items, nil
}
