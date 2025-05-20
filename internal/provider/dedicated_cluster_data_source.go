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

type dedicatedClusterDataSourceData struct {
	ProjectId          types.String        `tfsdk:"project_id"`
	ClusterId          types.String        `tfsdk:"cluster_id"`
	DisplayName        types.String        `tfsdk:"display_name"`
	CloudProvider      types.String        `tfsdk:"cloud_provider"`
	RegionId           types.String        `tfsdk:"region_id"`
	Labels             types.Map           `tfsdk:"labels"`
	Port               types.Int32         `tfsdk:"port"`
	PausePlan          *pausePlan          `tfsdk:"pause_plan"`
	State              types.String        `tfsdk:"state"`
	Version            types.String        `tfsdk:"version"`
	CreatedBy          types.String        `tfsdk:"created_by"`
	CreateTime         types.String        `tfsdk:"create_time"`
	UpdateTime         types.String        `tfsdk:"update_time"`
	RegionDisplayName  types.String        `tfsdk:"region_display_name"`
	Annotations        types.Map           `tfsdk:"annotations"`
	TiDBNodeSetting    *tidbNodeSetting    `tfsdk:"tidb_node_setting"`
	TiKVNodeSetting    *tikvNodeSetting    `tfsdk:"tikv_node_setting"`
	TiFlashNodeSetting *tiflashNodeSetting `tfsdk:"tiflash_node_setting"`
}

var _ datasource.DataSource = &dedicatedClusterDataSource{}

type dedicatedClusterDataSource struct {
	provider *tidbcloudProvider
}

func NewDedicatedClusterDataSource() datasource.DataSource {
	return &dedicatedClusterDataSource{}
}

func (d *dedicatedClusterDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dedicated_cluster"
}

func (d *dedicatedClusterDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	var ok bool
	if d.provider, ok = req.ProviderData.(*tidbcloudProvider); !ok {
		resp.Diagnostics.AddError("Internal provider error",
			fmt.Sprintf("Error in Configure: expected %T but got %T", tidbcloudProvider{}, req.ProviderData))
	}
}

func (d *dedicatedClusterDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "dedicated cluster data source",
		Attributes: map[string]schema.Attribute{
			"project_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the project.",
				Computed:            true,
			},
			"cluster_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the cluster.",
				Required:            true,
			},
			"display_name": schema.StringAttribute{
				MarkdownDescription: "The name of the cluster.",
				Computed:            true,
			},
			"cloud_provider": schema.StringAttribute{
				MarkdownDescription: "The cloud provider on which your cluster is hosted.",
				Computed:            true,
			},
			"region_id": schema.StringAttribute{
				MarkdownDescription: "The region where the cluster is deployed.",
				Computed:            true,
			},
			"labels": schema.MapAttribute{
				MarkdownDescription: "A map of labels assigned to the cluster.",
				Computed:            true,
				ElementType:         types.StringType,
			},
			"port": schema.Int32Attribute{
				MarkdownDescription: "The port used for accessing the cluster.",
				Computed:            true,
			},
			"pause_plan": schema.SingleNestedAttribute{
				MarkdownDescription: "Pause plan details for the cluster.",
				Computed:            true,
				Attributes: map[string]schema.Attribute{
					"pause_type": schema.StringAttribute{
						MarkdownDescription: "The type of pause.",
						Computed:            true,
					},
					"scheduled_resume_time": schema.StringAttribute{
						MarkdownDescription: "The scheduled time for resuming the cluster.",
						Computed:            true,
					},
				},
			},
			"state": schema.StringAttribute{
				MarkdownDescription: "The current state of the cluster.",
				Computed:            true,
			},
			"version": schema.StringAttribute{
				MarkdownDescription: "The version of the cluster.",
				Computed:            true,
			},
			"created_by": schema.StringAttribute{
				MarkdownDescription: "The creator of the cluster.",
				Computed:            true,
			},
			"create_time": schema.StringAttribute{
				MarkdownDescription: "The creation time of the cluster.",
				Computed:            true,
			},
			"update_time": schema.StringAttribute{
				MarkdownDescription: "The last update time of the cluster.",
				Computed:            true,
			},
			"region_display_name": schema.StringAttribute{
				MarkdownDescription: "The display name of the region.",
				Computed:            true,
			},
			"annotations": schema.MapAttribute{
				MarkdownDescription: "A map of annotations for the cluster.",
				Computed:            true,
				ElementType:         types.StringType,
			},
			"tidb_node_setting": schema.SingleNestedAttribute{
				MarkdownDescription: "Settings for TiDB nodes.",
				Computed:            true,
				Attributes: map[string]schema.Attribute{
					"node_spec_key": schema.StringAttribute{
						MarkdownDescription: "The key of the node spec.",
						Computed:            true,
					},
					"node_count": schema.Int32Attribute{
						MarkdownDescription: "The number of nodes in the default node group.",
						Computed:            true,
					},
					"node_group_id": schema.StringAttribute{
						MarkdownDescription: "The ID of the default node group.",
						Computed:            true,
					},
					"node_group_display_name": schema.StringAttribute{
						MarkdownDescription: "The display name of the default node group.",
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
			"tikv_node_setting": schema.SingleNestedAttribute{
				MarkdownDescription: "Settings for TiKV nodes.",
				Computed:            true,
				Attributes: map[string]schema.Attribute{
					"node_spec_key": schema.StringAttribute{
						MarkdownDescription: "The node specification key.",
						Computed:            true,
					},
					"node_count": schema.Int32Attribute{
						MarkdownDescription: "The number of nodes in the cluster.",
						Computed:            true,
					},
					"storage_size_gi": schema.Int32Attribute{
						MarkdownDescription: "The storage size in GiB.",
						Computed:            true,
					},
					"storage_type": schema.StringAttribute{
						MarkdownDescription: "The storage type.",
						Computed:            true,
					},
					"node_spec_display_name": schema.StringAttribute{
						MarkdownDescription: "The display name of the node spec.",
						Computed:            true,
					},
					"raft_store_iops": schema.Int32Attribute{
						MarkdownDescription: "The IOPS of raft store",
						Computed:            true,
					},
				},
			},
			"tiflash_node_setting": schema.SingleNestedAttribute{
				MarkdownDescription: "Settings for TiFlash nodes.",
				Computed:            true,
				Attributes: map[string]schema.Attribute{
					"node_spec_key": schema.StringAttribute{
						MarkdownDescription: "The node specification key.",
						Computed:            true,
					},
					"node_count": schema.Int32Attribute{
						MarkdownDescription: "The number of nodes in the cluster.",
						Computed:            true,
					},
					"storage_size_gi": schema.Int32Attribute{
						MarkdownDescription: "The storage size in GiB.",
						Computed:            true,
					},
					"storage_type": schema.StringAttribute{
						MarkdownDescription: "The storage type.",
						Computed:            true,
					},
					"node_spec_display_name": schema.StringAttribute{
						MarkdownDescription: "The display name of the node spec.",
						Computed:            true,
					},
					"raft_store_iops": schema.Int32Attribute{
						MarkdownDescription: "The IOPS of raft store",
						Computed:            true,
					},
				},
			},
		},
	}
}

func (d *dedicatedClusterDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data dedicatedClusterDataSourceData
	diags := req.Config.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Trace(ctx, "read dedicated cluster data source")
	cluster, err := d.provider.DedicatedClient.GetCluster(ctx, data.ClusterId.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Read Error", fmt.Sprintf("Unable to call GetCluster, got error: %s", err))
		return
	}

	labels, diag := types.MapValueFrom(ctx, types.StringType, *cluster.Labels)
	if diag.HasError() {
		return
	}
	annotations, diag := types.MapValueFrom(ctx, types.StringType, *cluster.Annotations)
	if diag.HasError() {
		return
	}

	data.ClusterId = types.StringValue(*cluster.ClusterId)
	data.DisplayName = types.StringValue(cluster.DisplayName)
	data.CloudProvider = types.StringValue(string(*cluster.CloudProvider))
	data.RegionId = types.StringValue(cluster.RegionId)
	data.Labels = labels
	data.Port = types.Int32Value(cluster.Port)
	data.State = types.StringValue(string(*cluster.State))
	data.Version = types.StringValue(*cluster.Version)
	data.CreatedBy = types.StringValue(*cluster.CreatedBy)
	data.CreateTime = types.StringValue(cluster.CreateTime.String())
	data.UpdateTime = types.StringValue(cluster.UpdateTime.String())
	data.RegionDisplayName = types.StringValue(*cluster.RegionDisplayName)
	data.Annotations = annotations
	data.ProjectId = types.StringValue((*cluster.Labels)[LabelsKeyProjectId])

	// tidb node setting
	for _, group := range cluster.TidbNodeSetting.TidbNodeGroups {
		if *group.IsDefaultGroup {
			var endpoints []attr.Value
			for _, e := range group.Endpoints {
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

			defaultTiProxySetting := tiProxySetting{}
			if group.TiproxySetting != nil {
				defaultTiProxySetting = tiProxySetting{
					Type:      types.StringValue(string(*group.TiproxySetting.Type)),
					NodeCount: types.Int32Value(*group.TiproxySetting.NodeCount.Get()),
				}
			}
			publicEndpointSetting, err := d.provider.DedicatedClient.GetPublicEndpoint(ctx, data.ClusterId.ValueString(), *group.TidbNodeGroupId)
			if err != nil {
				resp.Diagnostics.AddError("Read Error", fmt.Sprintf("Unable to call GetPublicEndpoint, got error: %s", err))
				return
			}
			data.TiDBNodeSetting = &tidbNodeSetting{
				NodeSpecKey:           types.StringValue(*group.NodeSpecKey),
				NodeCount:             types.Int32Value(group.NodeCount),
				NodeGroupId:           types.StringValue(*group.TidbNodeGroupId),
				NodeGroupDisplayName:  types.StringValue(*group.DisplayName),
				NodeSpecDisplayName:   types.StringValue(*group.NodeSpecDisplayName),
				IsDefaultGroup:        types.BoolValue(*group.IsDefaultGroup),
				State:                 types.StringValue(string(*group.State)),
				Endpoints:             endpointsList,
				TiProxySetting:        &defaultTiProxySetting,
				PublicEndpointSetting: convertDedicatedPublicEndpointSetting(publicEndpointSetting),
			}
		}
	}

	data.TiKVNodeSetting = &tikvNodeSetting{
		NodeSpecKey:         types.StringValue(cluster.TikvNodeSetting.NodeSpecKey),
		NodeCount:           types.Int32Value(cluster.TikvNodeSetting.NodeCount),
		StorageSizeGi:       types.Int32Value(cluster.TikvNodeSetting.StorageSizeGi),
		StorageType:         types.StringValue(string(*cluster.TikvNodeSetting.StorageType)),
		NodeSpecDisplayName: types.StringValue(*cluster.TikvNodeSetting.NodeSpecDisplayName),
	}
	if cluster.TikvNodeSetting.RaftStoreIops.IsSet() {
		data.TiKVNodeSetting.RaftStoreIOPS = types.Int32Value(*cluster.TikvNodeSetting.RaftStoreIops.Get())
	}

	// tiflash node setting
	if cluster.TiflashNodeSetting != nil {
		data.TiFlashNodeSetting = &tiflashNodeSetting{
			NodeSpecKey:         types.StringValue(cluster.TiflashNodeSetting.NodeSpecKey),
			NodeCount:           types.Int32Value(cluster.TiflashNodeSetting.NodeCount),
			StorageSizeGi:       types.Int32Value(cluster.TiflashNodeSetting.StorageSizeGi),
			StorageType:         types.StringValue(string(*cluster.TiflashNodeSetting.StorageType)),
			NodeSpecDisplayName: types.StringValue(*cluster.TiflashNodeSetting.NodeSpecDisplayName),
		}
		if cluster.TiflashNodeSetting.RaftStoreIops.IsSet() {
			data.TiFlashNodeSetting.RaftStoreIOPS = types.Int32Value(*cluster.TiflashNodeSetting.RaftStoreIops.Get())
		}
	}

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}
