package provider

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"math/rand"
	"strconv"
)

type clustersDataSourceData struct {
	Id        types.String   `tfsdk:"id"`
	Page      types.Int64    `tfsdk:"page"`
	PageSize  types.Int64    `tfsdk:"page_size"`
	Clusters  []clusterItems `tfsdk:"items"`
	Total     types.Int64    `tfsdk:"total"`
	ProjectId types.String   `tfsdk:"project_id"`
}

type clusterItems struct {
	Id              string                   `tfsdk:"id"`
	ProjectId       string                   `tfsdk:"project_id"`
	Name            string                   `tfsdk:"name"`
	CloudProvider   string                   `tfsdk:"cloud_provider"`
	ClusterType     string                   `tfsdk:"cluster_type"`
	Region          string                   `tfsdk:"region"`
	CreateTimestamp string                   `tfsdk:"create_timestamp"`
	Config          *clusterConfigDataSource `tfsdk:"config"`
	Status          *clusterStatusDataSource `tfsdk:"status"`
}

type clusterConfigDataSource struct {
	Port       int64       `tfsdk:"port"`
	Components *components `tfsdk:"components"`
}

type clusterStatusDataSource struct {
	TidbVersion       string      `tfsdk:"tidb_version"`
	ClusterStatus     string      `tfsdk:"cluster_status"`
	ConnectionStrings *connection `tfsdk:"connection_strings"`
}

type connection struct {
	DefaultUser string                `tfsdk:"default_user"`
	Standard    *connectionStandard   `tfsdk:"standard"`
	VpcPeering  *connectionVpcPeering `tfsdk:"vpc_peering"`
}

type connectionStandard struct {
	Host string `tfsdk:"host"`
	Port int64  `tfsdk:"port"`
}

type connectionVpcPeering struct {
	Host string `tfsdk:"host"`
	Port int64  `tfsdk:"port"`
}

// Ensure provider defined types fully satisfy framework interfaces
var _ provider.DataSourceType = clustersDataSourceType{}
var _ datasource.DataSource = clustersDataSource{}

type clustersDataSourceType struct{}

func (t clustersDataSourceType) GetSchema(ctx context.Context) (tfsdk.Schema, diag.Diagnostics) {
	return tfsdk.Schema{
		MarkdownDescription: "clusters data source",
		Attributes: map[string]tfsdk.Attribute{
			"id": {
				MarkdownDescription: "data source ID.",
				Computed:            true,
				Type:                types.StringType,
			},
			"page": {
				MarkdownDescription: "Default:1 The number of pages.",
				Optional:            true,
				Computed:            true,
				Type:                types.Int64Type,
			},
			"page_size": {
				MarkdownDescription: "Default:10 The size of a pages.",
				Optional:            true,
				Computed:            true,
				Type:                types.Int64Type,
			},
			"project_id": {
				MarkdownDescription: "The ID of the project",
				Required:            true,
				Type:                types.StringType,
			},
			"items": {
				MarkdownDescription: "The items of clusters in the project.",
				Computed:            true,
				Attributes: tfsdk.ListNestedAttributes(map[string]tfsdk.Attribute{
					"id": {
						MarkdownDescription: "The ID of the cluster.",
						Computed:            true,
						Type:                types.StringType,
					},
					"project_id": {
						MarkdownDescription: "The ID of the project.",
						Computed:            true,
						Type:                types.StringType,
					},
					"name": {
						MarkdownDescription: "The name of the cluster.",
						Computed:            true,
						Type:                types.StringType,
					},
					"cluster_type": {
						MarkdownDescription: "The cluster type.",
						Computed:            true,
						Type:                types.StringType,
					},
					"cloud_provider": {
						MarkdownDescription: "Enum: \"AWS\" \"GCP\", The cloud provider on which your TiDB cluster is hosted.",
						Computed:            true,
						Type:                types.StringType,
					},
					"region": {
						MarkdownDescription: "Region of the cluster.",
						Computed:            true,
						Type:                types.StringType,
					},
					"create_timestamp": {
						MarkdownDescription: "The creation time of the cluster in Unix timestamp seconds (epoch time).",
						Computed:            true,
						Type:                types.StringType,
					},
					"config": {
						MarkdownDescription: "The configuration of the cluster.",
						Computed:            true,
						Attributes: tfsdk.SingleNestedAttributes(map[string]tfsdk.Attribute{
							"port": {
								MarkdownDescription: "The TiDB port for connection. The port must be in the range of 1024-65535 except 10080, 4000 in default.\n" +
									"  - For a Serverless Tier cluster, only port 4000 is available.",
								Computed: true,
								Type:     types.Int64Type,
							},
							"components": {
								MarkdownDescription: "The components of the cluster.",
								Computed:            true,
								Attributes: tfsdk.SingleNestedAttributes(map[string]tfsdk.Attribute{
									"tidb": {
										MarkdownDescription: "The TiDB component of the cluster",
										Computed:            true,
										Attributes: tfsdk.SingleNestedAttributes(map[string]tfsdk.Attribute{
											"node_size": {
												Computed: true,
												MarkdownDescription: "The size of the TiDB component in the cluster, You can get the available node size of each region from the [tidbcloud_cluster_specs datasource](../data-sources/cluster_specs.md).\n" +
													"  - If the vCPUs of TiDB or TiKV component is 2 or 4, then their vCPUs need to be the same.\n" +
													"  - If the vCPUs of TiDB or TiKV component is 2 or 4, then the cluster does not support TiFlash.\n" +
													"  - Can not modify node_size of an existing cluster.",
												Type: types.StringType,
											},
											"node_quantity": {
												MarkdownDescription: "The number of nodes in the cluster. You can get the minimum and step of a node quantity from the [tidbcloud_cluster_specs datasource](../data-sources/cluster_specs.md).",
												Computed:            true,
												Type:                types.Int64Type,
											},
										}),
									},
									"tikv": {
										MarkdownDescription: "The TiKV component of the cluster",
										Computed:            true,
										Attributes: tfsdk.SingleNestedAttributes(map[string]tfsdk.Attribute{
											"node_size": {
												MarkdownDescription: "The size of the TiKV component in the cluster, You can get the available node size of each region from the [tidbcloud_cluster_specs datasource](../data-sources/cluster_specs.md).\n" +
													"  - If the vCPUs of TiDB or TiKV component is 2 or 4, then their vCPUs need to be the same.\n" +
													"  - If the vCPUs of TiDB or TiKV component is 2 or 4, then the cluster does not support TiFlash.\n" +
													"  - Can not modify node_size of an existing cluster.",
												Computed: true,
												Type:     types.StringType,
											},
											"storage_size_gib": {
												MarkdownDescription: "The storage size of a node in the cluster. You can get the minimum and maximum of storage size from the [tidbcloud_cluster_specs datasource](../data-sources/cluster_specs.md).\n" +
													"  - Can not modify storage_size_gib of an existing cluster.",
												Computed: true,
												Type:     types.Int64Type,
											},
											"node_quantity": {
												MarkdownDescription: "The number of nodes in the cluster. You can get the minimum and step of a node quantity from the [tidbcloud_cluster_specs datasource](../data-sources/cluster_specs.md).\n" +
													"  - TiKV do not support decreasing node quantity.\n" +
													"  - The node_quantity of TiKV must be a multiple of 3.",
												Computed: true,
												Type:     types.Int64Type,
											},
										}),
									},
									"tiflash": {
										MarkdownDescription: "The TiFlash component of the cluster.",
										Computed:            true,
										Optional:            true,
										Attributes: tfsdk.SingleNestedAttributes(map[string]tfsdk.Attribute{
											"node_size": {
												MarkdownDescription: "The size of the TiFlash component in the cluster, You can get the available node size of each region from the [tidbcloud_cluster_specs datasource](../data-sources/cluster_specs.md).\n" +
													"  - Can not modify node_size of an existing cluster.",
												Computed: true,
												Type:     types.StringType,
											},
											"storage_size_gib": {
												MarkdownDescription: "The storage size of a node in the cluster. You can get the minimum and maximum of storage size from the [tidbcloud_cluster_specs datasource](../data-sources/cluster_specs.md).\n" +
													"  - Can not modify storage_size_gib of an existing cluster.",
												Computed: true,
												Type:     types.Int64Type,
											},
											"node_quantity": {
												MarkdownDescription: "The number of nodes in the cluster. You can get the minimum and step of a node quantity from the [tidbcloud_cluster_specs datasource](../data-sources/cluster_specs.md).\n" +
													"  - TiFlash do not support decreasing node quantity.",
												Computed: true,
												Type:     types.Int64Type,
											},
										}),
									},
								}),
							},
						}),
					},
					"status": {
						MarkdownDescription: "The status of the cluster.",
						Computed:            true,
						Attributes: tfsdk.SingleNestedAttributes(map[string]tfsdk.Attribute{
							"tidb_version": {
								MarkdownDescription: "TiDB version.",
								Computed:            true,
								Type:                types.StringType,
							},
							"cluster_status": {
								MarkdownDescription: "Status of the cluster.",
								Computed:            true,
								Type:                types.StringType,
							},
							"connection_strings": {
								MarkdownDescription: "Connection strings.",
								Computed:            true,
								Attributes: tfsdk.SingleNestedAttributes(map[string]tfsdk.Attribute{
									"default_user": {
										MarkdownDescription: "The default TiDB user for connection.",
										Computed:            true,
										Type:                types.StringType,
									},
									"standard": {
										MarkdownDescription: "Standard connection string.",
										Computed:            true,
										Attributes: tfsdk.SingleNestedAttributes(map[string]tfsdk.Attribute{
											"host": {
												MarkdownDescription: "The host of standard connection.",
												Computed:            true,
												Type:                types.StringType,
											},
											"port": {
												MarkdownDescription: "The TiDB port for connection. The port must be in the range of 1024-65535 except 10080.",
												Computed:            true,
												Type:                types.Int64Type,
											},
										}),
									},
									"vpc_peering": {
										MarkdownDescription: "VPC peering connection string.",
										Computed:            true,
										Attributes: tfsdk.SingleNestedAttributes(map[string]tfsdk.Attribute{
											"host": {
												MarkdownDescription: "The host of VPC peering connection.",
												Computed:            true,
												Type:                types.StringType,
											},
											"port": {
												MarkdownDescription: "The TiDB port for connection. The port must be in the range of 1024-65535 except 10080.",
												Computed:            true,
												Type:                types.Int64Type,
											},
										}),
									},
								}),
							},
						}),
					},
				}),
			},
			"total": {
				MarkdownDescription: "The total number of project clusters.",
				Computed:            true,
				Type:                types.Int64Type,
			},
		},
	}, nil
}

func (t clustersDataSourceType) NewDataSource(ctx context.Context, in provider.Provider) (datasource.DataSource, diag.Diagnostics) {
	provider, diags := convertProviderType(in)

	return clustersDataSource{
		provider: provider,
	}, diags
}

type clustersDataSource struct {
	provider tidbcloudProvider
}

func (d clustersDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data clustersDataSourceData
	diags := req.Config.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// set default value
	if data.Page.IsNull() || data.Page.IsUnknown() {
		data.Page = types.Int64{Value: 1}
	}
	if data.PageSize.IsNull() || data.PageSize.IsUnknown() {
		data.PageSize = types.Int64{Value: 10}
	}

	tflog.Trace(ctx, "read clusters data source")
	clusters, err := d.provider.client.GetClusters(data.ProjectId.Value, data.Page.Value, data.PageSize.Value)
	if err != nil {
		resp.Diagnostics.AddError("Read Error", fmt.Sprintf("Unable to call read cluster, got error: %s", err))
		return
	}

	data.Total = types.Int64{Value: clusters.Total}
	var items []clusterItems
	for _, key := range clusters.Items {
		clusterItem := clusterItems{
			Id:              strconv.FormatUint(key.Id, 10),
			ProjectId:       strconv.FormatUint(key.ProjectId, 10),
			Name:            key.Name,
			CloudProvider:   key.CloudProvider,
			ClusterType:     key.ClusterType,
			Region:          key.Region,
			CreateTimestamp: key.CreateTimestamp,
			Config: &clusterConfigDataSource{
				Port: int64(key.Config.Port),
				Components: &components{
					TiDB: &componentTiDB{
						NodeSize:     key.Config.Components.TiDB.NodeSize,
						NodeQuantity: key.Config.Components.TiDB.NodeQuantity,
					},
					TiKV: &componentTiKV{
						NodeSize:       key.Config.Components.TiKV.NodeSize,
						NodeQuantity:   key.Config.Components.TiKV.NodeQuantity,
						StorageSizeGib: key.Config.Components.TiKV.StorageSizeGib,
					},
				},
			},
			Status: &clusterStatusDataSource{
				TidbVersion:   key.Status.TidbVersion,
				ClusterStatus: key.Status.ClusterStatus,
				ConnectionStrings: &connection{
					DefaultUser: key.Status.ConnectionStrings.DefaultUser,
				},
			},
		}
		if key.Status.ConnectionStrings.Standard.Port != 0 {
			clusterItem.Status.ConnectionStrings.Standard = &connectionStandard{
				Host: key.Status.ConnectionStrings.Standard.Host,
				Port: int64(key.Status.ConnectionStrings.Standard.Port),
			}
		}
		if key.Status.ConnectionStrings.VpcPeering.Port != 0 {
			clusterItem.Status.ConnectionStrings.VpcPeering = &connectionVpcPeering{
				Host: key.Status.ConnectionStrings.VpcPeering.Host,
				Port: int64(key.Status.ConnectionStrings.VpcPeering.Port),
			}
		}
		if key.Config.Components.TiFlash != nil {
			clusterItem.Config.Components.TiFlash = &componentTiFlash{
				NodeSize:       key.Config.Components.TiFlash.NodeSize,
				NodeQuantity:   key.Config.Components.TiFlash.NodeQuantity,
				StorageSizeGib: key.Config.Components.TiFlash.StorageSizeGib,
			}
		}
		items = append(items, clusterItem)
	}
	data.Clusters = items

	data.Id = types.String{Value: strconv.FormatInt(rand.Int63(), 10)}

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}
