package provider

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	clusterApi "github.com/c4pt0r/go-tidbcloud-sdk-v1/client/cluster"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"github.com/tidbcloud/terraform-provider-tidbcloud/tidbcloud"
)

const dev = "DEVELOPER"
const ded = "DEDICATED"

// Enum: [AVAILABLE CREATING MODIFYING PAUSED RESUMING UNAVAILABLE IMPORTING MAINTAINING PAUSING]
type clusterStatus string

const (
	clusterStatusCreating    clusterStatus = "CREATING"
	clusterStatusAvailable   clusterStatus = "AVAILABLE"
	clusterStatusModifying   clusterStatus = "MODIFYING"
	clusterStatusPaused      clusterStatus = "PAUSED"
	clusterStatusResuming    clusterStatus = "RESUMING"
	clusterStatusUnavailable clusterStatus = "UNAVAILABLE"
	clusterStatusImporting   clusterStatus = "IMPORTING"
	clusterStatusMaintaining clusterStatus = "MAINTAINING"
	clusterStatusPausing     clusterStatus = "PAUSING"
)

const (
	clusterServerlessCreateTimeout  = 180 * time.Second
	clusterServerlessCreateInterval = 2 * time.Second
	clusterCreateTimeout            = time.Hour
	clusterCreateInterval           = 60 * time.Second
	clusterUpdateTimeout            = time.Hour
	clusterUpdateInterval           = 20 * time.Second
)

type clusterResourceData struct {
	ClusterId       types.String             `tfsdk:"id"`
	ProjectId       string                   `tfsdk:"project_id"`
	Name            string                   `tfsdk:"name"`
	ClusterType     string                   `tfsdk:"cluster_type"`
	CloudProvider   string                   `tfsdk:"cloud_provider"`
	Region          string                   `tfsdk:"region"`
	CreateTimestamp types.String             `tfsdk:"create_timestamp"`
	Config          clusterConfig            `tfsdk:"config"`
	Status          *clusterStatusDataSource `tfsdk:"status"`
}

type clusterConfig struct {
	Paused       *bool        `tfsdk:"paused"`
	RootPassword types.String `tfsdk:"root_password"`
	Port         types.Int64  `tfsdk:"port"`
	Components   *components  `tfsdk:"components"`
	IPAccessList []ipAccess   `tfsdk:"ip_access_list"`
}

type components struct {
	TiDB    *componentTiDB    `tfsdk:"tidb"`
	TiKV    *componentTiKV    `tfsdk:"tikv"`
	TiFlash *componentTiFlash `tfsdk:"tiflash"`
}

type componentTiDB struct {
	NodeSize     string `tfsdk:"node_size"`
	NodeQuantity int32  `tfsdk:"node_quantity"`
}

type componentTiKV struct {
	NodeSize       string `tfsdk:"node_size"`
	StorageSizeGib int32  `tfsdk:"storage_size_gib"`
	NodeQuantity   int32  `tfsdk:"node_quantity"`
}

type componentTiFlash struct {
	NodeSize       string `tfsdk:"node_size"`
	StorageSizeGib int32  `tfsdk:"storage_size_gib"`
	NodeQuantity   int32  `tfsdk:"node_quantity"`
}

type ipAccess struct {
	CIDR        string `tfsdk:"cidr"`
	Description string `tfsdk:"description"`
}

type clusterResource struct {
	provider *tidbcloudProvider
}

func NewClusterResource() resource.Resource {
	return &clusterResource{}
}

func (r *clusterResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cluster"
}

func (r *clusterResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	var ok bool
	if r.provider, ok = req.ProviderData.(*tidbcloudProvider); !ok {
		resp.Diagnostics.AddError("Internal provider error",
			fmt.Sprintf("Error in Configure: expected %T but got %T", tidbcloudProvider{}, req.ProviderData))
	}
}

func (r *clusterResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "cluster resource",
		Attributes: map[string]schema.Attribute{
			"project_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the project. You can get the project ID from [tidbcloud_projects datasource](../data-sources/projects.md).",
				Required:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the cluster.",
				Required:            true,
			},
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The ID of the cluster.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"cluster_type": schema.StringAttribute{
				MarkdownDescription: "Enum: \"DEDICATED\" \"DEVELOPER\", The cluster type.",
				Required:            true,
			},
			"cloud_provider": schema.StringAttribute{
				MarkdownDescription: "Enum: \"AWS\" \"GCP\", The cloud provider on which your TiDB cluster is hosted.",
				Required:            true,
			},
			"create_timestamp": schema.StringAttribute{
				MarkdownDescription: "The creation time of the cluster in Unix timestamp seconds (epoch time).",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"region": schema.StringAttribute{
				MarkdownDescription: "the region value should match the cloud provider's region code. You can get the complete list of available regions from the [tidbcloud_cluster_specs datasource](../data-sources/cluster_specs.md).",
				Required:            true,
			},
			"status": schema.SingleNestedAttribute{
				MarkdownDescription: "The status of the cluster.",
				Computed:            true,
				PlanModifiers: []planmodifier.Object{
					clusterResourceStatus(),
				},
				Attributes: map[string]schema.Attribute{
					"tidb_version": schema.StringAttribute{
						MarkdownDescription: "TiDB version.",
						Computed:            true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"cluster_status": schema.StringAttribute{
						MarkdownDescription: "Status of the cluster.",
						Computed:            true,
					},
					"connection_strings": schema.SingleNestedAttribute{
						MarkdownDescription: "Connection strings.",
						Computed:            true,
						PlanModifiers: []planmodifier.Object{
							objectplanmodifier.UseStateForUnknown(),
						},
						Attributes: map[string]schema.Attribute{
							"default_user": schema.StringAttribute{
								MarkdownDescription: "The default TiDB user for connection.",
								Computed:            true,
								PlanModifiers: []planmodifier.String{
									stringplanmodifier.UseStateForUnknown(),
								},
							},
							"standard": schema.SingleNestedAttribute{
								MarkdownDescription: "Standard connection string.",
								Computed:            true,
								PlanModifiers: []planmodifier.Object{
									objectplanmodifier.UseStateForUnknown(),
								},
								Attributes: map[string]schema.Attribute{
									"host": schema.StringAttribute{
										MarkdownDescription: "The host of standard connection.",
										Computed:            true,
										PlanModifiers: []planmodifier.String{
											stringplanmodifier.UseStateForUnknown(),
										},
									},
									"port": schema.Int64Attribute{
										MarkdownDescription: "The TiDB port for connection. The port must be in the range of 1024-65535 except 10080.",
										Computed:            true,
										PlanModifiers: []planmodifier.Int64{
											int64planmodifier.UseStateForUnknown(),
										},
									},
								},
							},
							"vpc_peering": schema.SingleNestedAttribute{
								MarkdownDescription: "VPC peering connection string.",
								Computed:            true,
								PlanModifiers: []planmodifier.Object{
									objectplanmodifier.UseStateForUnknown(),
								},
								Attributes: map[string]schema.Attribute{
									"host": schema.StringAttribute{
										MarkdownDescription: "The host of VPC peering connection.",
										Computed:            true,
										PlanModifiers: []planmodifier.String{
											stringplanmodifier.UseStateForUnknown(),
										},
									},
									"port": schema.Int64Attribute{
										MarkdownDescription: "The TiDB port for connection. The port must be in the range of 1024-65535 except 10080.",
										Computed:            true,
										PlanModifiers: []planmodifier.Int64{
											int64planmodifier.UseStateForUnknown(),
										},
									},
								},
							},
						},
					},
				},
			},
			"config": schema.SingleNestedAttribute{
				MarkdownDescription: "The configuration of the cluster.",
				Required:            true,
				Attributes: map[string]schema.Attribute{
					"root_password": schema.StringAttribute{
						MarkdownDescription: "The root password to access the cluster. It must be 8-64 characters.",
						Optional:            true,
					},
					"port": schema.Int64Attribute{
						MarkdownDescription: "The TiDB port for connection. The port must be in the range of 1024-65535 except 10080, 4000 in default.\n" +
							"  - For a Serverless Tier cluster, only port 4000 is available.",
						Optional: true,
						Computed: true,
						PlanModifiers: []planmodifier.Int64{
							int64planmodifier.UseStateForUnknown(),
						},
					},
					"paused": schema.BoolAttribute{
						MarkdownDescription: "lag that indicates whether the cluster is paused. true means to pause the cluster, and false means to resume the cluster.\n" +
							"  - The cluster can be paused only when the cluster_status is \"AVAILABLE\"." +
							"  - The cluster can be resumed only when the cluster_status is \"PAUSED\".",
						Optional: true,
					},
					"components": schema.SingleNestedAttribute{
						MarkdownDescription: "The components of the cluster.\n" +
							"  - For a Serverless Tier cluster, the components value can not be set." +
							"  - For a Dedicated Tier cluster, the components value must be set.",
						Optional: true,
						Computed: true,
						PlanModifiers: []planmodifier.Object{
							objectplanmodifier.UseStateForUnknown(),
						},
						Attributes: map[string]schema.Attribute{
							"tidb": schema.SingleNestedAttribute{
								MarkdownDescription: "The TiDB component of the cluster",
								Required:            true,
								PlanModifiers: []planmodifier.Object{
									objectplanmodifier.UseStateForUnknown(),
								},
								Attributes: map[string]schema.Attribute{
									"node_size": schema.StringAttribute{
										Required: true,
										MarkdownDescription: "The size of the TiDB component in the cluster, You can get the available node size of each region from the [tidbcloud_cluster_specs datasource](../data-sources/cluster_specs.md).\n" +
											"  - If the vCPUs of TiDB or TiKV component is 2 or 4, then their vCPUs need to be the same.\n" +
											"  - If the vCPUs of TiDB or TiKV component is 2 or 4, then the cluster does not support TiFlash.\n" +
											"  - Can not modify node_size of an existing cluster.",
										PlanModifiers: []planmodifier.String{
											stringplanmodifier.UseStateForUnknown(),
										},
									},
									"node_quantity": schema.Int64Attribute{
										MarkdownDescription: "The number of nodes in the cluster. You can get the minimum and step of a node quantity from the [tidbcloud_cluster_specs datasource](../data-sources/cluster_specs.md).",
										Required:            true,
										PlanModifiers: []planmodifier.Int64{
											int64planmodifier.UseStateForUnknown(),
										},
									},
								},
							},
							"tikv": schema.SingleNestedAttribute{
								MarkdownDescription: "The TiKV component of the cluster",
								Required:            true,
								PlanModifiers: []planmodifier.Object{
									objectplanmodifier.UseStateForUnknown(),
								},
								Attributes: map[string]schema.Attribute{
									"node_size": schema.StringAttribute{
										MarkdownDescription: "The size of the TiKV component in the cluster, You can get the available node size of each region from the [tidbcloud_cluster_specs datasource](../data-sources/cluster_specs.md).\n" +
											"  - If the vCPUs of TiDB or TiKV component is 2 or 4, then their vCPUs need to be the same.\n" +
											"  - If the vCPUs of TiDB or TiKV component is 2 or 4, then the cluster does not support TiFlash.\n" +
											"  - Can not modify node_size of an existing cluster.",
										Required: true,
										PlanModifiers: []planmodifier.String{
											stringplanmodifier.UseStateForUnknown(),
										},
									},
									"storage_size_gib": schema.Int64Attribute{
										MarkdownDescription: "The storage size of a node in the cluster. You can get the minimum and maximum of storage size from the [tidbcloud_cluster_specs datasource](../data-sources/cluster_specs.md).\n" +
											"  - Can not modify storage_size_gib of an existing cluster.",
										Required: true,
										PlanModifiers: []planmodifier.Int64{
											int64planmodifier.UseStateForUnknown(),
										},
									},
									"node_quantity": schema.Int64Attribute{
										MarkdownDescription: "The number of nodes in the cluster. You can get the minimum and step of a node quantity from the [tidbcloud_cluster_specs datasource](../data-sources/cluster_specs.md).\n" +
											"  - TiKV do not support decreasing node quantity.\n" +
											"  - The node_quantity of TiKV must be a multiple of 3.",
										Required: true,
										PlanModifiers: []planmodifier.Int64{
											int64planmodifier.UseStateForUnknown(),
										},
									},
								},
							},
							"tiflash": schema.SingleNestedAttribute{
								MarkdownDescription: "The TiFlash component of the cluster.",
								Optional:            true,
								PlanModifiers: []planmodifier.Object{
									objectplanmodifier.UseStateForUnknown(),
								},
								Attributes: map[string]schema.Attribute{
									"node_size": schema.StringAttribute{
										MarkdownDescription: "The size of the TiFlash component in the cluster, You can get the available node size of each region from the [tidbcloud_cluster_specs datasource](../data-sources/cluster_specs.md).\n" +
											"  - Can not modify node_size of an existing cluster.",
										Required: true,
										PlanModifiers: []planmodifier.String{
											stringplanmodifier.UseStateForUnknown(),
										},
									},
									"storage_size_gib": schema.Int64Attribute{
										MarkdownDescription: "The storage size of a node in the cluster. You can get the minimum and maximum of storage size from the [tidbcloud_cluster_specs datasource](../data-sources/cluster_specs.md).\n" +
											"  - Can not modify storage_size_gib of an existing cluster.",
										Required: true,
										PlanModifiers: []planmodifier.Int64{
											int64planmodifier.UseStateForUnknown(),
										},
									},
									"node_quantity": schema.Int64Attribute{
										MarkdownDescription: "The number of nodes in the cluster. You can get the minimum and step of a node quantity from the [tidbcloud_cluster_specs datasource](../data-sources/cluster_specs.md).\n" +
											"  - TiFlash do not support decreasing node quantity.",
										Required: true,
										PlanModifiers: []planmodifier.Int64{
											int64planmodifier.UseStateForUnknown(),
										},
									},
								},
							},
						},
					},
					"ip_access_list": schema.ListNestedAttribute{
						MarkdownDescription: "A list of IP addresses and Classless Inter-Domain Routing (CIDR) addresses that are allowed to access the TiDB Cloud cluster via [standard connection](https://docs.pingcap.com/tidbcloud/connect-to-tidb-cluster#connect-via-standard-connection).",
						Optional:            true,
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"cidr": schema.StringAttribute{
									MarkdownDescription: "The IP address or CIDR range that you want to add to the cluster's IP access list.",
									Required:            true,
								},
								"description": schema.StringAttribute{
									MarkdownDescription: "Description that explains the purpose of the entry.",
									Required:            true,
								},
							},
						},
					},
				},
			},
		},
	}
}

func (r clusterResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if !r.provider.configured {
		resp.Diagnostics.AddError(
			"Provider not configured",
			"The provider hasn't been configured before apply, likely because it depends on an unknown value from another resource. This leads to weird stuff happening, so we'd prefer if you didn't do that. Thanks!",
		)
		return
	}

	// get data from config
	var data clusterResourceData
	diags := req.Config.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// for Serverless cluster, components is not allowed. or plan and state may be inconsistent
	if data.ClusterType == dev {
		if data.Config.Components != nil {
			resp.Diagnostics.AddError("Create Error", fmt.Sprintf("components is not allowed in %s cluster_type", dev))
			return
		}
	}

	// for DEDICATED cluster, components is required.
	if data.ClusterType == ded {
		if data.Config.Components == nil {
			resp.Diagnostics.AddError("Create Error", fmt.Sprintf("components is required in %s cluster_type", ded))
			return
		}
	}

	// write logs using the tflog package
	// see https://pkg.go.dev/github.com/hashicorp/terraform-plugin-log/tflog
	tflog.Trace(ctx, "created cluster_resource")
	createClusterParams := clusterApi.NewCreateClusterParams().WithProjectID(data.ProjectId).WithBody(buildCreateClusterBody(data))
	createClusterResp, err := r.provider.client.CreateCluster(createClusterParams)
	if err != nil {
		resp.Diagnostics.AddError("Create Error", fmt.Sprintf("Unable to call CreateCluster, got error: %s", err))
		return
	}
	// set clusterId. other computed attributes are not returned by create, they will be set when refresh
	clusterId := *createClusterResp.Payload.ID
	data.ClusterId = types.StringValue(clusterId)
	if r.provider.sync {
		var cluster *clusterApi.GetClusterOKBody
		if data.ClusterType == dev {
			tflog.Info(ctx, "wait serverless cluster ready")
			cluster, err = WaitClusterReady(ctx, clusterServerlessCreateTimeout, clusterServerlessCreateInterval, data.ProjectId, clusterId, r.provider.client)
			if err != nil {
				resp.Diagnostics.AddError(
					"Cluster creation failed",
					fmt.Sprintf("Cluster is not ready, get error: %s", err),
				)
				return
			}
		} else {
			tflog.Info(ctx, "wait dedicated cluster ready")
			cluster, err = WaitClusterReady(ctx, clusterCreateTimeout, clusterCreateInterval, data.ProjectId, clusterId, r.provider.client)
			if err != nil {
				resp.Diagnostics.AddError(
					"Cluster creation failed",
					fmt.Sprintf("Cluster is not ready, get error: %s", err),
				)
				return
			}
		}
		refreshClusterResourceData(ctx, cluster, &data)
	} else {
		// we refresh in create for any unknown value. if someone has other opinions which is better, he can delete the refresh logic
		tflog.Trace(ctx, "read cluster_resource")
		getClusterParams := clusterApi.NewGetClusterParams().WithProjectID(data.ProjectId).WithClusterID(data.ClusterId.ValueString())
		getClusterResp, err := r.provider.client.GetCluster(getClusterParams)
		if err != nil {
			resp.Diagnostics.AddError("Create Error", fmt.Sprintf("Unable to call GetCluster, got error: %s", err))
			return
		}
		refreshClusterResourceData(ctx, getClusterResp.Payload, &data)
	}

	// save into the Terraform state.
	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func buildCreateClusterBody(data clusterResourceData) clusterApi.CreateClusterBody {
	// required
	rootPassWord := data.Config.RootPassword.ValueString()
	payload := clusterApi.CreateClusterBody{
		Name:          &data.Name,
		ClusterType:   &data.ClusterType,
		CloudProvider: &data.CloudProvider,
		Region:        &data.Region,
		Config: &clusterApi.CreateClusterParamsBodyConfig{
			RootPassword: &rootPassWord,
		},
	}

	// optional
	if data.Config.Components != nil {
		tidb := data.Config.Components.TiDB
		tikv := data.Config.Components.TiKV
		tiflash := data.Config.Components.TiFlash

		components := &clusterApi.CreateClusterParamsBodyConfigComponents{
			Tidb: &clusterApi.CreateClusterParamsBodyConfigComponentsTidb{
				NodeSize:     &tidb.NodeSize,
				NodeQuantity: &tidb.NodeQuantity,
			},
			Tikv: &clusterApi.CreateClusterParamsBodyConfigComponentsTikv{
				NodeSize:       &tikv.NodeSize,
				StorageSizeGib: &tikv.StorageSizeGib,
				NodeQuantity:   &tikv.NodeQuantity,
			},
		}
		// tiflash is optional
		if tiflash != nil {
			components.Tiflash = &clusterApi.CreateClusterParamsBodyConfigComponentsTiflash{
				NodeSize:       &tiflash.NodeSize,
				StorageSizeGib: &tiflash.StorageSizeGib,
				NodeQuantity:   &tiflash.NodeQuantity,
			}
		}

		payload.Config.Components = components
	}
	if data.Config.IPAccessList != nil {
		var IPAccessList []*clusterApi.CreateClusterParamsBodyConfigIPAccessListItems0
		for _, key := range data.Config.IPAccessList {
			cidr := key.CIDR
			IPAccessList = append(IPAccessList, &clusterApi.CreateClusterParamsBodyConfigIPAccessListItems0{
				Cidr:        &cidr,
				Description: key.Description,
			})
		}
		payload.Config.IPAccessList = IPAccessList
	}
	if !data.Config.Port.IsNull() && !data.Config.Port.IsUnknown() {
		payload.Config.Port = int32(data.Config.Port.ValueInt64())
	}

	return payload
}

func (r clusterResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var projectId, clusterId string

	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("project_id"), &projectId)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("id"), &clusterId)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// call read api
	tflog.Trace(ctx, "read cluster_resource")
	getClusterParams := clusterApi.NewGetClusterParams().WithProjectID(projectId).WithClusterID(clusterId)
	getClusterResp, err := r.provider.client.GetCluster(getClusterParams)
	if err != nil {
		resp.Diagnostics.AddError("Read Error", fmt.Sprintf("Unable to call GetClusterById, got error: %s", err))
		return
	}

	// refresh data with read result
	var data clusterResourceData
	// root_password, ip_access_list and pause will not return by read api, so we just use state's value even it changed on console!
	// use types.String in case ImportState method throw unhandled null value
	var rootPassword types.String
	var iPAccessList []ipAccess
	var paused *bool
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("config").AtName("root_password"), &rootPassword)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("config").AtName("ip_access_list"), &iPAccessList)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("config").AtName("paused"), &paused)...)
	data.Config.RootPassword = rootPassword
	data.Config.IPAccessList = iPAccessList
	data.Config.Paused = paused
	refreshClusterResourceData(ctx, getClusterResp.Payload, &data)

	// save into the Terraform state
	diags := resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func refreshClusterResourceData(ctx context.Context, resp *clusterApi.GetClusterOKBody, data *clusterResourceData) {
	// must return
	data.Name = resp.Name
	data.ClusterId = types.StringValue(*resp.ID)
	data.Region = resp.Region
	data.ProjectId = *resp.ProjectID
	data.ClusterType = resp.ClusterType
	data.CloudProvider = resp.CloudProvider
	data.CreateTimestamp = types.StringValue(resp.CreateTimestamp)
	data.Config.Port = types.Int64Value(int64(resp.Config.Port))
	tidb := resp.Config.Components.Tidb
	tikv := resp.Config.Components.Tikv
	data.Config.Components = &components{
		TiDB: &componentTiDB{
			NodeSize:     *tidb.NodeSize,
			NodeQuantity: *tidb.NodeQuantity,
		},
		TiKV: &componentTiKV{
			NodeSize:       *tikv.NodeSize,
			NodeQuantity:   *tikv.NodeQuantity,
			StorageSizeGib: *tikv.StorageSizeGib,
		},
	}

	var standard connectionStandard
	var vpcPeering connectionVpcPeering
	if resp.Status.ConnectionStrings.Standard != nil {
		standard.Host = resp.Status.ConnectionStrings.Standard.Host
		standard.Port = resp.Status.ConnectionStrings.Standard.Port
	}
	if resp.Status.ConnectionStrings.VpcPeering != nil {
		vpcPeering.Host = resp.Status.ConnectionStrings.VpcPeering.Host
		vpcPeering.Port = resp.Status.ConnectionStrings.VpcPeering.Port
	}
	data.Status = &clusterStatusDataSource{
		TidbVersion:   resp.Status.TidbVersion,
		ClusterStatus: types.StringValue(resp.Status.ClusterStatus),
		ConnectionStrings: &connection{
			DefaultUser: resp.Status.ConnectionStrings.DefaultUser,
			Standard:    &standard,
			VpcPeering:  &vpcPeering,
		},
	}
	// may return
	tiflash := resp.Config.Components.Tiflash
	if tiflash != nil {
		data.Config.Components.TiFlash = &componentTiFlash{
			NodeSize:       *tiflash.NodeSize,
			NodeQuantity:   *tiflash.NodeQuantity,
			StorageSizeGib: *tiflash.StorageSizeGib,
		}
	}

	// not return
	// IPAccessList, and password and pause will not update for it will not return by read api

}

// Update since open api is patch without check for the invalid parameter. we do a lot of check here to avoid inconsistency
// check the date can't be updated
// if plan and state is different, we can execute updated
func (r clusterResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// get plan
	var data clusterResourceData
	diags := req.Plan.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	// get state
	var state clusterResourceData
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Severless can not be changed now
	if data.ClusterType == dev {
		resp.Diagnostics.AddError(
			"Update error",
			"Unable to update Serverless cluster",
		)
		return
	}

	// only components and paused can be changed now
	if data.Name != state.Name || data.ClusterType != state.ClusterType || data.Region != state.Region || data.CloudProvider != state.CloudProvider ||
		data.ProjectId != state.ProjectId || data.ClusterId != state.ClusterId {
		resp.Diagnostics.AddError(
			"Update error",
			"You may update the name,cluster_type,region,cloud_provider or projectId. They can not be changed, only components can be changed now",
		)
		return
	}
	if !data.Config.Port.IsNull() && !data.Config.Port.IsNull() && data.Config.Port.ValueInt64() != state.Config.Port.ValueInt64() {
		resp.Diagnostics.AddError(
			"Update error",
			"port can not be changed, only components can be changed now",
		)
		return
	}
	if data.Config.IPAccessList != nil {
		// You cannot add an IP access list to an existing cluster without an IP rule.
		if len(state.Config.IPAccessList) == 0 {
			resp.Diagnostics.AddError(
				"Update error",
				"ip_access_list can not be added to the existing cluster.",
			)
			return
		}

		// You cannot insert or delete IP rule.
		if len(data.Config.IPAccessList) != len(state.Config.IPAccessList) {
			resp.Diagnostics.AddError(
				"Update error",
				"ip_access_list can not be changed, only components can be changed now",
			)
			return
		}

		// You cannot update the IP rule.
		newIPAccessList := make([]ipAccess, len(data.Config.IPAccessList))
		copy(newIPAccessList, data.Config.IPAccessList)
		sort.Slice(newIPAccessList, func(i, j int) bool {
			return newIPAccessList[i].CIDR < newIPAccessList[j].CIDR
		})

		currentIPAccessList := make([]ipAccess, len(state.Config.IPAccessList))
		copy(currentIPAccessList, state.Config.IPAccessList)
		sort.Slice(currentIPAccessList, func(i, j int) bool {
			return currentIPAccessList[i].CIDR < currentIPAccessList[j].CIDR
		})

		for index, key := range newIPAccessList {
			if currentIPAccessList[index].CIDR != key.CIDR || currentIPAccessList[index].Description != key.Description {
				resp.Diagnostics.AddError(
					"Update error",
					"ip_access_list can not be changed, only components can be changed now",
				)
				return
			}
		}
	} else {
		// You cannot remove the IP access list.
		if len(state.Config.IPAccessList) > 0 {
			resp.Diagnostics.AddError(
				"Update error",
				"ip_access_list can not be changed, only components can be changed now",
			)
			return
		}
	}

	// check Components
	tidb := data.Config.Components.TiDB
	tikv := data.Config.Components.TiKV
	tiflash := data.Config.Components.TiFlash
	tidbState := state.Config.Components.TiDB
	tikvState := state.Config.Components.TiKV
	tiflashState := state.Config.Components.TiFlash
	if tidb.NodeSize != tidbState.NodeSize {
		resp.Diagnostics.AddError(
			"Update error",
			"tidb node_size can't be changed",
		)
		return
	}
	if tikv.NodeSize != tikvState.NodeSize || tikv.StorageSizeGib != tikvState.StorageSizeGib {
		resp.Diagnostics.AddError(
			"Update error",
			"tikv node_size or storage_size_gib can't be changed",
		)
		return
	}
	if tiflash != nil && tiflashState != nil {
		// if cluster have tiflash already, then we can't specify NodeSize and StorageSizeGib
		if tiflash.NodeSize != tiflashState.NodeSize || tiflash.StorageSizeGib != tiflashState.StorageSizeGib {
			resp.Diagnostics.AddError(
				"Update error",
				"tiflash node_size or storage_size_gib can't be changed",
			)
			return
		}
	}

	// build UpdateClusterBody
	var updateClusterBody clusterApi.UpdateClusterBody
	updateClusterBody.Config = &clusterApi.UpdateClusterParamsBodyConfig{}
	// build paused
	if data.Config.Paused != nil {
		if state.Config.Paused == nil || *data.Config.Paused != *state.Config.Paused {
			updateClusterBody.Config.Paused = data.Config.Paused
		}
	}
	// build components
	var isComponentsChanged = false
	if tidb.NodeQuantity != tidbState.NodeQuantity || tikv.NodeQuantity != tikvState.NodeQuantity {
		isComponentsChanged = true
	}

	var componentTiFlash *clusterApi.UpdateClusterParamsBodyConfigComponentsTiflash
	if tiflash != nil {
		if tiflashState == nil {
			isComponentsChanged = true
			componentTiFlash = &clusterApi.UpdateClusterParamsBodyConfigComponentsTiflash{
				NodeQuantity:   &tiflash.NodeQuantity,
				NodeSize:       &tiflash.NodeSize,
				StorageSizeGib: &tiflash.StorageSizeGib,
			}
		} else if tiflash.NodeQuantity != tiflashState.NodeQuantity {
			isComponentsChanged = true
			// NodeSize can't be changed
			componentTiFlash = &clusterApi.UpdateClusterParamsBodyConfigComponentsTiflash{
				NodeQuantity: &tiflash.NodeQuantity,
			}
		}
	}

	if isComponentsChanged {
		updateClusterBody.Config.Components = &clusterApi.UpdateClusterParamsBodyConfigComponents{
			Tidb: &clusterApi.UpdateClusterParamsBodyConfigComponentsTidb{
				NodeQuantity: &tidb.NodeQuantity,
			},
			Tikv: &clusterApi.UpdateClusterParamsBodyConfigComponentsTikv{
				NodeQuantity: &tikv.NodeQuantity,
			},
			Tiflash: componentTiFlash,
		}
	}

	tflog.Trace(ctx, "update cluster_resource")
	updateClusterParams := clusterApi.NewUpdateClusterParams().WithProjectID(data.ProjectId).WithClusterID(data.ClusterId.ValueString()).WithBody(updateClusterBody)
	_, err := r.provider.client.UpdateCluster(updateClusterParams)
	if err != nil {
		resp.Diagnostics.AddError("Update Error", fmt.Sprintf("Unable to call UpdateClusterById, got error: %s", err))
		return
	}

	if r.provider.sync {
		tflog.Info(ctx, "wait cluster ready")
		cluster, err := WaitClusterReady(ctx, clusterUpdateTimeout, clusterUpdateInterval, data.ProjectId, data.ClusterId.ValueString(), r.provider.client)
		if err != nil {
			resp.Diagnostics.AddError(
				"Cluster update failed",
				fmt.Sprintf("Cluster is not ready, get error: %s", err),
			)
			return
		}
		refreshClusterResourceData(ctx, cluster, &data)
	} else {
		// we refresh for any unknown value. if someone has other opinions which is better, he can delete the refresh logic
		tflog.Trace(ctx, "read cluster_resource")
		getClusterResp, err := r.provider.client.GetCluster(clusterApi.NewGetClusterParams().WithProjectID(data.ProjectId).WithClusterID(data.ClusterId.ValueString()))
		if err != nil {
			resp.Diagnostics.AddError("Update Error", fmt.Sprintf("Unable to call GetClusterById, got error: %s", err))
			return
		}
		refreshClusterResourceData(ctx, getClusterResp.Payload, &data)
	}

	// save into the Terraform state.
	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (r clusterResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data clusterResourceData

	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Trace(ctx, "delete cluster_resource")
	_, err := r.provider.client.DeleteCluster(clusterApi.NewDeleteClusterParams().WithProjectID(data.ProjectId).WithClusterID(data.ClusterId.ValueString()))
	if err != nil {
		resp.Diagnostics.AddError("Delete Error", fmt.Sprintf("Unable to call DeleteClusterById, got error: %s", err))
		return
	}
}

func (r clusterResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	idParts := strings.Split(req.ID, ",")

	if len(idParts) != 2 || idParts[0] == "" || idParts[1] == "" {
		resp.Diagnostics.AddError(
			"Unexpected Import Identifier",
			fmt.Sprintf("Expected import identifier with format: project_id,cluster_id. Got: %q", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_id"), idParts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), idParts[1])...)
}

func WaitClusterReady(ctx context.Context, timeout time.Duration, interval time.Duration, projectId, clusterId string,
	client tidbcloud.TiDBCloudClient) (*clusterApi.GetClusterOKBody, error) {
	stateConf := &retry.StateChangeConf{
		Pending: []string{
			string(clusterStatusCreating),
			string(clusterStatusModifying),
			string(clusterStatusResuming),
			string(clusterStatusUnavailable),
			string(clusterStatusImporting),
			string(clusterStatusPausing),
		},
		Target: []string{
			string(clusterStatusAvailable),
			string(clusterStatusPaused),
			string(clusterStatusMaintaining),
		},
		Timeout:      timeout,
		MinTimeout:   500 * time.Millisecond,
		PollInterval: interval,
		Refresh:      clusterStateRefreshFunc(ctx, projectId, clusterId, client),
	}

	outputRaw, err := stateConf.WaitForStateContext(ctx)

	if output, ok := outputRaw.(*clusterApi.GetClusterOKBody); ok {
		return output, err
	}
	return nil, err
}

func clusterStateRefreshFunc(ctx context.Context, projectId, clusterId string,
	client tidbcloud.TiDBCloudClient) retry.StateRefreshFunc {
	return func() (interface{}, string, error) {
		param := clusterApi.NewGetClusterParams().WithProjectID(projectId).WithClusterID(clusterId).WithContext(ctx)
		getClusterResp, err := client.GetCluster(param)
		if err != nil {
			tflog.Warn(ctx, fmt.Sprintf("get clusrer error: %s", err))
			if getClusterResp != nil && getClusterResp.Code() < http.StatusInternalServerError {
				// regard as not found and retry again. Default is 20 times
				return nil, "", nil
			} else {
				return nil, "", err
			}
		}
		return getClusterResp.Payload, getClusterResp.Payload.Status.ClusterStatus, nil
	}
}
