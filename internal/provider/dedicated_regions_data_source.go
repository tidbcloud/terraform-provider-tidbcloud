package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type dedicatedRegionsDataSourceData struct {
	CloudProvider types.String      `tfsdk:"cloud_provider"`
	ProjectId     types.String      `tfsdk:"project_id"`
	Regions       []dedicatedRegion `tfsdk:"regions"`
}

var _ datasource.DataSource = &dedicatedRegionsDataSource{}

type dedicatedRegionsDataSource struct {
	provider *tidbcloudProvider
}

func NewDedicatedRegionsDataSource() datasource.DataSource {
	return &dedicatedRegionsDataSource{}
}

func (d *dedicatedRegionsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dedicated_regions"
}

func (d *dedicatedRegionsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	var ok bool
	if d.provider, ok = req.ProviderData.(*tidbcloudProvider); !ok {
		resp.Diagnostics.AddError("Internal provider error",
			fmt.Sprintf("Error in Configure: expected %T but got %T", tidbcloudProvider{}, req.ProviderData))
	}
}

func (d *dedicatedRegionsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "dedicated regions data source",
		Attributes: map[string]schema.Attribute{
			"cloud_provider": schema.StringAttribute{
				MarkdownDescription: "The cloud provider of the regions. If set, it will return the regions that can be selected under this provider.",
				Optional:            true,
			},
			"project_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the project. If set, it will return the regions that can be selected under this project.",
				Optional:            true,
			},
			"regions": schema.ListNestedAttribute{
				MarkdownDescription: "The regions.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"region_id": schema.StringAttribute{
							MarkdownDescription: "The ID of the region.",
							Computed:            true,
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
				},
			},
		},
	}
}

func (d *dedicatedRegionsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data dedicatedRegionsDataSourceData
	diags := req.Config.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Trace(ctx, "read regions data source")
	regions, err := d.provider.DedicatedClient.ListRegions(ctx, data.CloudProvider.ValueString(), data.ProjectId.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Read Error", fmt.Sprintf("Unable to call ListRegions, got error: %s", err))
		return
	}
	var items []dedicatedRegion
	for _, r := range regions {
		items = append(items, dedicatedRegion{
			RegionId:      types.StringValue(*r.RegionId),
			CloudProvider: types.StringValue(string(*r.CloudProvider)),
			DisplayName:   types.StringValue(*r.DisplayName),
		})
	}
	data.Regions = items

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}
