package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type dedicatedNodeGroupDataSourceData struct {
	ClusterId             types.String           `tfsdk:"cluster_id"`
	NodeSpecKey           types.String           `tfsdk:"node_spec_key"`
	NodeCount             types.Int64            `tfsdk:"node_count"`
	NodeGroupId           types.String           `tfsdk:"node_group_id"`
	DisplayName           types.String           `tfsdk:"display_name"`
	NodeSpecDisplayName   types.String           `tfsdk:"node_spec_display_name"`
	IsDefaultGroup        types.Bool             `tfsdk:"is_default_group"`
	State                 types.String           `tfsdk:"state"`
	Endpoints             types.List             `tfsdk:"endpoints"`
	TiProxySetting        *tiProxySetting        `tfsdk:"tiproxy_setting"`
	PublicEndpointSetting *publicEndpointSetting `tfsdk:"public_endpoint_setting"`
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
			"endpoints": schema.ListAttribute{
				MarkdownDescription: "The endpoints of the node group.",
				Computed:            true,
				ElementType:         types.ObjectType{AttrTypes: endpointItemAttrTypes},
			},
			"tiproxy_setting": schema.SingleNestedAttribute{
				MarkdownDescription: "Settings for TiProxy nodes.",
				Computed:            true,
				Attributes: map[string]schema.Attribute{
					"node_spec_key": schema.StringAttribute{
						MarkdownDescription: "The key of the node spec.",
						Computed:            true,
					},
					"node_spec_version": schema.StringAttribute{
						MarkdownDescription: "The node specification version.",
						Computed:            true,
					},
					"node_count": schema.Int32Attribute{
						MarkdownDescription: "The number of TiProxy nodes.",
						Computed:            true,
					},
					"node_spec_display_name": schema.StringAttribute{
						MarkdownDescription: "The display name of the node spec.",
						Computed:            true,
					},
				},
			},
			"public_endpoint_setting": schema.SingleNestedAttribute{
				MarkdownDescription: "Settings for public endpoint.",
				Computed:            true,
				Attributes: map[string]schema.Attribute{
					"enabled": schema.BoolAttribute{
						MarkdownDescription: "Whether public endpoint is enabled.",
						Computed:            true,
					},
					"ip_access_list": schema.ListAttribute{
						MarkdownDescription: "IP access list for the public endpoint.",
						Computed:            true,
						ElementType:         types.ObjectType{AttrTypes: ipAccessListItemAttrTypes},
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
	publicEndpointSetting, err := d.provider.DedicatedClient.GetPublicEndpoint(ctx, data.ClusterId.ValueString(), data.NodeGroupId.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Read Error", fmt.Sprintf("Unable to call GetPublicEndpoint, got error: %s", err))
		return
	}

	data.NodeSpecKey = types.StringValue(string(*nodeGroup.NodeSpecKey))
	data.NodeCount = types.Int64Value(int64(nodeGroup.NodeCount))
	data.DisplayName = types.StringValue(string(*nodeGroup.DisplayName))
	data.NodeSpecDisplayName = types.StringValue(string(*nodeGroup.NodeSpecDisplayName))
	data.IsDefaultGroup = types.BoolValue(bool(*nodeGroup.IsDefaultGroup))
	data.State = types.StringValue(string(*nodeGroup.State))
	var endpoints []attr.Value
	for _, e := range nodeGroup.Endpoints {
		endpointObj, objDiags := types.ObjectValue(
			endpointItemAttrTypes,
			map[string]attr.Value{
				"host":            types.StringValue(*e.Host),
				"port":            types.Int32Value(*e.Port),
				"connection_type": types.StringValue(string(*e.ConnectionType)),
			},
		)
		diags.Append(objDiags...)
		endpoints = append(endpoints, endpointObj)
	}
	endpointsList, listDiags := types.ListValue(types.ObjectType{
		AttrTypes: endpointItemAttrTypes,
	}, endpoints)
	diags.Append(listDiags...)
	data.Endpoints = endpointsList
	if nodeGroup.TiproxySetting != nil {
		tiProxy := tiProxySetting{
			NodeSpecKey:         types.StringValue(nodeGroup.TiproxySetting.NodeSpecKey),
			NodeSpecVersion:     types.StringValue(*nodeGroup.TiproxySetting.NodeSpecVersion),
			NodeCount:           types.Int32Value(*nodeGroup.TiproxySetting.NodeCount.Get()),
			NodeSpecDisplayName: types.StringValue(*nodeGroup.TiproxySetting.NodeSpecDisplayName),
		}
		data.TiProxySetting = &tiProxy
	}
	data.PublicEndpointSetting = convertDedicatedPublicEndpointSetting(publicEndpointSetting)

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}
