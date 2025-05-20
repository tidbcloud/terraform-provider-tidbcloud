package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/juju/errors"
	"github.com/tidbcloud/tidbcloud-cli/pkg/tidbcloud/v1beta1/dedicated"
)

type dedicatedNodeGroupsDataSourceData struct {
	ClusterId   types.String     `tfsdk:"cluster_id"`
	NodeSpecKey types.String     `tfsdk:"node_spec_key"`
	NodeGroups  []nodeGroupItems `tfsdk:"node_groups"`
}

type nodeGroupItems struct {
	NodeCount             types.Int32            `tfsdk:"node_count"`
	NodeGroupId           types.String           `tfsdk:"node_group_id"`
	DisplayName           types.String           `tfsdk:"display_name"`
	NodeSpecDisplayName   types.String           `tfsdk:"node_spec_display_name"`
	IsDefaultGroup        types.Bool             `tfsdk:"is_default_group"`
	State                 types.String           `tfsdk:"state"`
	Endpoints             types.List             `tfsdk:"endpoints"`
	TiProxySetting        *tiProxySetting        `tfsdk:"tiproxy_setting"`
	PublicEndpointSetting *publicEndpointSetting `tfsdk:"public_endpoint_setting"`
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
						"node_count": schema.Int32Attribute{
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
						"tiproxy_setting": schema.SingleNestedAttribute{
							MarkdownDescription: "Settings for TiProxy nodes.",
							Computed:            true,
							Attributes: map[string]schema.Attribute{
								"type": schema.StringAttribute{
									MarkdownDescription: "The type of TiProxy nodes." +
										"- SMALL: Low performance instance with 2 vCPUs and 4 GiB memory. Max QPS: 30, Max Data Traffic: 90 MiB/s." +
										"- LARGE: High performance instance with 8 vCPUs and 16 GiB memory. Max QPS: 100, Max Data Traffic: 300 MiB/s.",
									Computed: true,
								},
								"node_count": schema.Int32Attribute{
									MarkdownDescription: "The number of TiProxy nodes.",
									Computed:            true,
								},
							},
						},
						"public_endpoint_setting": schema.SingleNestedAttribute{
							MarkdownDescription: "Settings for public endpoints.",
							Computed:            true,
							Attributes: map[string]schema.Attribute{
								"enabled": schema.BoolAttribute{
									MarkdownDescription: "Whether public endpoints are enabled.",
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
	nodeGroups, err := d.retrieveTiDBNodeGroups(ctx, data.ClusterId.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Read Error", fmt.Sprintf("Unable to call GetTiDBNodeGroup, got error: %s", err))
		return
	}
	var items []nodeGroupItems
	for _, nodeGroup := range nodeGroups {
		if *nodeGroup.IsDefaultGroup {
			data.NodeSpecKey = types.StringValue(*nodeGroup.NodeSpecKey)
		}

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

		tiProxy := tiProxySetting{}
		if nodeGroup.TiproxySetting != nil {
			tiProxy = tiProxySetting{
				Type:      types.StringValue(string(*nodeGroup.TiproxySetting.Type)),
				NodeCount: types.Int32Value(*nodeGroup.TiproxySetting.NodeCount.Get()),
			}
		}
		publicEndpointSetting, err := d.provider.DedicatedClient.GetPublicEndpoint(ctx, data.ClusterId.ValueString(), *nodeGroup.TidbNodeGroupId)
		if err != nil {
			resp.Diagnostics.AddError("Read Error", fmt.Sprintf("Unable to call GetPublicEndpoint, got error: %s", err))
			return
		}

		items = append(items, nodeGroupItems{
			NodeCount:             types.Int32Value(nodeGroup.NodeCount),
			NodeGroupId:           types.StringValue(*nodeGroup.TidbNodeGroupId),
			DisplayName:           types.StringValue(*nodeGroup.DisplayName),
			NodeSpecDisplayName:   types.StringValue(*nodeGroup.NodeSpecDisplayName),
			IsDefaultGroup:        types.BoolValue(*nodeGroup.IsDefaultGroup),
			State:                 types.StringValue(string(*nodeGroup.State)),
			Endpoints:             endpointsList,
			TiProxySetting:        &tiProxy,
			PublicEndpointSetting: convertDedicatedPublicEndpointSetting(publicEndpointSetting),
		})
	}
	data.NodeGroups = items

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (d *dedicatedNodeGroupsDataSource) retrieveTiDBNodeGroups(ctx context.Context, projectId string) ([]dedicated.Dedicatedv1beta1TidbNodeGroup, error) {
	var items []dedicated.Dedicatedv1beta1TidbNodeGroup
	pageSizeInt32 := int32(DefaultPageSize)
	var pageToken *string

	nodeGroups, err := d.provider.DedicatedClient.ListTiDBNodeGroups(ctx, projectId, &pageSizeInt32, nil)
	if err != nil {
		return nil, errors.Trace(err)
	}
	items = append(items, nodeGroups.TidbNodeGroups...)
	for {
		pageToken = nodeGroups.NextPageToken
		if IsNilOrEmpty(pageToken) {
			break
		}
		nodeGroups, err = d.provider.DedicatedClient.ListTiDBNodeGroups(ctx, projectId, &pageSizeInt32, pageToken)
		if err != nil {
			return nil, errors.Trace(err)
		}
		items = append(items, nodeGroups.TidbNodeGroups...)
	}
	return items, nil
}
