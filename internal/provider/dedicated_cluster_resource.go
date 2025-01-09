package provider

import (
	"context"
	"errors"
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

type dedicatedClusterResourceData struct {
	ProjectId          types.String        `tfsdk:"project_id"`
	ClusterId          types.String        `tfsdk:"id"`
	Name               types.String        `tfsdk:"name"`
	CloudProvider      types.String        `tfsdk:"cloud_provider"`
	RegionId           types.String        `tfsdk:"region_id"`
	Labels             types.Map           `tfsdk:"labels"`
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
	Annotations        types.Map           `tfsdk:"annotations"`
	TiDBNodeSetting    tidbNodeSetting     `tfsdk:"tidb_node_setting"`
	TiKVNodeSetting    tikvNodeSetting     `tfsdk:"tikv_node_setting"`
	TiFlashNodeSetting *tiflashNodeSetting `tfsdk:"tiflash_node_setting"`
}

type pausePlan struct {
	PauseType           types.String `tfsdk:"pause_type"`
	scheduledResumeTime types.String `tfsdk:"scheduled_resume_time"`
}

type tidbNodeSetting struct {
	NodeSpecKey          types.String `tfsdk:"node_spec_key"`
	NodeCount            types.Int64  `tfsdk:"node_count"`
	NodeGroupId          types.String `tfsdk:"node_group_id"`
	NodeGroupDisplayName types.String `tfsdk:"node_group_display_name"`
	NodeSpecDisplayName  types.String `tfsdk:"node_spec_display_name"`
	IsDefaultGroup       types.Bool   `tfsdk:"is_default_group"`
	State                types.String `tfsdk:"state"`
}

type tikvNodeSetting struct {
	NodeSpecKey         types.String `tfsdk:"node_spec_key"`
	NodeCount           types.Int64  `tfsdk:"node_count"`
	StorageSizeGi       types.Int64  `tfsdk:"storage_size_gi"`
	StorageType         types.String `tfsdk:"storage_type"`
	NodeSpecDisplayName types.String `tfsdk:"node_spec_display_name"`
}

type tiflashNodeSetting struct {
	NodeSpecKey         types.String `tfsdk:"node_spec_key"`
	NodeCount           types.Int64  `tfsdk:"node_count"`
	StorageSizeGi       types.Int64  `tfsdk:"storage_size_gi"`
	StorageType         types.String `tfsdk:"storage_type"`
	NodeSpecDisplayName types.String `tfsdk:"node_spec_display_name"`
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
						MarkdownDescription: "The key of the node spec.",
						Required:            true,
					},
					"node_count": schema.Int64Attribute{
						MarkdownDescription: "The number of nodes in the default node group.",
						Required:            true,
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

	tflog.Trace(ctx, "create dedicated_cluster_resource")
	body, err := buildCreateDedicatedClusterBody(ctx, data)
	if err != nil {
		resp.Diagnostics.AddError("Create Error", fmt.Sprintf("Unable to build CreateCluster body, got error: %s", err))
		return
	}
	cluster, err := r.provider.DedicatedClient.CreateCluster(ctx, &body)
	if err != nil {
		resp.Diagnostics.AddError("Create Error", fmt.Sprintf("Unable to call CreateCluster, got error: %s", err))
		return
	}
	// set clusterId. other computed attributes are not returned by create, they will be set when refresh
	clusterId := *cluster.ClusterId
	data.ClusterId = types.StringValue(clusterId)
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

	// save into the Terraform state.
	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (r dedicatedClusterResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var clusterId string

	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("id"), &clusterId)...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Debug(ctx, fmt.Sprintf("read dedicated_cluster_resource clusterid: %s", clusterId))

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

	var rootPassword types.String
	labels, diag := types.MapValueFrom(ctx, types.StringType, *cluster.Labels)
	if diag.HasError() {
		return
	}
	annotations, diag := types.MapValueFrom(ctx, types.StringType, *cluster.Annotations)
	if diag.HasError() {
		return
	}
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("root_password"), &rootPassword)...)
	data.RootPassword = rootPassword
	data.Annotations = annotations
	data.Labels = labels
	refreshDedicatedClusterResourceData(ctx, cluster, &data)

	// save into the Terraform state
	diags := resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func refreshDedicatedClusterResourceData(ctx context.Context, resp *dedicated.TidbCloudOpenApidedicatedv1beta1Cluster, data *dedicatedClusterResourceData) {
	// must return
	labels, diag := types.MapValueFrom(ctx, types.StringType, *resp.Labels)
	if diag.HasError() {
		return
	}
	annotations, diag := types.MapValueFrom(ctx, types.StringType, *resp.Annotations)
	if diag.HasError() {
		return
	}
	data.ClusterId = types.StringValue(*resp.ClusterId)
	data.Name = types.StringValue(resp.DisplayName)
	data.CloudProvider = types.StringValue(string(*resp.CloudProvider))
	data.RegionId = types.StringValue(resp.RegionId)
	data.Labels = labels
	data.Port = types.Int64Value(int64(resp.Port))
	data.State = types.StringValue(string(*resp.State))
	data.Version = types.StringValue(*resp.Version)
	data.CreatedBy = types.StringValue(*resp.CreatedBy)
	data.CreateTime = types.StringValue(resp.CreateTime.String())
	data.UpdateTime = types.StringValue(resp.UpdateTime.String())
	data.RegionDisplayName = types.StringValue(*resp.RegionDisplayName)
	data.Annotations = annotations

	// tidb node setting
	for _, group := range resp.TidbNodeSetting.TidbNodeGroups {
		if *group.IsDefaultGroup {
			data.TiDBNodeSetting = tidbNodeSetting{
				NodeSpecKey:          types.StringValue(*group.NodeSpecKey),
				NodeCount:            types.Int64Value(int64(group.NodeCount)),
				NodeGroupId:          types.StringValue(*group.TidbNodeGroupId),
				NodeGroupDisplayName: types.StringValue(*group.DisplayName),
				NodeSpecDisplayName:  types.StringValue(*group.NodeSpecDisplayName),
				IsDefaultGroup:       types.BoolValue(*group.IsDefaultGroup),
				State:                types.StringValue(string(*group.State)),
			}
		}
	}

	data.TiKVNodeSetting = tikvNodeSetting{
		NodeSpecKey:         types.StringValue(resp.TikvNodeSetting.NodeSpecKey),
		NodeCount:           types.Int64Value(int64(resp.TikvNodeSetting.NodeCount)),
		StorageSizeGi:       types.Int64Value(int64(resp.TikvNodeSetting.StorageSizeGi)),
		StorageType:         types.StringValue(string(resp.TikvNodeSetting.StorageType)),
		NodeSpecDisplayName: types.StringValue(*resp.TikvNodeSetting.NodeSpecDisplayName),
	}

	// may return
	// tiflash node setting
	if resp.TiflashNodeSetting != nil {
		data.TiFlashNodeSetting = &tiflashNodeSetting{
			NodeSpecKey:         types.StringValue(resp.TiflashNodeSetting.NodeSpecKey),
			NodeCount:           types.Int64Value(int64(resp.TiflashNodeSetting.NodeCount)),
			StorageSizeGi:       types.Int64Value(int64(resp.TiflashNodeSetting.StorageSizeGi)),
			StorageType:         types.StringValue(string(resp.TiflashNodeSetting.StorageType)),
			NodeSpecDisplayName: types.StringValue(*resp.TiflashNodeSetting.NodeSpecDisplayName),
		}
	}

}

func (r dedicatedClusterResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// get plan
	var plan dedicatedClusterResourceData
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	// get state
	var state dedicatedClusterResourceData
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if plan.Paused != state.Paused {
		switch plan.Paused.ValueBool() {
		case true:
			tflog.Trace(ctx, "pause cluster")
			_, err := r.provider.DedicatedClient.PauseCluster(ctx, state.ClusterId.ValueString())
			if err != nil {
				resp.Diagnostics.AddError("Pause Error", fmt.Sprintf("Unable to call PauseCluster, got error: %s", err))
				return
			}
		case false:
			tflog.Trace(ctx, "resume cluster")
			_, err := r.provider.DedicatedClient.ResumeCluster(ctx, state.ClusterId.ValueString())
			if err != nil {
				resp.Diagnostics.AddError("Resume Error", fmt.Sprintf("Unable to call ResumeCluster, got error: %s", err))
				return
			}
		}
		state.Paused = plan.Paused
	} else {
		body := &dedicated.ClusterServiceUpdateClusterRequest{}
		// components change
		// tidb
		body.TidbNodeSetting = &dedicated.V1beta1UpdateClusterRequestTidbNodeSetting{
			NodeSpecKey: plan.TiDBNodeSetting.NodeSpecKey.ValueStringPointer(),
		}
		// tikv
		if plan.TiKVNodeSetting != state.TiKVNodeSetting {
			nodeCountInt32 := int32(plan.TiKVNodeSetting.NodeCount.ValueInt64())
			storageSizeGiInt32 := int32(plan.TiKVNodeSetting.StorageSizeGi.ValueInt64())
			body.TikvNodeSetting = &dedicated.V1beta1UpdateClusterRequestStorageNodeSetting{
				NodeSpecKey:   plan.TiKVNodeSetting.NodeSpecKey.ValueStringPointer(),
				NodeCount:     *dedicated.NewNullableInt32(&nodeCountInt32),
				StorageSizeGi: &storageSizeGiInt32,
			}
		}

		// tiflash
		if plan.TiFlashNodeSetting != nil {
			nodeCountInt32 := int32(plan.TiFlashNodeSetting.NodeCount.ValueInt64())
			storageSizeGiInt32 := int32(plan.TiFlashNodeSetting.StorageSizeGi.ValueInt64())
			body.TiflashNodeSetting = &dedicated.V1beta1UpdateClusterRequestStorageNodeSetting{
				NodeSpecKey:   plan.TiFlashNodeSetting.NodeSpecKey.ValueStringPointer(),
				NodeCount:     *dedicated.NewNullableInt32(&nodeCountInt32),
				StorageSizeGi: &storageSizeGiInt32,
			}
		}

		if plan.Name != state.Name {
			body.DisplayName = plan.Name.ValueStringPointer()
		}

		var labels map[string]string
		diag := plan.Labels.ElementsAs(ctx, &labels, false)
		if diag.HasError() {
			return
		}
		body.Labels = &labels

		// call update api
		tflog.Trace(ctx, "update dedicated_cluster_resource")
		_, err := r.provider.DedicatedClient.UpdateCluster(ctx, state.ClusterId.ValueString(), body)
		if err != nil {
			resp.Diagnostics.AddError("Update Error", fmt.Sprintf("Unable to call UpdateCluster, got error: %s", err))
			return
		}
	}

	tflog.Info(ctx, "wait cluster ready")
	cluster, err := WaitDedicatedClusterReady(ctx, clusterUpdateTimeout, clusterUpdateInterval, state.ClusterId.ValueString(), r.provider.DedicatedClient)
	if err != nil {
		resp.Diagnostics.AddError(
			"Cluster update failed",
			fmt.Sprintf("Cluster is not ready, get error: %s", err),
		)
		return
	}
	refreshDedicatedClusterResourceData(ctx, cluster, &state)

	// save into the Terraform state.
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

func (r dedicatedClusterResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var clusterId string

	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("id"), &clusterId)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Trace(ctx, "delete dedicated_cluster_resource")
	_, err := r.provider.DedicatedClient.DeleteCluster(ctx, clusterId)
	if err != nil {
		resp.Diagnostics.AddError("Delete Error", fmt.Sprintf("Unable to call DeleteCluster, got error: %s", err))
		return
	}
}

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

func buildCreateDedicatedClusterBody(ctx context.Context, data dedicatedClusterResourceData) (dedicated.TidbCloudOpenApidedicatedv1beta1Cluster, error) {
	displayName := data.Name.ValueString()
	regionId := data.RegionId.ValueString()
	rootPassword := data.RootPassword.ValueString()
	version := data.Version.ValueString()

	// tidb node groups
	var nodeGroups []dedicated.Dedicatedv1beta1TidbNodeGroup
	nodeGroups = append(nodeGroups, dedicated.Dedicatedv1beta1TidbNodeGroup{
		NodeCount: int32(data.TiDBNodeSetting.NodeCount.ValueInt64()),
	})

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

	var labels map[string]string
	diag := data.Labels.ElementsAs(ctx, &labels, false)
	if diag.HasError() {
		return dedicated.TidbCloudOpenApidedicatedv1beta1Cluster{}, errors.New("Unable to convert labels")
	}

	return dedicated.TidbCloudOpenApidedicatedv1beta1Cluster{
		DisplayName:        displayName,
		RegionId:           regionId,
		Labels:             &labels,
		TidbNodeSetting:    tidbNodeSetting,
		TikvNodeSetting:    tikvNodeSetting,
		TiflashNodeSetting: tiflashNodeSetting,
		Port:               int32(data.Port.ValueInt64()),
		RootPassword:       &rootPassword,
		Version:            &version,
	}, nil
}

// // normalizeMap is used to sort the map to avoid the diff form terraform state and client response
// func normalizeMap(m map[string]string) map[string]string {
// 	sortedKeys := make([]string, 0, len(m))
// 	for k := range m {
// 		sortedKeys = append(sortedKeys, k)
// 	}
// 	sort.Strings(sortedKeys)

// 	sortedMap := make(map[string]string)
// 	for _, k := range sortedKeys {
// 		sortedMap[k] = m[k]
// 	}

// 	return sortedMap
// }
