package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type dedicatedPrivateLinkServiceDataSourceData struct {
	ClusterId         types.String `tfsdk:"cluster_id"`
	NodeGroupId       types.String `tfsdk:"node_group_id"`
	ServiceName       types.String `tfsdk:"service_name"`
	ServiceDNSName    types.String `tfsdk:"service_dns_name"`
	AvailableZones    types.List   `tfsdk:"available_zones"`
	State             types.String `tfsdk:"state"`
	RegionId          types.String `tfsdk:"region_id"`
	RegionDisplayName types.String `tfsdk:"region_display_name"`
	CloudProvider     types.String `tfsdk:"cloud_provider"`
}

var _ datasource.DataSource = &dedicatedPrivateLinkServiceDataSource{}

type dedicatedPrivateLinkServiceDataSource struct {
	provider *tidbcloudProvider
}

func NewDedicatedPrivateLinkServiceDataSource() datasource.DataSource {
	return &dedicatedPrivateLinkServiceDataSource{}
}

func (d *dedicatedPrivateLinkServiceDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dedicated_private_link_service"
}

func (d *dedicatedPrivateLinkServiceDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	var ok bool
	if d.provider, ok = req.ProviderData.(*tidbcloudProvider); !ok {
		resp.Diagnostics.AddError("Internal provider error",
			fmt.Sprintf("Error in Configure: expected %T but got %T", tidbcloudProvider{}, req.ProviderData))
	}
}

func (d *dedicatedPrivateLinkServiceDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "dedicated private link service data source",
		Attributes: map[string]schema.Attribute{
			"cluster_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the cluster.",
				Required:            true,
			},
			"node_group_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the TiDB node group.",
				Required:            true,
			},
			"service_name": schema.StringAttribute{
				MarkdownDescription: "The service name of the private link service.",
				Computed:            true,
			},
			"service_dns_name": schema.StringAttribute{
				MarkdownDescription: "The DNS name of the private link service.",
				Computed:            true,
			},
			"available_zones": schema.ListAttribute{
				MarkdownDescription: "The availability zones where the private link service is available.",
				Computed:            true,
				ElementType:         types.StringType,
			},
			"state": schema.StringAttribute{
				MarkdownDescription: "The state of the private link service.",
				Computed:            true,
			},
			"region_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the region where the private link service is located.",
				Computed:            true,
			},
			"region_display_name": schema.StringAttribute{
				MarkdownDescription: "The display name of the region where the private link service is located.",
				Computed:            true,
			},
			"cloud_provider": schema.StringAttribute{
				MarkdownDescription: "The cloud provider of the private link service.",
				Computed:            true,
			},
		},
	}
}

func (d *dedicatedPrivateLinkServiceDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data dedicatedPrivateLinkServiceDataSourceData
	diags := req.Config.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if d.provider == nil || d.provider.DedicatedClient == nil {
		resp.Diagnostics.AddError("Internal provider error", "The provider has not been configured before reading the dedicated private link service data source.")
		return
	}

	tflog.Trace(ctx, "read dedicated private link service data source")

	privateLinkService, err := d.provider.DedicatedClient.GetPrivateLinkService(ctx, data.ClusterId.ValueString(), data.NodeGroupId.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Read Error", fmt.Sprintf("Unable to call GetPrivateLinkService, got error: %s", err))
		return
	}
	if privateLinkService == nil {
		resp.Diagnostics.AddError("Read Error", "GetPrivateLinkService returned nil response")
		return
	}

	data.ServiceName = types.StringValue(*privateLinkService.ServiceName)
	data.ServiceDNSName = types.StringValue(*privateLinkService.ServiceDnsName)
	data.State = types.StringValue(string(*privateLinkService.State))
	data.RegionId = types.StringValue(*privateLinkService.RegionId)
	data.RegionDisplayName = types.StringValue(*privateLinkService.RegionDisplayName)
	data.CloudProvider = types.StringValue(string(*privateLinkService.CloudProvider))

	data.AvailableZones = types.ListNull(types.StringType)
	if len(privateLinkService.AvailableZones) > 0 {
		availableZones, diag := types.ListValueFrom(ctx, types.StringType, privateLinkService.AvailableZones)
		resp.Diagnostics.Append(diag...)
		if resp.Diagnostics.HasError() {
			return
		}
		data.AvailableZones = availableZones
	}

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}
