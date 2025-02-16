package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type dedicatedNetworkContainersDataSourceData struct {
	NetworkContainers []networkContainerItem `tfsdk:"network_containers"`
}

type networkContainerItem struct {
	NetworkContainerId types.String `tfsdk:"network_container_id"`
	RegionId           types.String `tfsdk:"region_id"`
	CidrNotion         types.String `tfsdk:"cidr_notion"`
	State              types.String `tfsdk:"state"`
	CloudProvider      types.String `tfsdk:"cloud_provider"`
	RegionDisplayName  types.String `tfsdk:"region_display_name"`
	VpcId              types.String `tfsdk:"vpc_id"`
	Labels             types.Map    `tfsdk:"labels"`
}

var _ datasource.DataSource = &dedicatedNetworkContainersDataSource{}

type dedicatedNetworkContainersDataSource struct {
	provider *tidbcloudProvider
}

func NewDedicatedNetworkContainersDataSource() datasource.DataSource {
	return &dedicatedNetworkContainersDataSource{}
}

func (d *dedicatedNetworkContainersDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dedicated_network_containers"
}

func (d *dedicatedNetworkContainersDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	var ok bool
	if d.provider, ok = req.ProviderData.(*tidbcloudProvider); !ok {
		resp.Diagnostics.AddError("Internal provider error",
			fmt.Sprintf("Error in Configure: expected %T but got %T", tidbcloudProvider{}, req.ProviderData))
	}
}

func (d *dedicatedNetworkContainersDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "dedicated network containers data source",
		Attributes: map[string]schema.Attribute{
			"network_containers": schema.ListNestedAttribute{
				MarkdownDescription: "The network containers.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"network_container_id": schema.StringAttribute{
							Description: "The ID of the network container",
							Computed:    true,
						},
						"region_id": schema.StringAttribute{
							Description: "The region ID for the network container",
							Computed:    true,
						},
						"cidr_notion": schema.StringAttribute{
							Description: "CIDR notation for the network container",
							Computed:    true,
						},
						"state": schema.StringAttribute{
							Description: "The state of the network container",
							Computed:    true,
						},
						"cloud_provider": schema.StringAttribute{
							Description: "The cloud provider for the network container",
							Computed:    true,
						},
						"region_display_name": schema.StringAttribute{
							Description: "The display name of the region",
							Computed:    true,
						},
						"vpc_id": schema.StringAttribute{
							Description: "The VPC ID for the network container",
							Computed:    true,
						},
						"labels": schema.MapAttribute{
							Description: "The labels for the network container",
							Computed:    true,
							ElementType: types.StringType,
						},
					},
				},
			},
		},
	}
}

func (d *dedicatedNetworkContainersDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data dedicatedNetworkContainersDataSourceData
	diags := req.Config.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Trace(ctx, "read network containers data source")
	networkContainers, err := d.provider.DedicatedClient.ListNetworkContainers(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Read Error", fmt.Sprintf("Unable to call ListNetworkContainers, got error: %s", err))
		return
	}

	var items []networkContainerItem
	for _, networkContainer := range networkContainers {
		labels, diag := types.MapValueFrom(ctx, types.StringType, *networkContainer.Labels)
		if diag.HasError() {
			return
		}
		items = append(items, networkContainerItem{
			NetworkContainerId: types.StringValue(*networkContainer.NetworkContainerId),
			RegionId:           types.StringValue(networkContainer.RegionId),
			CidrNotion:         types.StringValue(*networkContainer.CidrNotion),
			State:              types.StringValue(string(*networkContainer.State)),
			CloudProvider:      types.StringValue(string(*networkContainer.CloudProvider)),
			RegionDisplayName:  types.StringValue(*networkContainer.RegionDisplayName),
			VpcId:              types.StringValue(*networkContainer.VpcId),
			Labels:             labels,
		})
	}
	data.NetworkContainers = items

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}
