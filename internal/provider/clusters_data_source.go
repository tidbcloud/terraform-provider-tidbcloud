package provider

import (
	"context"
	"fmt"
	"math/rand"
	"strconv"

	clusterApi "github.com/c4pt0r/go-tidbcloud-sdk-v1/client/cluster"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
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
	Port       int32       `tfsdk:"port"`
	Components *components `tfsdk:"components"`
}

type clusterStatusDataSource struct {
	TidbVersion       string       `tfsdk:"tidb_version"`
	ClusterStatus     types.String `tfsdk:"cluster_status"`
	ConnectionStrings *connection  `tfsdk:"connection_strings"`
}

type connection struct {
	DefaultUser string                `tfsdk:"default_user"`
	Standard    *connectionStandard   `tfsdk:"standard"`
	VpcPeering  *connectionVpcPeering `tfsdk:"vpc_peering"`
}

type connectionStandard struct {
	Host string `tfsdk:"host"`
	Port int32  `tfsdk:"port"`
}

type connectionVpcPeering struct {
	Host string `tfsdk:"host"`
	Port int32  `tfsdk:"port"`
}

// Ensure provider defined types fully satisfy framework interfaces
var _ datasource.DataSource = &clustersDataSource{}

type clustersDataSource struct {
	provider *tidbcloudProvider
}

func NewClustersDataSource() datasource.DataSource {
	return &clustersDataSource{}
}

func (d *clustersDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_clusters"
}

func (d *clustersDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	var ok bool
	if d.provider, ok = req.ProviderData.(*tidbcloudProvider); !ok {
		resp.Diagnostics.AddError("Internal provider error",
			fmt.Sprintf("Error in Configure: expected %T but got %T", tidbcloudProvider{}, req.ProviderData))
	}
}

func (d *clustersDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "clusters data source",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "data source ID.",
				Computed:            true,
			},
			"page": schema.Int64Attribute{
				MarkdownDescription: "Default:1 The number of pages.",
				Optional:            true,
				Computed:            true,
			},
			"page_size": schema.Int64Attribute{
				MarkdownDescription: "Default:10 The size of a pages.",
				Optional:            true,
				Computed:            true,
			},
			"project_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the project",
				Required:            true,
			},
			"items": schema.ListNestedAttribute{
				MarkdownDescription: "The items of clusters in the project.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							MarkdownDescription: "The ID of the cluster.",
							Computed:            true,
						},
						"project_id": schema.StringAttribute{
							MarkdownDescription: "The ID of the project.",
							Computed:            true,
						},
						"name": schema.StringAttribute{
							MarkdownDescription: "The name of the cluster.",
							Computed:            true,
						},
						"cluster_type": schema.StringAttribute{
							MarkdownDescription: "The cluster type.",
							Computed:            true,
						},
						"cloud_provider": schema.StringAttribute{
							MarkdownDescription: "Enum: \"AWS\" \"GCP\", The cloud provider on which your TiDB cluster is hosted.",
							Computed:            true,
						},
						"region": schema.StringAttribute{
							MarkdownDescription: "Region of the cluster.",
							Computed:            true,
						},
						"create_timestamp": schema.StringAttribute{
							MarkdownDescription: "The creation time of the cluster in Unix timestamp seconds (epoch time).",
							Computed:            true,
						},
						"config": schema.SingleNestedAttribute{
							MarkdownDescription: "The configuration of the cluster.",
							Computed:            true,
							Attributes: map[string]schema.Attribute{
								"port": schema.Int64Attribute{
									MarkdownDescription: "The TiDB port for connection. The port must be in the range of 1024-65535 except 10080, 4000 in default.\n" +
										"  - For a Serverless Tier cluster, only port 4000 is available.",
									Computed: true,
								},
								"components": schema.SingleNestedAttribute{
									MarkdownDescription: "The components of the cluster.",
									Computed:            true,
									Attributes: map[string]schema.Attribute{
										"tidb": schema.SingleNestedAttribute{
											MarkdownDescription: "The TiDB component of the cluster",
											Computed:            true,
											Attributes: map[string]schema.Attribute{
												"node_size": schema.StringAttribute{
													Computed: true,
													MarkdownDescription: "The size of the TiDB component in the cluster, You can get the available node size of each region from the [tidbcloud_cluster_specs datasource](../data-sources/cluster_specs.md).\n" +
														"  - If the vCPUs of TiDB or TiKV component is 2 or 4, then their vCPUs need to be the same.\n" +
														"  - If the vCPUs of TiDB or TiKV component is 2 or 4, then the cluster does not support TiFlash.\n" +
														"  - Can not modify node_size of an existing cluster.",
												},
												"node_quantity": schema.Int64Attribute{
													MarkdownDescription: "The number of nodes in the cluster. You can get the minimum and step of a node quantity from the [tidbcloud_cluster_specs datasource](../data-sources/cluster_specs.md).",
													Computed:            true,
												},
											},
										},
										"tikv": schema.SingleNestedAttribute{
											MarkdownDescription: "The TiKV component of the cluster",
											Computed:            true,
											Attributes: map[string]schema.Attribute{
												"node_size": schema.StringAttribute{
													MarkdownDescription: "The size of the TiKV component in the cluster, You can get the available node size of each region from the [tidbcloud_cluster_specs datasource](../data-sources/cluster_specs.md).\n" +
														"  - If the vCPUs of TiDB or TiKV component is 2 or 4, then their vCPUs need to be the same.\n" +
														"  - If the vCPUs of TiDB or TiKV component is 2 or 4, then the cluster does not support TiFlash.\n" +
														"  - Can not modify node_size of an existing cluster.",
													Computed: true,
												},
												"storage_size_gib": schema.Int64Attribute{
													MarkdownDescription: "The storage size of a node in the cluster. You can get the minimum and maximum of storage size from the [tidbcloud_cluster_specs datasource](../data-sources/cluster_specs.md).\n" +
														"  - Can not modify storage_size_gib of an existing cluster.",
													Computed: true,
												},
												"node_quantity": schema.Int64Attribute{
													MarkdownDescription: "The number of nodes in the cluster. You can get the minimum and step of a node quantity from the [tidbcloud_cluster_specs datasource](../data-sources/cluster_specs.md).\n" +
														"  - TiKV do not support decreasing node quantity.\n" +
														"  - The node_quantity of TiKV must be a multiple of 3.",
													Computed: true,
												},
											},
										},
										"tiflash": schema.SingleNestedAttribute{
											MarkdownDescription: "The TiFlash component of the cluster.",
											Computed:            true,
											Optional:            true,
											Attributes: map[string]schema.Attribute{
												"node_size": schema.StringAttribute{
													MarkdownDescription: "The size of the TiFlash component in the cluster, You can get the available node size of each region from the [tidbcloud_cluster_specs datasource](../data-sources/cluster_specs.md).\n" +
														"  - Can not modify node_size of an existing cluster.",
													Computed: true,
												},
												"storage_size_gib": schema.Int64Attribute{
													MarkdownDescription: "The storage size of a node in the cluster. You can get the minimum and maximum of storage size from the [tidbcloud_cluster_specs datasource](../data-sources/cluster_specs.md).\n" +
														"  - Can not modify storage_size_gib of an existing cluster.",
													Computed: true,
												},
												"node_quantity": schema.Int64Attribute{
													MarkdownDescription: "The number of nodes in the cluster. You can get the minimum and step of a node quantity from the [tidbcloud_cluster_specs datasource](../data-sources/cluster_specs.md).\n" +
														"  - TiFlash do not support decreasing node quantity.",
													Computed: true,
												},
											},
										},
									},
								},
							},
						},
						"status": schema.SingleNestedAttribute{
							MarkdownDescription: "The status of the cluster.",
							Computed:            true,
							Attributes: map[string]schema.Attribute{
								"tidb_version": schema.StringAttribute{
									MarkdownDescription: "TiDB version.",
									Computed:            true,
								},
								"cluster_status": schema.StringAttribute{
									MarkdownDescription: "Status of the cluster.",
									Computed:            true,
								},
								"connection_strings": schema.SingleNestedAttribute{
									MarkdownDescription: "Connection strings.",
									Computed:            true,
									Attributes: map[string]schema.Attribute{
										"default_user": schema.StringAttribute{
											MarkdownDescription: "The default TiDB user for connection.",
											Computed:            true,
										},
										"standard": schema.SingleNestedAttribute{
											MarkdownDescription: "Standard connection string.",
											Computed:            true,
											Attributes: map[string]schema.Attribute{
												"host": schema.StringAttribute{
													MarkdownDescription: "The host of standard connection.",
													Computed:            true,
												},
												"port": schema.Int64Attribute{
													MarkdownDescription: "The TiDB port for connection. The port must be in the range of 1024-65535 except 10080.",
													Computed:            true,
												},
											},
										},
										"vpc_peering": schema.SingleNestedAttribute{
											MarkdownDescription: "VPC peering connection string.",
											Computed:            true,
											Attributes: map[string]schema.Attribute{
												"host": schema.StringAttribute{
													MarkdownDescription: "The host of VPC peering connection.",
													Computed:            true,
												},
												"port": schema.Int64Attribute{
													MarkdownDescription: "The TiDB port for connection. The port must be in the range of 1024-65535 except 10080.",
													Computed:            true,
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			"total": schema.Int64Attribute{
				MarkdownDescription: "The total number of project clusters.",
				Computed:            true,
			},
		},
	}
}

func (d *clustersDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data clustersDataSourceData
	diags := req.Config.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// set default value
	var page int64 = 1
	var pageSize int64 = 10
	if !data.Page.IsNull() && !data.Page.IsUnknown() {
		page = data.Page.ValueInt64()
	}
	if !data.PageSize.IsNull() && !data.PageSize.IsUnknown() {
		pageSize = data.PageSize.ValueInt64()
	}

	tflog.Trace(ctx, "read clusters data source")
	listClustersOfProjectParams := clusterApi.NewListClustersOfProjectParams().WithProjectID(data.ProjectId.ValueString()).WithPage(&page).WithPageSize(&pageSize)
	listClustersOfProjectResp, err := d.provider.client.ListClustersOfProject(listClustersOfProjectParams)
	if err != nil {
		resp.Diagnostics.AddError("Read Error", fmt.Sprintf("Unable to call read cluster, got error: %s", err))
		return
	}

	data.Total = types.Int64Value(*listClustersOfProjectResp.Payload.Total)
	var items []clusterItems
	for _, key := range listClustersOfProjectResp.Payload.Items {
		clusterItem := clusterItems{
			Id:              *key.ID,
			ProjectId:       *key.ProjectID,
			Name:            key.Name,
			CloudProvider:   key.CloudProvider,
			ClusterType:     key.ClusterType,
			Region:          key.Region,
			CreateTimestamp: key.CreateTimestamp,
			Config: &clusterConfigDataSource{
				Port: key.Config.Port,
				Components: &components{
					TiDB: &componentTiDB{
						NodeSize:     *key.Config.Components.Tidb.NodeSize,
						NodeQuantity: *key.Config.Components.Tidb.NodeQuantity,
					},
					TiKV: &componentTiKV{
						NodeSize:       *key.Config.Components.Tikv.NodeSize,
						NodeQuantity:   *key.Config.Components.Tikv.NodeQuantity,
						StorageSizeGib: *key.Config.Components.Tikv.StorageSizeGib,
					},
				},
			},
			Status: &clusterStatusDataSource{
				TidbVersion:   key.Status.TidbVersion,
				ClusterStatus: types.StringValue(key.Status.ClusterStatus),
				ConnectionStrings: &connection{
					DefaultUser: key.Status.ConnectionStrings.DefaultUser,
				},
			},
		}
		var standard connectionStandard
		var vpcPeering connectionVpcPeering
		if key.Status.ConnectionStrings.Standard != nil {
			standard.Host = key.Status.ConnectionStrings.Standard.Host
			standard.Port = key.Status.ConnectionStrings.Standard.Port
		}
		if key.Status.ConnectionStrings.VpcPeering != nil {
			vpcPeering.Host = key.Status.ConnectionStrings.VpcPeering.Host
			vpcPeering.Port = key.Status.ConnectionStrings.VpcPeering.Port
		}
		clusterItem.Status.ConnectionStrings.Standard = &standard
		clusterItem.Status.ConnectionStrings.VpcPeering = &vpcPeering

		if key.Config.Components.Tiflash != nil {
			clusterItem.Config.Components.TiFlash = &componentTiFlash{
				NodeSize:       *key.Config.Components.Tiflash.NodeSize,
				NodeQuantity:   *key.Config.Components.Tiflash.NodeQuantity,
				StorageSizeGib: *key.Config.Components.Tiflash.StorageSizeGib,
			}
		}
		items = append(items, clusterItem)
	}
	data.Clusters = items

	data.Id = types.StringValue(strconv.FormatInt(rand.Int63(), 10))

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}
