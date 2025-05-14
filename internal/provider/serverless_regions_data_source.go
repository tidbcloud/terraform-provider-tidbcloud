package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type ServerlessRegionsDataSourceData struct {
	Regions []serverlessRegion `tfsdk:"regions"`
}

type serverlessRegion struct {
	Name          types.String `tfsdk:"name"`
	RegionId      types.String `tfsdk:"region_id"`
	CloudProvider types.String `tfsdk:"cloud_provider"`
	DisplayName   types.String `tfsdk:"display_name"`
}

var _ datasource.DataSource = &ServerlessRegionsDataSource{}

type ServerlessRegionsDataSource struct {
	provider *tidbcloudProvider
}

func NewServerlessRegionsDataSource() datasource.DataSource {
	return &ServerlessRegionsDataSource{}
}

func (d *ServerlessRegionsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_serverless_regions"
}

func (d *ServerlessRegionsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	var ok bool
	if d.provider, ok = req.ProviderData.(*tidbcloudProvider); !ok {
		resp.Diagnostics.AddError("Internal provider error",
			fmt.Sprintf("Error in Configure: expected %T but got %T", tidbcloudProvider{}, req.ProviderData))
	}
}

func (d *ServerlessRegionsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Serverless regions data source",
		Attributes: map[string]schema.Attribute{
			"regions": schema.ListNestedAttribute{
				MarkdownDescription: "The regions.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							MarkdownDescription: "The unique name of the region.",
							Computed:            true,
						},
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

func (d *ServerlessRegionsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ServerlessRegionsDataSourceData
	diags := req.Config.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Trace(ctx, "read serverless regions data source")
	regions, err := d.provider.ServerlessClient.ListProviderRegions(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Read Error", fmt.Sprintf("Unable to call ListRegions, got error: %s", err))
		return
	}
	var items []serverlessRegion
	for _, r := range regions {
		items = append(items, serverlessRegion{
			Name:          types.StringValue(*r.Name),
			RegionId:      types.StringValue(*r.RegionId),
			CloudProvider: types.StringValue(string(*r.CloudProvider)),
			DisplayName:   types.StringValue(*r.DisplayName),
		})
	}
	data.Regions = items

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}
