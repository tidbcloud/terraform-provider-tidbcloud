package provider

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"github.com/tidbcloud/terraform-provider-tidbcloud/tidbcloud"
	"github.com/tidbcloud/tidbcloud-cli/pkg/tidbcloud/v1beta1/dedicated"
)

// const dev = "DEVELOPER"
// const ded = "DEDICATED"

// // Enum: [AVAILABLE CREATING MODIFYING PAUSED RESUMING UNAVAILABLE IMPORTING MAINTAINING PAUSING]
// type clusterStatus string

const (
	dedicatedClusterStatusCreating    clusterStatus = "CREATING"
	dedicatedClusterStatusDeleting    clusterStatus = "DELETING"
	dedicatedClusterStatusActive      clusterStatus = "ACTIVE"
	dedicatedClusterStatusRestoring   clusterStatus = "RESTORING"
	dedicatedClusterStatusMaintenance clusterStatus = "MAINTENANCE"
	dedicatedClusterStatusDeleted     clusterStatus = "DELETED"
	dedicatedClusterStatusInactive    clusterStatus = "INACTIVE"
	dedicatedClusterStatusUPgrading   clusterStatus = "UPGRADING"
	dedicatedClusterStatusImporting   clusterStatus = "IMPORTING"
	dedicatedClusterStatusModifying   clusterStatus = "MODIFYING"
	dedicatedClusterStatusPausing     clusterStatus = "PAUSING"
	dedicatedClusterStatusPaused      clusterStatus = "PAUSED"
	dedicatedClusterStatusResuming    clusterStatus = "RESUMING"
)

// const (
// 	clusterServerlessCreateTimeout  = 180 * time.Second
// 	clusterServerlessCreateInterval = 2 * time.Second
// 	clusterCreateTimeout            = time.Hour
// 	clusterCreateInterval           = 60 * time.Second
// 	clusterUpdateTimeout            = time.Hour
// 	clusterUpdateInterval           = 20 * time.Second
// )

type dedicatedClusterResourceData struct {
	ProjectId          types.String        `tfsdk:"project_id"`
	ClusterId          types.String        `tfsdk:"id"`
	Name               types.String        `tfsdk:"name"`
	CloudProvider      types.String        `tfsdk:"cloud_provider"`
	RegionId           types.String        `tfsdk:"region_id"`
	Labels             map[string]string   `tfsdk:"labels"`
	RootPassword       types.String        `tfsdk:"root_password"`
	Port               types.Int64         `tfsdk:"port"`
	Paused             types.Bool          `tfsdk:"paused"`
	PausePlan          *pausePlan          `tfsdk:"pause_plan"`
	State              types.String        `tfsdk:"state"`
	Version            types.String        `tfsdk:"version"`
	CreatedBy          types.String        `tfsdk:"created_by"`
	CreateTime         types.String        `tfsdk:"create_time"`
	UpdateTime         types.String        `tfsdk:"update_time"`
	RegionDisplayName  types.String        `tfsdk:"region_display_name"`
	Annotations        map[string]string   `tfsdk:"annotations"`
	TiDBNodeSetting    tidbNodeSetting     `tfsdk:"tidb_node_setting"`
	TiKVNodeSetting    tikvNodeSetting     `tfsdk:"tikv_node_setting"`
	TiFlashNodeSetting *tiflashNodeSetting `tfsdk:"tiflash_node_setting"`
}

type pausePlan struct {
	PauseType           types.String `tfsdk:"pause_type"`
	scheduledResumeTime types.String `tfsdk:"scheduled_resume_time"`
}

type tidbNodeSetting struct {
	NodeSpecKey types.String `tfsdk:"node_spec_key"`
	NodeCount   types.Int64  `tfsdk:"node_count"`
	NodeGroups  []nodeGroup  `tfsdk:"node_groups"`
}

type nodeGroup struct {
	NodeSpecKey          types.String          `tfsdk:"node_spec_key"`
	NodeCount            types.Int64           `tfsdk:"node_count"`
	NodeGroupId          types.String          `tfsdk:"node_group_id"`
	NodeGroupDisplayName types.String          `tfsdk:"node_group_display_name"`
	NodeSpecDisplayName  types.String          `tfsdk:"node_spec_display_name"`
	IsDefaultGroup       types.Bool            `tfsdk:"is_default_group"`
	State                types.String          `tfsdk:"state"`
	NodeChangingProgress *nodeChangingProgress `tfsdk:"node_changing_progress"`
}

type nodeChangingProgress struct {
	MatchingNodeSpecNodeCount  types.Int64 `tfsdk:"matching_node_spec_node_count"`
	RemainingDeletionNodeCount types.Int64 `tfsdk:"remaining_deletion_node_count"`
}

type tikvNodeSetting struct {
	NodeSpecKey          types.String          `tfsdk:"node_spec_key"`
	NodeCount            types.Int64           `tfsdk:"node_count"`
	StorageSizeGi        types.Int64           `tfsdk:"storage_size_gi"`
	StorageType          types.String          `tfsdk:"storage_type"`
	NodeSpecDisplayName  types.String          `tfsdk:"node_spec_display_name"`
	NodeChangingProgress *nodeChangingProgress `tfsdk:"node_changing_progress"`
}

type tiflashNodeSetting struct {
	NodeSpecKey          types.String          `tfsdk:"node_spec_key"`
	NodeCount            types.Int64           `tfsdk:"node_count"`
	StorageSizeGi        types.Int64           `tfsdk:"storage_size_gi"`
	StorageType          types.String          `tfsdk:"storage_type"`
	NodeSpecDisplayName  types.String          `tfsdk:"node_spec_display_name"`
	NodeChangingProgress *nodeChangingProgress `tfsdk:"node_changing_progress"`
}

type dedicatedClusterResource struct {
	provider *tidbcloudProvider
}

func NewDedicatedClusterResource() resource.Resource {
	return &dedicatedClusterResource{}
}

func (r *dedicatedClusterResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dedicated_cluster"
}

func (r *dedicatedClusterResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *dedicatedClusterResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "dedicated cluster resource",
		Attributes: map[string]schema.Attribute{
			"project_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the project.",
				Computed:            true,
			},
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of the cluster.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the cluster.",
				Required:            true,
			},
			"cloud_provider": schema.StringAttribute{
				MarkdownDescription: "The cloud provider on which your cluster is hosted.",
				Computed:            true,
			},
			"region_id": schema.StringAttribute{
				MarkdownDescription: "The region where the cluster is deployed.",
				Required:            true,
			},
			"labels": schema.MapAttribute{
				MarkdownDescription: "A map of labels assigned to the cluster.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Map{
					mapplanmodifier.UseStateForUnknown(),
				},
				ElementType: types.StringType,
			},
			"root_password": schema.StringAttribute{
				MarkdownDescription: "The root password to access the cluster.",
				Optional:            true,
			},
			"port": schema.Int64Attribute{
				MarkdownDescription: "The port used for accessing the cluster.",
				Optional:            true,
			},
			"paused": schema.BoolAttribute{
				MarkdownDescription: "Whether the cluster is paused.",
				Optional:            true,
			},
			"pause_plan": schema.SingleNestedAttribute{
				MarkdownDescription: "Pause plan details for the cluster.",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"pause_type": schema.StringAttribute{
						MarkdownDescription: "The type of pause.",
						Optional:            true,
					},
					"scheduled_resume_time": schema.StringAttribute{
						MarkdownDescription: "The scheduled time for resuming the cluster.",
						Optional:            true,
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
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Map{
					mapplanmodifier.UseStateForUnknown(),
				},
				ElementType: types.StringType,
			},
			"tidb_node_setting": schema.SingleNestedAttribute{
				MarkdownDescription: "Settings for TiDB nodes.",
				Required:            true,
				Attributes: map[string]schema.Attribute{
					"node_spec_key": schema.StringAttribute{
						MarkdownDescription: "The node specification key.",
						Required:            true,
					},
					"node_count": schema.Int64Attribute{
						MarkdownDescription: "The number of nodes in the cluster.",
						Required:            true,
					},
					"node_groups": schema.ListNestedAttribute{
						MarkdownDescription: "List of node groups.",
						Optional:            true,
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"node_spec_key": schema.StringAttribute{
									MarkdownDescription: "The node specification key.",
									Computed:            true,
								},
								"node_count": schema.Int64Attribute{
									MarkdownDescription: "The number of nodes in the group.",
									Required:            true,
								},
								"node_group_id": schema.StringAttribute{
									MarkdownDescription: "The ID of the TiDB node group.",
									Computed:            true,
								},
								"node_group_display_name": schema.StringAttribute{
									MarkdownDescription: "The display name of the TiDB node group.",
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
								"node_changing_progress": schema.SingleNestedAttribute{
									MarkdownDescription: "Details of node change progress.",
									Computed:            true,
									Attributes: map[string]schema.Attribute{
										"matching_node_spec_node_count": schema.Int64Attribute{
											MarkdownDescription: "Count of nodes matching the specification.",
											Computed:            true,
										},
										"remaining_deletion_node_count": schema.Int64Attribute{
											MarkdownDescription: "Count of nodes remaining to be deleted.",
											Computed:            true,
										},
									},
								},
							},
						},
					},
				},
			},
			"tikv_node_setting": schema.SingleNestedAttribute{
				MarkdownDescription: "Settings for TiKV nodes.",
				Required:            true,
				Attributes: map[string]schema.Attribute{
					"node_spec_key": schema.StringAttribute{
						MarkdownDescription: "The node specification key.",
						Required:            true,
					},
					"node_count": schema.Int64Attribute{
						MarkdownDescription: "The number of nodes in the cluster.",
						Required:            true,
					},
					"storage_size_gi": schema.Int64Attribute{
						MarkdownDescription: "The storage size in GiB.",
						Required:            true,
					},
					"storage_type": schema.StringAttribute{
						MarkdownDescription: "The storage type.",
						Required:            true,
					},
					"node_spec_display_name": schema.StringAttribute{
						MarkdownDescription: "The display name of the node spec.",
						Computed:            true,
					},
					"node_changing_progress": schema.SingleNestedAttribute{
						MarkdownDescription: "Details of node change progress.",
						Computed:            true,
						Attributes: map[string]schema.Attribute{
							"matching_node_spec_node_count": schema.Int64Attribute{
								MarkdownDescription: "Count of nodes matching the specification.",
								Computed:            true,
							},
							"remaining_deletion_node_count": schema.Int64Attribute{
								MarkdownDescription: "Count of nodes remaining to be deleted.",
								Computed:            true,
							},
						},
					},
				},
			},
			"tiflash_node_setting": schema.SingleNestedAttribute{
				MarkdownDescription: "Settings for TiFlash nodes.",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"node_spec_key": schema.StringAttribute{
						MarkdownDescription: "The node specification key.",
						Required:            true,
					},
					"node_count": schema.Int64Attribute{
						MarkdownDescription: "The number of nodes in the cluster.",
						Required:            true,
					},
					"storage_size_gi": schema.Int64Attribute{
						MarkdownDescription: "The storage size in GiB.",
						Required:            true,
					},
					"storage_type": schema.StringAttribute{
						MarkdownDescription: "The storage type.",
						Required:            true,
					},
					"node_spec_display_name": schema.StringAttribute{
						MarkdownDescription: "The display name of the node spec.",
						Computed:            true,
					},
					"node_changing_progress": schema.SingleNestedAttribute{
						MarkdownDescription: "Details of node change progress.",
						Computed:            true,
						Attributes: map[string]schema.Attribute{
							"matching_node_spec_node_count": schema.Int64Attribute{
								MarkdownDescription: "Count of nodes matching the specification.",
								Computed:            true,
							},
							"remaining_deletion_node_count": schema.Int64Attribute{
								MarkdownDescription: "Count of nodes remaining to be deleted.",
								Computed:            true,
							},
						},
					},
				},
			},
		},
	}
}

func (r dedicatedClusterResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if !r.provider.configured {
		resp.Diagnostics.AddError(
			"Provider not configured",
			"The provider hasn't been configured before apply, likely because it depends on an unknown value from another resource. This leads to weird stuff happening, so we'd prefer if you didn't do that. Thanks!",
		)
		return
	}

	// get data from config
	var data dedicatedClusterResourceData
	diags := req.Config.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Trace(ctx, "created dedicated_cluster_resource")
	body := buildCreateDedicatedClusterBody(data)
	cluster, err := r.provider.DedicatedClient.CreateCluster(ctx, &body)
	if err != nil {
		resp.Diagnostics.AddError("Create Error", fmt.Sprintf("Unable to call CreateCluster, got error: %s", err))
		return
	}
	// set clusterId. other computed attributes are not returned by create, they will be set when refresh
	clusterId := *cluster.ClusterId
	data.ClusterId = types.StringValue(clusterId)
	if r.provider.sync {
		tflog.Info(ctx, "wait dedicated cluster ready")
		cluster, err = WaitDedicatedClusterReady(ctx, clusterCreateTimeout, clusterCreateInterval, clusterId, r.provider.DedicatedClient)
		if err != nil {
			resp.Diagnostics.AddError(
				"Cluster creation failed",
				fmt.Sprintf("Cluster is not ready, get error: %s", err),
			)
			return
		}
		refreshDedicatedClusterResourceData(ctx, cluster, &data)
	}

	// save into the Terraform state.
	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (r dedicatedClusterResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var clusterId string

	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("cluster_id"), &clusterId)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// call read api
	tflog.Trace(ctx, "read dedicated_cluster_resource")
	cluster, err := r.provider.DedicatedClient.GetCluster(ctx, clusterId)
	if err != nil {
		resp.Diagnostics.AddError("Read Error", fmt.Sprintf("Unable to call GetCluster, got error: %s", err))
		return
	}

	// refresh data with read result
	var data dedicatedClusterResourceData
	// root_password, ip_access_list and pause will not return by read api, so we just use state's value even it changed on console!
	// use types.String in case ImportState method throw unhandled null value
	var rootPassword types.String
	var paused *bool
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("config").AtName("root_password"), &rootPassword)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("config").AtName("paused"), &paused)...)
	data.RootPassword = rootPassword
	data.Paused = types.BoolValue(*paused)
	refreshDedicatedClusterResourceData(ctx, cluster, &data)

	// save into the Terraform state
	diags := resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func refreshDedicatedClusterResourceData(ctx context.Context, resp *dedicated.TidbCloudOpenApidedicatedv1beta1Cluster, data *dedicatedClusterResourceData) {
	// must return
	data.ClusterId = types.StringValue(*resp.ClusterId)
	data.Name = types.StringValue(resp.DisplayName)
	data.CloudProvider = types.StringValue(string(*resp.CloudProvider))
	data.RegionId = types.StringValue(resp.RegionId)
	data.Labels = *resp.Labels
	data.Port = types.Int64Value(int64(resp.Port))
	data.State = types.StringValue(string(*resp.State))
	data.Version = types.StringValue(*resp.Version)
	data.CreatedBy = types.StringValue(*resp.CreatedBy)
	data.CreateTime = types.StringValue(resp.CreateTime.String())
	data.UpdateTime = types.StringValue(resp.UpdateTime.String())
	data.RegionDisplayName = types.StringValue(*resp.RegionDisplayName)
	data.Annotations = *resp.Annotations

	// tidb node setting
	var tidbNodeCounts int64
	var dataNodeGroups []nodeGroup
	for _, g := range resp.TidbNodeSetting.TidbNodeGroups {
		var tidbNodeChangingProgress *nodeChangingProgress
		if g.NodeChangingProgress != nil {
			tidbNodeChangingProgress = &nodeChangingProgress{
				MatchingNodeSpecNodeCount:  convertInt32PtrToInt64(g.NodeChangingProgress.MatchingNodeSpecNodeCount),
				RemainingDeletionNodeCount: convertInt32PtrToInt64(g.NodeChangingProgress.RemainingDeletionNodeCount),
			}
		}
		dataNodeGroups = append(dataNodeGroups, nodeGroup{
			NodeSpecKey:          types.StringValue(*g.NodeSpecKey),
			NodeCount:            types.Int64Value(int64(g.NodeCount)),
			NodeGroupId:          types.StringValue(*g.TidbNodeGroupId),
			NodeGroupDisplayName: types.StringValue(*g.DisplayName),
			NodeSpecDisplayName:  types.StringValue(*g.NodeSpecDisplayName),
			IsDefaultGroup:       types.BoolValue(bool(*g.IsDefaultGroup)),
			State:                types.StringValue(string(*g.State)),
			NodeChangingProgress: tidbNodeChangingProgress,
		})
		tidbNodeCounts += int64(g.NodeCount)
	}
	data.TiDBNodeSetting = tidbNodeSetting{
		NodeSpecKey: types.StringValue(resp.TidbNodeSetting.NodeSpecKey),
		NodeCount:   types.Int64Value(tidbNodeCounts),
		NodeGroups:  dataNodeGroups,
	}

	// tikv node setting
	var tikvNodeChangingProgress *nodeChangingProgress
	if resp.TikvNodeSetting.NodeChangingProgress != nil {
		tikvNodeChangingProgress = &nodeChangingProgress{
			MatchingNodeSpecNodeCount:  convertInt32PtrToInt64(resp.TikvNodeSetting.NodeChangingProgress.MatchingNodeSpecNodeCount),
			RemainingDeletionNodeCount: convertInt32PtrToInt64(resp.TikvNodeSetting.NodeChangingProgress.RemainingDeletionNodeCount),
		}
	}
	data.TiKVNodeSetting = tikvNodeSetting{
		NodeSpecKey:          types.StringValue(resp.TikvNodeSetting.NodeSpecKey),
		NodeCount:            types.Int64Value(int64(resp.TikvNodeSetting.NodeCount)),
		StorageSizeGi:        types.Int64Value(int64(resp.TikvNodeSetting.StorageSizeGi)),
		StorageType:          types.StringValue(string(resp.TikvNodeSetting.StorageType)),
		NodeSpecDisplayName:  types.StringValue(*resp.TikvNodeSetting.NodeSpecDisplayName),
		NodeChangingProgress: tikvNodeChangingProgress,
	}

	// may return
	// tiflash node setting
	if resp.TiflashNodeSetting != nil {
		var tiflashNodeChangingProgress *nodeChangingProgress
		if resp.TiflashNodeSetting.NodeChangingProgress != nil {
			tiflashNodeChangingProgress = &nodeChangingProgress{
				MatchingNodeSpecNodeCount:  convertInt32PtrToInt64(resp.TiflashNodeSetting.NodeChangingProgress.MatchingNodeSpecNodeCount),
				RemainingDeletionNodeCount: convertInt32PtrToInt64(resp.TiflashNodeSetting.NodeChangingProgress.RemainingDeletionNodeCount),
			}
		}
		data.TiFlashNodeSetting = &tiflashNodeSetting{
			NodeSpecKey:          types.StringValue(resp.TiflashNodeSetting.NodeSpecKey),
			NodeCount:            types.Int64Value(int64(resp.TiflashNodeSetting.NodeCount)),
			StorageSizeGi:        types.Int64Value(int64(resp.TiflashNodeSetting.StorageSizeGi)),
			StorageType:          types.StringValue(string(resp.TiflashNodeSetting.StorageType)),
			NodeSpecDisplayName:  types.StringValue(*resp.TiflashNodeSetting.NodeSpecDisplayName),
			NodeChangingProgress: tiflashNodeChangingProgress,
		}
	}

	// not return
	// IPAccessList, and password and pause will not update for it will not return by read api
}

// Update since open api is patch without check for the invalid parameter. we do a lot of check here to avoid inconsistency
// check the date can't be updated
// if plan and state is different, we can execute updated
func (r dedicatedClusterResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// // get plan
	// var data clusterResourceData
	// diags := req.Plan.Get(ctx, &data)
	// resp.Diagnostics.Append(diags...)
	// if resp.Diagnostics.HasError() {
	// 	return
	// }
	// // get state
	// var state clusterResourceData
	// diags = req.State.Get(ctx, &state)
	// resp.Diagnostics.Append(diags...)
	// if resp.Diagnostics.HasError() {
	// 	return
	// }

	// // Severless can not be changed now
	// if data.ClusterType == dev {
	// 	resp.Diagnostics.AddError(
	// 		"Update error",
	// 		"Unable to update Serverless cluster",
	// 	)
	// 	return
	// }

	// // only components and paused can be changed now
	// if data.Name != state.Name || data.ClusterType != state.ClusterType || data.Region != state.Region || data.CloudProvider != state.CloudProvider ||
	// 	data.ProjectId != state.ProjectId || data.ClusterId != state.ClusterId {
	// 	resp.Diagnostics.AddError(
	// 		"Update error",
	// 		"You may update the name,cluster_type,region,cloud_provider or projectId. They can not be changed, only components can be changed now",
	// 	)
	// 	return
	// }
	// if !data.Config.Port.IsNull() && !data.Config.Port.IsNull() && data.Config.Port.ValueInt64() != state.Config.Port.ValueInt64() {
	// 	resp.Diagnostics.AddError(
	// 		"Update error",
	// 		"port can not be changed, only components can be changed now",
	// 	)
	// 	return
	// }
	// if data.Config.IPAccessList != nil {
	// 	// You cannot add an IP access list to an existing cluster without an IP rule.
	// 	if len(state.Config.IPAccessList) == 0 {
	// 		resp.Diagnostics.AddError(
	// 			"Update error",
	// 			"ip_access_list can not be added to the existing cluster.",
	// 		)
	// 		return
	// 	}

	// 	// You cannot insert or delete IP rule.
	// 	if len(data.Config.IPAccessList) != len(state.Config.IPAccessList) {
	// 		resp.Diagnostics.AddError(
	// 			"Update error",
	// 			"ip_access_list can not be changed, only components can be changed now",
	// 		)
	// 		return
	// 	}

	// 	// You cannot update the IP rule.
	// 	newIPAccessList := make([]ipAccess, len(data.Config.IPAccessList))
	// 	copy(newIPAccessList, data.Config.IPAccessList)
	// 	sort.Slice(newIPAccessList, func(i, j int) bool {
	// 		return newIPAccessList[i].CIDR < newIPAccessList[j].CIDR
	// 	})

	// 	currentIPAccessList := make([]ipAccess, len(state.Config.IPAccessList))
	// 	copy(currentIPAccessList, state.Config.IPAccessList)
	// 	sort.Slice(currentIPAccessList, func(i, j int) bool {
	// 		return currentIPAccessList[i].CIDR < currentIPAccessList[j].CIDR
	// 	})

	// 	for index, key := range newIPAccessList {
	// 		if currentIPAccessList[index].CIDR != key.CIDR || currentIPAccessList[index].Description != key.Description {
	// 			resp.Diagnostics.AddError(
	// 				"Update error",
	// 				"ip_access_list can not be changed, only components can be changed now",
	// 			)
	// 			return
	// 		}
	// 	}
	// } else {
	// 	// You cannot remove the IP access list.
	// 	if len(state.Config.IPAccessList) > 0 {
	// 		resp.Diagnostics.AddError(
	// 			"Update error",
	// 			"ip_access_list can not be changed, only components can be changed now",
	// 		)
	// 		return
	// 	}
	// }

	// // check Components
	// tidb := data.Config.Components.TiDB
	// tikv := data.Config.Components.TiKV
	// tiflash := data.Config.Components.TiFlash
	// tidbState := state.Config.Components.TiDB
	// tikvState := state.Config.Components.TiKV
	// tiflashState := state.Config.Components.TiFlash
	// if tidb.NodeSize != tidbState.NodeSize {
	// 	resp.Diagnostics.AddError(
	// 		"Update error",
	// 		"tidb node_size can't be changed",
	// 	)
	// 	return
	// }
	// if tikv.NodeSize != tikvState.NodeSize || tikv.StorageSizeGib != tikvState.StorageSizeGib {
	// 	resp.Diagnostics.AddError(
	// 		"Update error",
	// 		"tikv node_size or storage_size_gib can't be changed",
	// 	)
	// 	return
	// }
	// if tiflash != nil && tiflashState != nil {
	// 	// if cluster have tiflash already, then we can't specify NodeSize and StorageSizeGib
	// 	if tiflash.NodeSize != tiflashState.NodeSize || tiflash.StorageSizeGib != tiflashState.StorageSizeGib {
	// 		resp.Diagnostics.AddError(
	// 			"Update error",
	// 			"tiflash node_size or storage_size_gib can't be changed",
	// 		)
	// 		return
	// 	}
	// }

	// // build UpdateClusterBody
	// var updateClusterBody clusterApi.UpdateClusterBody
	// updateClusterBody.Config = &clusterApi.UpdateClusterParamsBodyConfig{}
	// // build paused
	// if data.Config.Paused != nil {
	// 	if state.Config.Paused == nil || *data.Config.Paused != *state.Config.Paused {
	// 		updateClusterBody.Config.Paused = data.Config.Paused
	// 	}
	// }
	// // build components
	// var isComponentsChanged = false
	// if tidb.NodeQuantity != tidbState.NodeQuantity || tikv.NodeQuantity != tikvState.NodeQuantity {
	// 	isComponentsChanged = true
	// }

	// var componentTiFlash *clusterApi.UpdateClusterParamsBodyConfigComponentsTiflash
	// if tiflash != nil {
	// 	if tiflashState == nil {
	// 		isComponentsChanged = true
	// 		componentTiFlash = &clusterApi.UpdateClusterParamsBodyConfigComponentsTiflash{
	// 			NodeQuantity:   &tiflash.NodeQuantity,
	// 			NodeSize:       &tiflash.NodeSize,
	// 			StorageSizeGib: &tiflash.StorageSizeGib,
	// 		}
	// 	} else if tiflash.NodeQuantity != tiflashState.NodeQuantity {
	// 		isComponentsChanged = true
	// 		// NodeSize can't be changed
	// 		componentTiFlash = &clusterApi.UpdateClusterParamsBodyConfigComponentsTiflash{
	// 			NodeQuantity: &tiflash.NodeQuantity,
	// 		}
	// 	}
	// }

	// if isComponentsChanged {
	// 	updateClusterBody.Config.Components = &clusterApi.UpdateClusterParamsBodyConfigComponents{
	// 		Tidb: &clusterApi.UpdateClusterParamsBodyConfigComponentsTidb{
	// 			NodeQuantity: &tidb.NodeQuantity,
	// 		},
	// 		Tikv: &clusterApi.UpdateClusterParamsBodyConfigComponentsTikv{
	// 			NodeQuantity: &tikv.NodeQuantity,
	// 		},
	// 		Tiflash: componentTiFlash,
	// 	}
	// }

	// tflog.Trace(ctx, "update cluster_resource")
	// updateClusterParams := clusterApi.NewUpdateClusterParams().WithProjectID(data.ProjectId).WithClusterID(data.ClusterId.ValueString()).WithBody(updateClusterBody)
	// _, err := r.provider.client.UpdateCluster(updateClusterParams)
	// if err != nil {
	// 	resp.Diagnostics.AddError("Update Error", fmt.Sprintf("Unable to call UpdateClusterById, got error: %s", err))
	// 	return
	// }

	// if r.provider.sync {
	// 	tflog.Info(ctx, "wait cluster ready")
	// 	cluster, err := WaitClusterReady(ctx, clusterUpdateTimeout, clusterUpdateInterval, data.ProjectId, data.ClusterId.ValueString(), r.provider.client)
	// 	if err != nil {
	// 		resp.Diagnostics.AddError(
	// 			"Cluster update failed",
	// 			fmt.Sprintf("Cluster is not ready, get error: %s", err),
	// 		)
	// 		return
	// 	}
	// 	refreshClusterResourceData(ctx, cluster, &data)
	// } else {
	// 	// we refresh for any unknown value. if someone has other opinions which is better, he can delete the refresh logic
	// 	tflog.Trace(ctx, "read cluster_resource")
	// 	getClusterResp, err := r.provider.client.GetCluster(clusterApi.NewGetClusterParams().WithProjectID(data.ProjectId).WithClusterID(data.ClusterId.ValueString()))
	// 	if err != nil {
	// 		resp.Diagnostics.AddError("Update Error", fmt.Sprintf("Unable to call GetClusterById, got error: %s", err))
	// 		return
	// 	}
	// 	refreshClusterResourceData(ctx, getClusterResp.Payload, &data)
	// }

	// // save into the Terraform state.
	// diags = resp.State.Set(ctx, &data)
	// resp.Diagnostics.Append(diags...)
	panic("not implemented")
}

func (r dedicatedClusterResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var clusterId string

	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("cluster_id"), &clusterId)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Trace(ctx, "delete cluster_resource")
	_, err := r.provider.DedicatedClient.DeleteCluster(ctx, clusterId)
	if err != nil {
		resp.Diagnostics.AddError("Delete Error", fmt.Sprintf("Unable to call DeleteCluster, got error: %s", err))
		return
	}
}

// func (r clusterResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
// 	idParts := strings.Split(req.ID, ",")

// 	if len(idParts) != 2 || idParts[0] == "" || idParts[1] == "" {
// 		resp.Diagnostics.AddError(
// 			"Unexpected Import Identifier",
// 			fmt.Sprintf("Expected import identifier with format: project_id,cluster_id. Got: %q", req.ID),
// 		)
// 		return
// 	}

// 	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_id"), idParts[0])...)
// 	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), idParts[1])...)
// }

func WaitDedicatedClusterReady(ctx context.Context, timeout time.Duration, interval time.Duration, clusterId string,
	client tidbcloud.TiDBCloudDedicatedClient) (*dedicated.TidbCloudOpenApidedicatedv1beta1Cluster, error) {
	stateConf := &retry.StateChangeConf{
		Pending: []string{
			string(dedicatedClusterStatusCreating),
			string(dedicatedClusterStatusModifying),
			string(dedicatedClusterStatusResuming),
			string(dedicatedClusterStatusImporting),
			string(dedicatedClusterStatusPausing),
			string(dedicatedClusterStatusUPgrading),
		},
		Target: []string{
			string(dedicatedClusterStatusActive),
			string(dedicatedClusterStatusPaused),
			string(dedicatedClusterStatusMaintenance),
		},
		Timeout:      timeout,
		MinTimeout:   500 * time.Millisecond,
		PollInterval: interval,
		Refresh:      dedicatedClusterStateRefreshFunc(ctx, clusterId, client),
	}

	outputRaw, err := stateConf.WaitForStateContext(ctx)

	if output, ok := outputRaw.(*dedicated.TidbCloudOpenApidedicatedv1beta1Cluster); ok {
		return output, err
	}
	return nil, err
}

func dedicatedClusterStateRefreshFunc(ctx context.Context, clusterId string,
	client tidbcloud.TiDBCloudDedicatedClient) retry.StateRefreshFunc {
	return func() (interface{}, string, error) {
		tflog.Trace(ctx, "Waiting for dedicated cluster ready")
		cluster, err := client.GetCluster(ctx, clusterId)
		if err != nil {
			return nil, "", err
		}
		return cluster, string(*cluster.State), nil
	}
}

func buildCreateDedicatedClusterBody(data dedicatedClusterResourceData) dedicated.TidbCloudOpenApidedicatedv1beta1Cluster {
	displayName := data.Name.ValueString()
	regionId := data.RegionId.ValueString()
	rootPassword := data.RootPassword.ValueString()
	version := data.Version.ValueString()

	// tidb node groups
	var nodeGroups []dedicated.Dedicatedv1beta1TidbNodeGroup
	for _, group := range data.TiDBNodeSetting.NodeGroups {
		displayName := group.NodeGroupDisplayName.ValueString()
		nodeGroups = append(nodeGroups, dedicated.Dedicatedv1beta1TidbNodeGroup{
			NodeCount:   int32(group.NodeCount.ValueInt64()),
			DisplayName: &displayName,
		})
	}

	// tidb node setting
	tidbNodeSpeckKey := data.TiDBNodeSetting.NodeSpecKey.ValueString()
	tidbNodeSetting := dedicated.V1beta1ClusterTidbNodeSetting{
		NodeSpecKey:    tidbNodeSpeckKey,
		TidbNodeGroups: nodeGroups,
	}

	// tikv node setting
	tikvNodeSpeckKey := data.TiKVNodeSetting.NodeSpecKey.ValueString()
	tikvNodeCount := int32(data.TiKVNodeSetting.NodeCount.ValueInt64())
	tikvStorageSizeGi := int32(data.TiKVNodeSetting.StorageSizeGi.ValueInt64())
	tikvStorageType := dedicated.ClusterStorageNodeSettingStorageType(data.TiKVNodeSetting.StorageType.ValueString())
	tikvNodeSetting := dedicated.V1beta1ClusterStorageNodeSetting{
		NodeSpecKey:   tikvNodeSpeckKey,
		NodeCount:     tikvNodeCount,
		StorageSizeGi: tikvStorageSizeGi,
		StorageType:   tikvStorageType,
	}

	var tiflashNodeSetting *dedicated.V1beta1ClusterStorageNodeSetting
	// tiflash node setting
	if data.TiFlashNodeSetting != nil {
		tiflashNodeSpeckKey := data.TiFlashNodeSetting.NodeSpecKey.ValueString()
		tikvNodeCount := int32(data.TiKVNodeSetting.NodeCount.ValueInt64())
		tiflashStorageSizeGi := int32(data.TiFlashNodeSetting.StorageSizeGi.ValueInt64())
		tiflashStorageType := dedicated.ClusterStorageNodeSettingStorageType(data.TiFlashNodeSetting.StorageType.ValueString())
		tiflashNodeSetting = &dedicated.V1beta1ClusterStorageNodeSetting{
			NodeSpecKey:   tiflashNodeSpeckKey,
			NodeCount:     tikvNodeCount,
			StorageSizeGi: tiflashStorageSizeGi,
			StorageType:   tiflashStorageType,
		}
	}

	return dedicated.TidbCloudOpenApidedicatedv1beta1Cluster{
		DisplayName:        displayName,
		RegionId:           regionId,
		Labels:             &data.Labels,
		TidbNodeSetting:    tidbNodeSetting,
		TikvNodeSetting:    tikvNodeSetting,
		TiflashNodeSetting: tiflashNodeSetting,
		Port:               int32(data.Port.ValueInt64()),
		RootPassword:       &rootPassword,
		Version:            &version,
	}
}
