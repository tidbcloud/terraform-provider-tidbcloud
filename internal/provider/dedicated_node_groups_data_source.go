package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type dedicatedNodeGroupsDataSourceData struct {
	ClusterId   types.String     `tfsdk:"cluster_id"`
	NodeSpecKey types.String     `tfsdk:"node_spec_key"`
	NodeGroups  []nodeGroupItems `tfsdk:"node_groups"`
}

type nodeGroupItems struct {
	NodeCount           types.Int64  `tfsdk:"node_count"`
	NodeGroupId         types.String `tfsdk:"node_group_id"`
	DisplayName         types.String `tfsdk:"display_name"`
	NodeSpecDisplayName types.String `tfsdk:"node_spec_display_name"`
	IsDefaultGroup      types.Bool   `tfsdk:"is_default_group"`
	State               types.String `tfsdk:"state"`
}

var _ datasource.DataSource = &dedicatedNodeGroupsDataSource{}

type dedicatedNodeGroupsDataSource struct {
	provider *tidbcloudProvider
}

func NewDedicatedNodeGroupsDataSource() datasource.DataSource {
	return &dedicatedNodeGroupsDataSource{}
}

func (d *dedicatedNodeGroupsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dedicated_node_groups"
}

func (d *dedicatedNodeGroupsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	var ok bool
	if d.provider, ok = req.ProviderData.(*tidbcloudProvider); !ok {
		resp.Diagnostics.AddError("Internal provider error",
			fmt.Sprintf("Error in Configure: expected %T but got %T", tidbcloudProvider{}, req.ProviderData))
	}
}

func (d *dedicatedNodeGroupsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "dedicated node groups data source",
		Attributes: map[string]schema.Attribute{
			"cluster_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the cluster.",
				Required:            true,
			},
			"node_spec_key": schema.StringAttribute{
				MarkdownDescription: "The key of the node spec.",
				Computed:            true,
			},
			"node_groups": schema.ListNestedAttribute{
				MarkdownDescription: "The node groups.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"node_group_id": schema.StringAttribute{
							MarkdownDescription: "The ID of the node group.",
							Computed:            true,
						},
						"node_count": schema.Int64Attribute{
							MarkdownDescription: "The number of nodes in the node group.",
							Computed:            true,
						},
						"display_name": schema.StringAttribute{
							MarkdownDescription: "The display name of the region.",
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
					},
				},
			},
		},
	}
}

func (d *dedicatedNodeGroupsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data dedicatedNodeGroupsDataSourceData
	diags := req.Config.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Trace(ctx, "read node group data source")
	nodeGroups, err := d.provider.DedicatedClient.ListTiDBNodeGroups(ctx, data.ClusterId.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Read Error", fmt.Sprintf("Unable to call GetTiDBNodeGroup, got error: %s", err))
		return
	}
	tflog.Debug(ctx, fmt.Sprintf("nodeGroups: %v", nodeGroups))
	var items []nodeGroupItems
	for _, nodeGroup := range nodeGroups {
		if *nodeGroup.IsDefaultGroup {
			data.NodeSpecKey = types.StringValue(*nodeGroup.NodeSpecKey)
		}
		items = append(items, nodeGroupItems{
			NodeCount:           types.Int64Value(int64(nodeGroup.NodeCount)),
			NodeGroupId:         types.StringValue(*nodeGroup.TidbNodeGroupId),
			DisplayName:         types.StringValue(*nodeGroup.DisplayName),
			NodeSpecDisplayName: types.StringValue(*nodeGroup.NodeSpecDisplayName),
			IsDefaultGroup:      types.BoolValue(*nodeGroup.IsDefaultGroup),
			State:               types.StringValue(string(*nodeGroup.State)),
		})
	}
	data.NodeGroups = items

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}
