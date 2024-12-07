package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type dedicatedCloudProvidersDataSourceData struct {
	CloudProviders []types.String `tfsdk:"cloud_providers"`
	ProjectId      types.String   `tfsdk:"project_id"`
}

var _ datasource.DataSource = &dedicatedCloudProvidersDataSource{}

type dedicatedCloudProvidersDataSource struct {
	provider *tidbcloudProvider
}

func NewDedicatedCloudProvidersDataSource() datasource.DataSource {
	return &dedicatedCloudProvidersDataSource{}
}

func (d *dedicatedCloudProvidersDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dedicated_cloud_providers"
}

func (d *dedicatedCloudProvidersDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	var ok bool
	if d.provider, ok = req.ProviderData.(*tidbcloudProvider); !ok {
		resp.Diagnostics.AddError("Internal provider error",
			fmt.Sprintf("Error in Configure: expected %T but got %T", tidbcloudProvider{}, req.ProviderData))
	}
}

func (d *dedicatedCloudProvidersDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "dedicated cloud providers data source",
		Attributes: map[string]schema.Attribute{
			"project_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the project.",
				Optional:            true,
			},
			"cloud_providers": schema.ListAttribute{
				MarkdownDescription: "The items of cloud providers",
				Computed:            true,
				ElementType:         types.StringType,
			},
		},
	}
}

func (d *dedicatedCloudProvidersDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data dedicatedCloudProvidersDataSourceData
	diags := req.Config.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Trace(ctx, "read regions data source")
	cloudProviders, err := d.provider.DedicatedClient.ListCloudProviders(ctx, data.ProjectId.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Read Error", fmt.Sprintf("Unable to call ListCloudProviders, got error: %s", err))
		return
	}
	var items []types.String
	for _, c := range cloudProviders {
		items = append(items, types.StringValue(string(c)))
	}
	data.CloudProviders = items

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}
