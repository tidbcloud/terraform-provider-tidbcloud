package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type dedicatedRegion struct {
	RegionId      types.String `tfsdk:"region_id"`
	CloudProvider types.String `tfsdk:"cloud_provider"`
	DisplayName   types.String `tfsdk:"display_name"`
}

var _ datasource.DataSource = &dedicatedRegionDataSource{}

type dedicatedRegionDataSource struct {
	provider *tidbcloudProvider
}

func NewDedicatedRegionDataSource() datasource.DataSource {
	return &dedicatedRegionDataSource{}
}

func (d *dedicatedRegionDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dedicated_region"
}

func (d *dedicatedRegionDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	var ok bool
	if d.provider, ok = req.ProviderData.(*tidbcloudProvider); !ok {
		resp.Diagnostics.AddError("Internal provider error",
			fmt.Sprintf("Error in Configure: expected %T but got %T", tidbcloudProvider{}, req.ProviderData))
	}
}

func (d *dedicatedRegionDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "dedicated region data source",
		Attributes: map[string]schema.Attribute{
			"region_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the region.",
				Required:            true,
			},
			"cloud_provider": schema.StringAttribute{
				MarkdownDescription: "The cloud provider of the region.",
				Computed:            true,
			},
			"display_name": schema.StringAttribute{
				MarkdownDescription: "The display name of the region.",
				Computed:            true,
			},
		},
	}
}

func (d *dedicatedRegionDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data dedicatedRegion
	diags := req.Config.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Trace(ctx, "read region data source")
	region, err := d.provider.DedicatedClient.GetRegion(ctx, data.RegionId.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Read Error", fmt.Sprintf("Unable to call GetRegion, got error: %s", err))
		return
	}

	data.CloudProvider = types.StringValue(string(*region.CloudProvider))
	data.DisplayName = types.StringValue(string(*region.DisplayName))

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}
