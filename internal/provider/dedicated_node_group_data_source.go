package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type dedicatedNodeGroupDataSourceData struct {
	ClusterId           types.String `tfsdk:"cluster_id"`
	NodeSpecKey         types.String `tfsdk:"node_spec_key"`
	NodeCount           types.Int64  `tfsdk:"node_count"`
	NodeGroupId         types.String `tfsdk:"node_group_id"`
	DisplayName         types.String `tfsdk:"display_name"`
	NodeSpecDisplayName types.String `tfsdk:"node_spec_display_name"`
	IsDefaultGroup      types.Bool   `tfsdk:"is_default_group"`
	State               types.String `tfsdk:"state"`
	Endpoints           []endpoint   `tfsdk:"endpoints"`
}

var _ datasource.DataSource = &dedicatedNodeGroupDataSource{}

type dedicatedNodeGroupDataSource struct {
	provider *tidbcloudProvider
}

func NewDedicatedNodeGroupDataSource() datasource.DataSource {
	return &dedicatedNodeGroupDataSource{}
}

func (d *dedicatedNodeGroupDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dedicated_node_group"
}

func (d *dedicatedNodeGroupDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	var ok bool
	if d.provider, ok = req.ProviderData.(*tidbcloudProvider); !ok {
		resp.Diagnostics.AddError("Internal provider error",
			fmt.Sprintf("Error in Configure: expected %T but got %T", tidbcloudProvider{}, req.ProviderData))
	}
}

func (d *dedicatedNodeGroupDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "dedicated node group data source",
		Attributes: map[string]schema.Attribute{
			"cluster_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the cluster.",
				Required:            true,
			},
			"node_group_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the node group.",
				Required:            true,
			},
			"node_count": schema.Int64Attribute{
				MarkdownDescription: "The number of nodes in the node group.",
				Computed:            true,
			},
			"display_name": schema.StringAttribute{
				MarkdownDescription: "The display name of the node group.",
				Computed:            true,
			},
			"node_spec_key": schema.StringAttribute{
				MarkdownDescription: "The key of the node spec.",
				Computed:            true,
			},
			"node_spec_display_name": schema.StringAttribute{
				MarkdownDescription: "The display name of the node spec.",
				Computed:            true,
			},
			"is_default_group": schema.BoolAttribute{
				MarkdownDescription: "Indicates if this is the default group.",
				Computed:            true,
			},
			"state": schema.StringAttribute{
				MarkdownDescription: "The state of the node group.",
				Computed:            true,
			},
			"endpoints": schema.ListNestedAttribute{
				MarkdownDescription: "The endpoints of the node group.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"host": schema.StringAttribute{
							MarkdownDescription: "The host of the endpoint.",
							Computed:            true,
						},
						"port": schema.Int32Attribute{
							MarkdownDescription: "The port of the endpoint.",
							Computed:            true,
						},
						"connection_type": schema.StringAttribute{
							MarkdownDescription: "The connection type of the endpoint.",
							Computed:            true,
						},
					},
				},
			},
		},
	}
}

func (d *dedicatedNodeGroupDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data dedicatedNodeGroupDataSourceData
	diags := req.Config.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Trace(ctx, "read node group data source")
	nodeGroup, err := d.provider.DedicatedClient.GetTiDBNodeGroup(ctx, data.ClusterId.ValueString(), data.NodeGroupId.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Read Error", fmt.Sprintf("Unable to call GetRTiDBNodeGroup, got error: %s", err))
		return
	}

	data.NodeSpecKey = types.StringValue(string(*nodeGroup.NodeSpecKey))
	data.NodeCount = types.Int64Value(int64(nodeGroup.NodeCount))
	data.DisplayName = types.StringValue(string(*nodeGroup.DisplayName))
	data.NodeSpecDisplayName = types.StringValue(string(*nodeGroup.NodeSpecDisplayName))
	data.IsDefaultGroup = types.BoolValue(bool(*nodeGroup.IsDefaultGroup))
	data.State = types.StringValue(string(*nodeGroup.State))
	var endpoints []endpoint
	for _, e := range nodeGroup.Endpoints {
		endpoints = append(endpoints, endpoint{
			Host:           types.StringValue(*e.Host),
			Port:           types.Int32Value(*e.Port),
			ConnectionType: types.StringValue(string(*e.ConnectionType)),
		})
	}
	data.Endpoints = endpoints

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}
