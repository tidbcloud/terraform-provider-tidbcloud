package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type dedicatedNetworkContainerDataSourceData struct {
	NetworkContainerId types.String `tfsdk:"network_container_id"`
	ProjectId          types.String `tfsdk:"project_id"`
	RegionId           types.String `tfsdk:"region_id"`
	RegionDisplayName  types.String `tfsdk:"region_display_name"`
	CloudProvider      types.String `tfsdk:"cloud_provider"`
	Labels             types.Map    `tfsdk:"labels"`
	State              types.String `tfsdk:"state"`
	CidrNotation         types.String `tfsdk:"cidr_notation"`
	VpcId              types.String `tfsdk:"vpc_id"`
}

var _ datasource.DataSource = &dedicatedNetworkContainerDataSource{}

type dedicatedNetworkContainerDataSource struct {
	provider *tidbcloudProvider
}

func NewDedicatedNetworkContainerDataSource() datasource.DataSource {
	return &dedicatedNetworkContainerDataSource{}
}

func (d *dedicatedNetworkContainerDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dedicated_network_container"
}

func (d *dedicatedNetworkContainerDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	var ok bool
	if d.provider, ok = req.ProviderData.(*tidbcloudProvider); !ok {
		resp.Diagnostics.AddError("Internal provider error",
			fmt.Sprintf("Error in Configure: expected %T but got %T", tidbcloudProvider{}, req.ProviderData))
	}
}

func (d *dedicatedNetworkContainerDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "dedicated network container data source",
		Attributes: map[string]schema.Attribute{
			"network_container_id": schema.StringAttribute{
				Description: "The ID of the network container",
				Required:    true,
			},
			"project_id": schema.StringAttribute{
				Description: "The project ID for the network container",
				Computed:    true,
			},
			"region_id": schema.StringAttribute{
				Description: "The region ID for the network container",
				Computed:    true,
			},
			"region_display_name": schema.StringAttribute{
				Description: "The region display name for the network container",
				Computed:    true,
			},
			"cloud_provider": schema.StringAttribute{
				Description: "The cloud provider for the network container",
				Computed:    true,
			},
			"labels": schema.MapAttribute{
				Description: "The labels for the network container",
				Computed:    true,
				ElementType: types.StringType,
			},
			"state": schema.StringAttribute{
				Description: "The state of the network container",
				Computed:    true,
			},
			"cidr_notation": schema.StringAttribute{
				Description: "CIDR notation for the network container",
				Computed:    true,
			},
			"vpc_id": schema.StringAttribute{
				Description: "The VPC ID for the network container",
				Computed:    true,
			},
		},
	}
}

func (d *dedicatedNetworkContainerDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data dedicatedNetworkContainerDataSourceData
	diags := req.Config.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Trace(ctx, "read network container data source")
	networkContainer, err := d.provider.DedicatedClient.GetNetworkContainer(ctx, data.NetworkContainerId.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Read Error", fmt.Sprintf("Unable to call GetNetworkContainer, got error: %s", err))
		return
	}

	labels, diag := types.MapValueFrom(ctx, types.StringType, *networkContainer.Labels)
	if diag.HasError() {
		return
	}
	data.RegionId = types.StringValue(networkContainer.RegionId)
	data.RegionDisplayName = types.StringValue(*networkContainer.RegionDisplayName)
	data.CloudProvider = types.StringValue(string(*networkContainer.CloudProvider))
	data.State = types.StringValue(string(*networkContainer.State))
	data.CidrNotation = types.StringValue(*networkContainer.CidrNotation)
	data.VpcId = types.StringValue(*networkContainer.VpcId)
	data.Labels = labels
	data.ProjectId = types.StringValue((*networkContainer.Labels)[LabelsKeyProjectId])

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}
