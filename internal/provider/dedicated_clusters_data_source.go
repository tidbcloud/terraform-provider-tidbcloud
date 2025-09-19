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

type dedicatedClustersDataSourceData struct {
	ProjectId types.String                     `tfsdk:"project_id"`
	Clusters  []dedicatedClusterDataSourceData `tfsdk:"clusters"`
}

var _ datasource.DataSource = &dedicatedClustersDataSource{}

type dedicatedClustersDataSource struct {
	provider *tidbcloudProvider
}

func NewDedicatedClustersDataSource() datasource.DataSource {
	return &dedicatedClustersDataSource{}
}

func (d *dedicatedClustersDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dedicated_clusters"
}

func (d *dedicatedClustersDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	var ok bool
	if d.provider, ok = req.ProviderData.(*tidbcloudProvider); !ok {
		resp.Diagnostics.AddError("Internal provider error",
			fmt.Sprintf("Error in Configure: expected %T but got %T", tidbcloudProvider{}, req.ProviderData))
	}
}

func (d *dedicatedClustersDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "dedicated clusters data source",
		Attributes: map[string]schema.Attribute{
			"project_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the project.",
				Optional:            true,
			},
			"clusters": schema.ListNestedAttribute{
				MarkdownDescription: "The clusters.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"project_id": schema.StringAttribute{
							MarkdownDescription: "The ID of the project.",
							Computed:            true,
						},
						"cluster_id": schema.StringAttribute{
							MarkdownDescription: "The ID of the cluster.",
							Computed:            true,
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
								"endpoints": schema.ListAttribute{
									MarkdownDescription: "The endpoints of the node group.",
									Computed:            true,
									ElementType:         types.ObjectType{AttrTypes: endpointItemAttrTypes},
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
				},
			},
		},
	}
}

func (d *dedicatedClustersDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data dedicatedClustersDataSourceData
	diags := req.Config.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Trace(ctx, "read dedicated clusters data source")
	clusters, err := d.retrieveClusters(ctx, data.ProjectId.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Read Error", fmt.Sprintf("Unable to call ListClusters, got error: %s", err))
		return
	}
	var items []dedicatedClusterDataSourceData
	for _, cluster := range clusters {
		var c dedicatedClusterDataSourceData
		labels, diag := types.MapValueFrom(ctx, types.StringType, *cluster.Labels)
		if diag.HasError() {
			return
		}
		annotations, diag := types.MapValueFrom(ctx, types.StringType, *cluster.Annotations)
		if diag.HasError() {
			return
		}

		c.ClusterId = types.StringValue(*cluster.ClusterId)
		c.DisplayName = types.StringValue(cluster.DisplayName)
		c.CloudProvider = types.StringValue(string(*cluster.CloudProvider))
		c.RegionId = types.StringValue(cluster.RegionId)
		c.Labels = labels
		c.Port = types.Int32Value(cluster.Port)
		c.State = types.StringValue(string(*cluster.State))
		c.Version = types.StringValue(*cluster.Version)
		c.CreatedBy = types.StringValue(*cluster.CreatedBy)
		c.CreateTime = types.StringValue(cluster.CreateTime.String())
		c.UpdateTime = types.StringValue(cluster.UpdateTime.String())
		c.RegionDisplayName = types.StringValue(*cluster.RegionDisplayName)
		c.Annotations = annotations
		c.ProjectId = types.StringValue((*cluster.Labels)[LabelsKeyProjectId])

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

				publicEndpointSetting, err := d.provider.DedicatedClient.GetPublicEndpoint(ctx, c.ClusterId.ValueString(), *group.TidbNodeGroupId)
				if err != nil {
					resp.Diagnostics.AddError("Read Error", fmt.Sprintf("Unable to call GetPublicEndpoint, got error: %s", err))
					return
				}

				c.TiDBNodeSetting = &tidbNodeSetting{
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

		c.TiKVNodeSetting = &tikvNodeSetting{
			NodeSpecKey:         types.StringValue(cluster.TikvNodeSetting.NodeSpecKey),
			NodeCount:           types.Int32Value(cluster.TikvNodeSetting.NodeCount),
			StorageSizeGi:       types.Int32Value(cluster.TikvNodeSetting.StorageSizeGi),
			StorageType:         types.StringValue(string(*cluster.TikvNodeSetting.StorageType)),
			NodeSpecDisplayName: types.StringValue(*cluster.TikvNodeSetting.NodeSpecDisplayName),
		}
		if cluster.TikvNodeSetting.RaftStoreIops.IsSet() {
			c.TiKVNodeSetting.RaftStoreIOPS = types.Int32Value(*cluster.TikvNodeSetting.RaftStoreIops.Get())
		}

		// tiflash node setting
		if cluster.TiflashNodeSetting != nil {
			c.TiFlashNodeSetting = &tiflashNodeSetting{
				NodeSpecKey:         types.StringValue(cluster.TiflashNodeSetting.NodeSpecKey),
				NodeCount:           types.Int32Value(cluster.TiflashNodeSetting.NodeCount),
				StorageSizeGi:       types.Int32Value(cluster.TiflashNodeSetting.StorageSizeGi),
				StorageType:         types.StringValue(string(*cluster.TiflashNodeSetting.StorageType)),
				NodeSpecDisplayName: types.StringValue(*cluster.TiflashNodeSetting.NodeSpecDisplayName),
			}
			if cluster.TiflashNodeSetting.RaftStoreIops.IsSet() {
				c.TiFlashNodeSetting.RaftStoreIOPS = types.Int32Value(*cluster.TiflashNodeSetting.RaftStoreIops.Get())
			}
		}

		if cluster.PausePlan != nil {
			p := pausePlan{
				PauseType: types.StringValue(string(cluster.PausePlan.PauseType)),
			}
			if cluster.PausePlan.ScheduledResumeTime != nil {
				p.ScheduledResumeTime = types.StringValue(cluster.PausePlan.ScheduledResumeTime.String())
			}
			c.PausePlan, diags = types.ObjectValueFrom(ctx, pausePlanAttrTypes, p)
		} else {
			c.PausePlan = types.ObjectNull(pausePlanAttrTypes)
		}

		items = append(items, c)
	}
	data.Clusters = items

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (d *dedicatedClustersDataSource) retrieveClusters(ctx context.Context, projectId string) ([]dedicated.TidbCloudOpenApidedicatedv1beta1Cluster, error) {
	var items []dedicated.TidbCloudOpenApidedicatedv1beta1Cluster
	pageSizeInt32 := int32(DefaultPageSize)
	var pageToken *string

	clusters, err := d.provider.DedicatedClient.ListClusters(ctx, projectId, &pageSizeInt32, nil)
	if err != nil {
		return nil, errors.Trace(err)
	}
	items = append(items, clusters.Clusters...)
	for {
		pageToken = clusters.NextPageToken
		if IsNilOrEmpty(pageToken) {
			break
		}
		clusters, err = d.provider.DedicatedClient.ListClusters(ctx, projectId, &pageSizeInt32, pageToken)
		if err != nil {
			return nil, errors.Trace(err)
		}
		items = append(items, clusters.Clusters...)
	}
	return items, nil
}
