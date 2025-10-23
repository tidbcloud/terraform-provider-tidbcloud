package provider

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int32planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"github.com/tidbcloud/terraform-provider-tidbcloud/tidbcloud"
	"github.com/tidbcloud/tidbcloud-cli/pkg/tidbcloud/v1beta1/dedicated"
)

type dedicatedClusterResourceData struct {
	ProjectId          types.String        `tfsdk:"project_id"`
	ClusterId          types.String        `tfsdk:"cluster_id"`
	DisplayName        types.String        `tfsdk:"display_name"`
	CloudProvider      types.String        `tfsdk:"cloud_provider"`
	RegionId           types.String        `tfsdk:"region_id"`
	Labels             types.Map           `tfsdk:"labels"`
	RootPassword       types.String        `tfsdk:"root_password"`
	Port               types.Int32         `tfsdk:"port"`
	Paused             types.Bool          `tfsdk:"paused"`
	PausePlan          types.Object        `tfsdk:"pause_plan"`
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
	ScheduledResumeTime types.String `tfsdk:"scheduled_resume_time"`
}

var pausePlanAttrTypes = map[string]attr.Type{
	"pause_type":            types.StringType,
	"scheduled_resume_time": types.StringType,
}

type tidbNodeSetting struct {
	NodeSpecKey           types.String           `tfsdk:"node_spec_key"`
	NodeCount             types.Int32            `tfsdk:"node_count"`
	NodeGroupId           types.String           `tfsdk:"node_group_id"`
	NodeGroupDisplayName  types.String           `tfsdk:"node_group_display_name"`
	NodeSpecDisplayName   types.String           `tfsdk:"node_spec_display_name"`
	IsDefaultGroup        types.Bool             `tfsdk:"is_default_group"`
	State                 types.String           `tfsdk:"state"`
	Endpoints             types.List             `tfsdk:"endpoints"`
	TiProxySetting        *tiProxySetting        `tfsdk:"tiproxy_setting"`
	PublicEndpointSetting *publicEndpointSetting `tfsdk:"public_endpoint_setting"`
}

type publicEndpointSetting struct {
	Enabled      types.Bool `tfsdk:"enabled"`
	IPAccessList types.List `tfsdk:"ip_access_list"`
}

var ipAccessListItemAttrTypes = map[string]attr.Type{
	"cidr_notation": types.StringType,
	"description":   types.StringType,
}

type tiProxySetting struct {
	NodeSpecKey         types.String `tfsdk:"node_spec_key"`
	NodeSpecVersion     types.String `tfsdk:"node_spec_version"`
	NodeCount           types.Int32  `tfsdk:"node_count"`
	NodeSpecDisplayName types.String `tfsdk:"node_spec_display_name"`
}

var endpointItemAttrTypes = map[string]attr.Type{
	"host":            types.StringType,
	"port":            types.Int32Type,
	"connection_type": types.StringType,
}

type tikvNodeSetting struct {
	NodeSpecKey         types.String `tfsdk:"node_spec_key"`
	NodeCount           types.Int32  `tfsdk:"node_count"`
	StorageSizeGi       types.Int32  `tfsdk:"storage_size_gi"`
	StorageType         types.String `tfsdk:"storage_type"`
	NodeSpecDisplayName types.String `tfsdk:"node_spec_display_name"`
	RaftStoreIOPS       types.Int32  `tfsdk:"raft_store_iops"`
}

type tiflashNodeSetting struct {
	NodeSpecKey         types.String `tfsdk:"node_spec_key"`
	NodeCount           types.Int32  `tfsdk:"node_count"`
	StorageSizeGi       types.Int32  `tfsdk:"storage_size_gi"`
	StorageType         types.String `tfsdk:"storage_type"`
	NodeSpecDisplayName types.String `tfsdk:"node_spec_display_name"`
	RaftStoreIOPS       types.Int32  `tfsdk:"raft_store_iops"`
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
				MarkdownDescription: "The ID of the project. When not provided, the default project will be used.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
			},
			"cluster_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the cluster.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"display_name": schema.StringAttribute{
				MarkdownDescription: "The name of the cluster.",
				Required:            true,
			},
			"cloud_provider": schema.StringAttribute{
				MarkdownDescription: "The cloud provider on which your cluster is hosted.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"region_id": schema.StringAttribute{
				MarkdownDescription: "The region where the cluster is deployed.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"labels": schema.MapAttribute{
				MarkdownDescription: "A map of labels assigned to the cluster.",
				Computed:            true,
				ElementType:         types.StringType,
			},
			"root_password": schema.StringAttribute{
				MarkdownDescription: "The root password to access the cluster.",
				Optional:            true,
				Sensitive:           true,
			},
			"port": schema.Int32Attribute{
				MarkdownDescription: "The port used for accessing the cluster.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Int32{
					int32planmodifier.UseStateForUnknown(),
				},
			},
			"paused": schema.BoolAttribute{
				MarkdownDescription: "Whether the cluster is paused.",
				Optional:            true,
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
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"create_time": schema.StringAttribute{
				MarkdownDescription: "The creation time of the cluster.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"update_time": schema.StringAttribute{
				MarkdownDescription: "The last update time of the cluster.",
				Computed:            true,
			},
			"region_display_name": schema.StringAttribute{
				MarkdownDescription: "The display name of the region.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"annotations": schema.MapAttribute{
				MarkdownDescription: "A map of annotations for the cluster.",
				Computed:            true,
				ElementType:         types.StringType,
			},
			"tidb_node_setting": schema.SingleNestedAttribute{
				MarkdownDescription: "Settings for TiDB nodes.",
				Required:            true,
				Attributes: map[string]schema.Attribute{
					"node_spec_key": schema.StringAttribute{
						MarkdownDescription: "The key of the node spec.",
						Required:            true,
					},
					"node_count": schema.Int32Attribute{
						MarkdownDescription: "The number of nodes in the default node group.",
						Required:            true,
					},
					"node_group_id": schema.StringAttribute{
						MarkdownDescription: "The ID of the default node group.",
						Computed:            true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"node_group_display_name": schema.StringAttribute{
						MarkdownDescription: "The display name of the default node group.",
						Computed:            true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"node_spec_display_name": schema.StringAttribute{
						MarkdownDescription: "The display name of the node spec.",
						Computed:            true,
					},
					"is_default_group": schema.BoolAttribute{
						MarkdownDescription: "Indicates if this is the default group.",
						Computed:            true,
						PlanModifiers: []planmodifier.Bool{
							boolplanmodifier.UseStateForUnknown(),
						},
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
						Optional:            true,
						Attributes: map[string]schema.Attribute{
							"node_spec_key": schema.StringAttribute{
								MarkdownDescription: "The key of the node spec.",
								Optional:            true,
							},
							"node_spec_version": schema.StringAttribute{
								MarkdownDescription: "The node specification version.",
								Computed:            true,
							},
							"node_count": schema.Int32Attribute{
								MarkdownDescription: "The number of TiProxy nodes.",
								Optional:            true,
							},
							"node_spec_display_name": schema.StringAttribute{
								MarkdownDescription: "The display name of the node spec.",
								Computed:            true,
							},
						},
					},
					"public_endpoint_setting": schema.SingleNestedAttribute{
						MarkdownDescription: "Settings for public endpoint.",
						Optional:            true,
						Computed:            true,
						PlanModifiers: []planmodifier.Object{
							objectplanmodifier.UseStateForUnknown(),
						},
						Attributes: map[string]schema.Attribute{
							"enabled": schema.BoolAttribute{
								MarkdownDescription: "Whether public endpoint is enabled.",
								Optional:            true,
								Computed:            true,
								PlanModifiers: []planmodifier.Bool{
									boolplanmodifier.UseStateForUnknown(),
								},
							},
							"ip_access_list": schema.ListAttribute{
								MarkdownDescription: "IP access list for the public endpoint.",
								Optional:            true,
								Computed:            true,
								PlanModifiers: []planmodifier.List{
									listplanmodifier.UseStateForUnknown(),
								},
								ElementType: types.ObjectType{AttrTypes: ipAccessListItemAttrTypes},
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
					"node_count": schema.Int32Attribute{
						MarkdownDescription: "The number of nodes in the cluster.",
						Required:            true,
					},
					"storage_size_gi": schema.Int32Attribute{
						MarkdownDescription: "The storage size in GiB.",
						Required:            true,
					},
					"storage_type": schema.StringAttribute{
						MarkdownDescription: "The storage type." +
							"- Basic: Data disk: gp3; Raft log disk: none." +
							"- Standard: Data disk: gp3; Raft log disk: gp3." +
							"- Performance: Data disk: gp3; Raft log disk: io2." +
							"- Plus: Data disk: io2; Raft log disk: none.",
						Optional: true,
						Computed: true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"node_spec_display_name": schema.StringAttribute{
						MarkdownDescription: "The display name of the node spec.",
						Computed:            true,
					},
					"raft_store_iops": schema.Int32Attribute{
						MarkdownDescription: "The IOPS of raft store",
						Optional:            true,
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
					"node_count": schema.Int32Attribute{
						MarkdownDescription: "The number of nodes in the cluster.",
						Required:            true,
					},
					"storage_size_gi": schema.Int32Attribute{
						MarkdownDescription: "The storage size in GiB.",
						Required:            true,
					},
					"storage_type": schema.StringAttribute{
						MarkdownDescription: "The storage type." +
							"- Basic: Data disk: gp3; Raft log disk: none." +
							"- Standard: Data disk: gp3; Raft log disk: gp3." +
							"- Performance: Data disk: gp3; Raft log disk: io2." +
							"- Plus: Data disk: io2; Raft log disk: none.",
						Optional: true,
						Computed: true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"node_spec_display_name": schema.StringAttribute{
						MarkdownDescription: "The display name of the node spec.",
						Computed:            true,
					},
					"raft_store_iops": schema.Int32Attribute{
						MarkdownDescription: "The IOPS of raft store",
						Optional:            true,
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
	body, err := buildCreateDedicatedClusterBody(data)
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

	// using tidb node group api create public endpoint setting
	pes, err := updatePublicEndpointSetting(ctx, r.provider.DedicatedClient, data.ClusterId.ValueString(), data.TiDBNodeSetting.NodeGroupId.ValueString(), data.TiDBNodeSetting.PublicEndpointSetting)
	if err != nil {
		resp.Diagnostics.AddError("Update Error", fmt.Sprintf("Unable to call UpdatePublicEndpoint, got error: %s", err))
		return
	}
	data.TiDBNodeSetting.PublicEndpointSetting = pes

	// sleep 1 minute to wait the endpoints updated, then get cluster to refresh the endpoints
	time.Sleep(1 * time.Minute)
	cluster, err = r.provider.DedicatedClient.GetCluster(ctx, clusterId)
	if err != nil {
		resp.Diagnostics.AddError("Read Error", fmt.Sprintf("Unable to call GetCluster, got error: %s", err))
		return
	}

	refreshDedicatedClusterResourceData(ctx, cluster, &data)
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
	tflog.Debug(ctx, fmt.Sprintf("read dedicated_cluster_resource cluster_id: %s", clusterId))

	// call read api
	tflog.Trace(ctx, "read dedicated_cluster_resource")
	cluster, err := r.provider.DedicatedClient.GetCluster(ctx, clusterId)
	if err != nil {
		resp.Diagnostics.AddError("Read Error", fmt.Sprintf("Unable to call GetCluster, got error: %s", err))
		return
	}

	var data dedicatedClusterResourceData
	var rootPassword types.String
	var paused types.Bool
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("root_password"), &rootPassword)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("paused"), &paused)...)
	data.RootPassword = rootPassword
	data.Paused = paused

	refreshDedicatedClusterResourceData(ctx, cluster, &data)

	publicEndpointSetting, err := r.provider.DedicatedClient.GetPublicEndpoint(ctx, clusterId, data.TiDBNodeSetting.NodeGroupId.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Read Error", fmt.Sprintf("Unable to call GetPublicEndpoint, got error: %s", err))
		return
	}
	data.TiDBNodeSetting.PublicEndpointSetting = convertDedicatedPublicEndpointSetting(publicEndpointSetting)

	// save into the Terraform state
	diags := resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func refreshDedicatedClusterResourceData(ctx context.Context, resp *dedicated.TidbCloudOpenApidedicatedv1beta1Cluster, data *dedicatedClusterResourceData) diag.Diagnostics {
	labels, diags := types.MapValueFrom(ctx, types.StringType, *resp.Labels)
	if diags.HasError() {
		return diags
	}
	annotations, diags := types.MapValueFrom(ctx, types.StringType, *resp.Annotations)
	if diags.HasError() {
		return diags
	}
	data.ClusterId = types.StringValue(*resp.ClusterId)
	data.DisplayName = types.StringValue(resp.DisplayName)
	data.CloudProvider = types.StringValue(string(*resp.CloudProvider))
	data.RegionId = types.StringValue(resp.RegionId)
	data.Labels = labels
	data.Port = types.Int32Value(resp.Port)
	data.State = types.StringValue(string(*resp.State))
	data.Version = types.StringValue(*resp.Version)
	data.CreatedBy = types.StringValue(*resp.CreatedBy)
	data.CreateTime = types.StringValue(resp.CreateTime.String())
	data.UpdateTime = types.StringValue(resp.UpdateTime.String())
	data.RegionDisplayName = types.StringValue(*resp.RegionDisplayName)
	data.Annotations = annotations
	data.ProjectId = types.StringValue((*resp.Labels)[LabelsKeyProjectId])

	// tidb node setting
	for _, group := range resp.TidbNodeSetting.TidbNodeGroups {
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
			if group.TiproxySetting != nil {
				data.TiDBNodeSetting.TiProxySetting = &tiProxySetting{
					NodeSpecKey: 	   types.StringValue(group.TiproxySetting.NodeSpecKey),
					NodeSpecVersion:  types.StringValue(*group.TiproxySetting.NodeSpecVersion),
					NodeCount: 	  types.Int32Value(*group.TiproxySetting.NodeCount.Get()),
					NodeSpecDisplayName: types.StringValue(*group.TiproxySetting.NodeSpecDisplayName),
				}
			}

			data.TiDBNodeSetting.NodeSpecKey = types.StringValue(*group.NodeSpecKey)
			data.TiDBNodeSetting.NodeCount = types.Int32Value(group.NodeCount)
			data.TiDBNodeSetting.NodeGroupId = types.StringValue(*group.TidbNodeGroupId)
			data.TiDBNodeSetting.NodeGroupDisplayName = types.StringValue(*group.DisplayName)
			data.TiDBNodeSetting.NodeSpecDisplayName = types.StringValue(*group.NodeSpecDisplayName)
			data.TiDBNodeSetting.IsDefaultGroup = types.BoolValue(*group.IsDefaultGroup)
			data.TiDBNodeSetting.State = types.StringValue(string(*group.State))
			data.TiDBNodeSetting.Endpoints = endpointsList
		}
	}

	// tikv node setting
	data.TiKVNodeSetting.NodeSpecKey = types.StringValue(resp.TikvNodeSetting.NodeSpecKey)
	data.TiKVNodeSetting.NodeCount = types.Int32Value(resp.TikvNodeSetting.NodeCount)
	data.TiKVNodeSetting.StorageSizeGi = types.Int32Value(resp.TikvNodeSetting.StorageSizeGi)
	data.TiKVNodeSetting.StorageType = types.StringValue(string(*resp.TikvNodeSetting.StorageType))
	data.TiKVNodeSetting.NodeSpecDisplayName = types.StringValue(*resp.TikvNodeSetting.NodeSpecDisplayName)

	// tiflash node setting
	if resp.TiflashNodeSetting != nil {
		if data.TiFlashNodeSetting == nil {
			data.TiFlashNodeSetting = &tiflashNodeSetting{
				NodeSpecKey:         types.StringValue(resp.TiflashNodeSetting.NodeSpecKey),
				NodeCount:           types.Int32Value(resp.TiflashNodeSetting.NodeCount),
				StorageSizeGi:       types.Int32Value(resp.TiflashNodeSetting.StorageSizeGi),
				StorageType:         types.StringValue(string(*resp.TiflashNodeSetting.StorageType)),
				NodeSpecDisplayName: types.StringValue(*resp.TiflashNodeSetting.NodeSpecDisplayName),
			}
		} else {
			data.TiFlashNodeSetting.NodeSpecKey = types.StringValue(resp.TiflashNodeSetting.NodeSpecKey)
			data.TiFlashNodeSetting.NodeCount = types.Int32Value(resp.TiflashNodeSetting.NodeCount)
			data.TiFlashNodeSetting.StorageSizeGi = types.Int32Value(resp.TiflashNodeSetting.StorageSizeGi)
			data.TiFlashNodeSetting.StorageType = types.StringValue(string(*resp.TiflashNodeSetting.StorageType))
			data.TiFlashNodeSetting.NodeSpecDisplayName = types.StringValue(*resp.TiflashNodeSetting.NodeSpecDisplayName)
		}
	}

	if resp.PausePlan != nil {
		p := pausePlan{
			PauseType: types.StringValue(string(resp.PausePlan.PauseType)),
		}
		if resp.PausePlan.ScheduledResumeTime != nil {
			p.ScheduledResumeTime = types.StringValue(resp.PausePlan.ScheduledResumeTime.String())
		}
		data.PausePlan, diags = types.ObjectValueFrom(ctx, pausePlanAttrTypes, p)
		if diags.HasError() {
			return diags
		}
	} else {
		data.PausePlan = types.ObjectNull(pausePlanAttrTypes)
	}
	return diags
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

	// Check if paused state is changing
	isPauseStateChanging := plan.Paused.ValueBool() != state.Paused.ValueBool()

	// Check if TiFlashNodeSetting is changing
	isTiFlashNodeSettingChanging := false
	if plan.TiFlashNodeSetting != nil && state.TiFlashNodeSetting != nil {
		isTiFlashNodeSettingChanging = plan.TiFlashNodeSetting.NodeCount != state.TiFlashNodeSetting.NodeCount ||
			plan.TiFlashNodeSetting.NodeSpecKey != state.TiFlashNodeSetting.NodeSpecKey ||
			plan.TiFlashNodeSetting.StorageSizeGi != state.TiFlashNodeSetting.StorageSizeGi ||
			plan.TiFlashNodeSetting.StorageType != state.TiFlashNodeSetting.StorageType ||
			plan.TiFlashNodeSetting.RaftStoreIOPS != state.TiFlashNodeSetting.RaftStoreIOPS
	} else if plan.TiFlashNodeSetting != nil {
		isTiFlashNodeSettingChanging = true
	} else if state.TiFlashNodeSetting != nil {
		isTiFlashNodeSettingChanging = true
	}

	isPublicEndpointSettingChanging := false
	if plan.TiDBNodeSetting.PublicEndpointSetting != nil && state.TiDBNodeSetting.PublicEndpointSetting != nil {
		isPublicEndpointSettingChanging = !plan.TiDBNodeSetting.PublicEndpointSetting.Enabled.Equal(state.TiDBNodeSetting.PublicEndpointSetting.Enabled) ||
			!plan.TiDBNodeSetting.PublicEndpointSetting.IPAccessList.Equal(state.TiDBNodeSetting.PublicEndpointSetting.IPAccessList)
	} else if plan.TiDBNodeSetting.PublicEndpointSetting != nil {
		isPublicEndpointSettingChanging = true
	} else if state.TiDBNodeSetting.PublicEndpointSetting != nil {
		isPublicEndpointSettingChanging = true
	}

	// Check if any other attributes are changing
	isOtherAttributesChanging := plan.DisplayName != state.DisplayName ||
		plan.TiDBNodeSetting.NodeCount != state.TiDBNodeSetting.NodeCount ||
		plan.TiDBNodeSetting.NodeSpecKey != state.TiDBNodeSetting.NodeSpecKey ||
		plan.TiDBNodeSetting.TiProxySetting != state.TiDBNodeSetting.TiProxySetting ||
		isPublicEndpointSettingChanging ||

		plan.TiKVNodeSetting.NodeCount != state.TiKVNodeSetting.NodeCount ||
		plan.TiKVNodeSetting.NodeSpecKey != state.TiKVNodeSetting.NodeSpecKey ||
		plan.TiKVNodeSetting.StorageSizeGi != state.TiKVNodeSetting.StorageSizeGi ||
		plan.TiKVNodeSetting.StorageType != state.TiKVNodeSetting.StorageType ||
		plan.TiKVNodeSetting.RaftStoreIOPS != state.TiKVNodeSetting.RaftStoreIOPS ||

		plan.RootPassword != state.RootPassword ||
		isTiFlashNodeSettingChanging

	// If trying to change pause state along with other attributes, return an error
	if isPauseStateChanging && isOtherAttributesChanging {
		resp.Diagnostics.AddError(
			"Invalid Update",
			"Cannot change cluster pause state along with other attributes. Please update pause state in a separate operation.",
		)
		return
	}

	if isPauseStateChanging {
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
	} else {
		if plan.RootPassword != state.RootPassword {
			err := r.provider.DedicatedClient.ChangeClusterRootPassword(ctx, state.ClusterId.ValueString(), &dedicated.V1beta1ClusterServiceResetRootPasswordBody{
				RootPassword: plan.RootPassword.ValueString(),
			})
			if err != nil {
				resp.Diagnostics.AddError("Update Error", fmt.Sprintf("Unable to call ChangeClusterRootPassword, got error: %s", err))
				return
			}
		}

		if isPublicEndpointSettingChanging {
			// using tidb node group api update public endpoint setting
			pes, err := updatePublicEndpointSetting(ctx, r.provider.DedicatedClient, state.ClusterId.ValueString(), state.TiDBNodeSetting.NodeGroupId.ValueString(), plan.TiDBNodeSetting.PublicEndpointSetting)
			if err != nil {
				resp.Diagnostics.AddError("Update Error", fmt.Sprintf("Unable to call UpdatePublicEndpoint, got error: %s", err))
				return
			}
			state.TiDBNodeSetting.PublicEndpointSetting = pes
			// because update public endpoint doesn't change the state of the cluster, sleep 1 minute to wait the endpoints updated
			time.Sleep(1 * time.Minute)
		}

		body := &dedicated.TheUpdatedClusterConfiguration{}
		// components change
		// tidb
		defaultNodeGroup := dedicated.UpdateClusterRequestTidbNodeSettingTidbNodeGroup{}
		nodeCount := plan.TiDBNodeSetting.NodeCount.ValueInt32()
		defaultNodeGroup.NodeCount = *dedicated.NewNullableInt32(&nodeCount)
		if plan.TiDBNodeSetting.TiProxySetting != nil {
			tiProxySetting := dedicated.Dedicatedv1beta1TidbNodeGroupTiProxySetting{}
			tiProxyNodeCount := plan.TiDBNodeSetting.TiProxySetting.NodeCount.ValueInt32()
			tiProxyNodeSpecKey := plan.TiDBNodeSetting.TiProxySetting.NodeSpecKey.ValueString()
			tiProxySetting.NodeCount = *dedicated.NewNullableInt32(&tiProxyNodeCount)
			tiProxySetting.NodeSpecKey = tiProxyNodeSpecKey
			defaultNodeGroup.TiproxySetting = &tiProxySetting
		}
		body.TidbNodeSetting = &dedicated.V1beta1UpdateClusterRequestTidbNodeSetting{
			NodeSpecKey: plan.TiDBNodeSetting.NodeSpecKey.ValueStringPointer(),
			TidbNodeGroups: []dedicated.UpdateClusterRequestTidbNodeSettingTidbNodeGroup{
				defaultNodeGroup,
			},
		}

		// tikv
		nodeCountInt32 := int32(plan.TiKVNodeSetting.NodeCount.ValueInt32())
		storageSizeGiInt32 := int32(plan.TiKVNodeSetting.StorageSizeGi.ValueInt32())
		storageType := dedicated.ClusterStorageNodeSettingStorageType(plan.TiKVNodeSetting.StorageType.ValueString())
		body.TikvNodeSetting = &dedicated.V1beta1UpdateClusterRequestStorageNodeSetting{
			NodeSpecKey:   plan.TiKVNodeSetting.NodeSpecKey.ValueStringPointer(),
			NodeCount:     *dedicated.NewNullableInt32(&nodeCountInt32),
			StorageSizeGi: &storageSizeGiInt32,
			StorageType:   &storageType,
		}
		if IsKnown(plan.TiKVNodeSetting.RaftStoreIOPS) {
			raftStoreIOPS := plan.TiKVNodeSetting.RaftStoreIOPS.ValueInt32()
			body.TikvNodeSetting.RaftStoreIops = *dedicated.NewNullableInt32(&raftStoreIOPS)
		}

		// tiflash
		if plan.TiFlashNodeSetting != nil {
			nodeCountInt32 := int32(plan.TiFlashNodeSetting.NodeCount.ValueInt32())
			storageSizeGiInt32 := int32(plan.TiFlashNodeSetting.StorageSizeGi.ValueInt32())
			storageType := dedicated.ClusterStorageNodeSettingStorageType(plan.TiFlashNodeSetting.StorageType.ValueString())
			body.TiflashNodeSetting = &dedicated.V1beta1UpdateClusterRequestStorageNodeSetting{
				NodeSpecKey:   plan.TiFlashNodeSetting.NodeSpecKey.ValueStringPointer(),
				NodeCount:     *dedicated.NewNullableInt32(&nodeCountInt32),
				StorageSizeGi: &storageSizeGiInt32,
				StorageType:   &storageType,
			}
			if IsKnown(plan.TiFlashNodeSetting.RaftStoreIOPS) {
				raftStoreIOPS := plan.TiFlashNodeSetting.RaftStoreIOPS.ValueInt32()
				body.TiflashNodeSetting.RaftStoreIops = *dedicated.NewNullableInt32(&raftStoreIOPS)
			}
		}

		if plan.DisplayName != state.DisplayName {
			body.DisplayName = plan.DisplayName.ValueStringPointer()
		}

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
	state.Paused = plan.Paused
	state.RootPassword = plan.RootPassword

	// save into the Terraform state.
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

func convertDedicatedPublicEndpointSetting(resp *dedicated.V1beta1PublicEndpointSetting) *publicEndpointSetting {
	publicEndpointSetting := publicEndpointSetting{}
	if resp.Enabled.IsSet() {
		publicEndpointSetting.Enabled = types.BoolValue(*resp.Enabled.Get())
	}
	items := make([]attr.Value, 0, len(resp.IpAccessList))
	for _, ip := range resp.IpAccessList {
		objVal, _ := types.ObjectValue(ipAccessListItemAttrTypes,
			map[string]attr.Value{
				"cidr_notation": types.StringValue(ip.CidrNotation),
				"description":   types.StringValue(*ip.Description),
			})
		items = append(items, objVal)
	}
	publicEndpointSetting.IPAccessList, _ = types.ListValue(types.ObjectType{AttrTypes: map[string]attr.Type{
		"cidr_notation": types.StringType,
		"description":   types.StringType,
	}}, items)
	return &publicEndpointSetting
}

func (r dedicatedClusterResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var clusterId string

	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("cluster_id"), &clusterId)...)
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

func (r dedicatedClusterResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("cluster_id"), req, resp)
}

func WaitDedicatedClusterReady(ctx context.Context, timeout time.Duration, interval time.Duration, clusterId string,
	client tidbcloud.TiDBCloudDedicatedClient) (*dedicated.TidbCloudOpenApidedicatedv1beta1Cluster, error) {
	stateConf := &retry.StateChangeConf{
		Pending: []string{
			string(dedicated.COMMONV1BETA1CLUSTERSTATE_CREATING),
			string(dedicated.COMMONV1BETA1CLUSTERSTATE_MODIFYING),
			string(dedicated.COMMONV1BETA1CLUSTERSTATE_RESUMING),
			string(dedicated.COMMONV1BETA1CLUSTERSTATE_IMPORTING),
			string(dedicated.COMMONV1BETA1CLUSTERSTATE_PAUSING),
			string(dedicated.COMMONV1BETA1CLUSTERSTATE_UPGRADING),
			string(dedicated.COMMONV1BETA1CLUSTERSTATE_DELETING),
		},
		Target: []string{
			string(dedicated.COMMONV1BETA1CLUSTERSTATE_ACTIVE),
			string(dedicated.COMMONV1BETA1CLUSTERSTATE_PAUSED),
			string(dedicated.COMMONV1BETA1CLUSTERSTATE_MAINTENANCE),
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

func buildCreateDedicatedClusterBody(data dedicatedClusterResourceData) (dedicated.TidbCloudOpenApidedicatedv1beta1Cluster, error) {
	if data.Paused.ValueBool() {
		return dedicated.TidbCloudOpenApidedicatedv1beta1Cluster{}, errors.New("can not create a cluster with paused set to true")
	}

	displayName := data.DisplayName.ValueString()
	regionId := data.RegionId.ValueString()
	rootPassword := data.RootPassword.ValueString()

	// tidb node groups
	defaultNodeGroup := dedicated.Dedicatedv1beta1TidbNodeGroup{}
	defaultNodeGroup.NodeCount = data.TiDBNodeSetting.NodeCount.ValueInt32()
	if data.TiDBNodeSetting.TiProxySetting != nil {
		tiProxySetting := dedicated.Dedicatedv1beta1TidbNodeGroupTiProxySetting{}
		tiProxyNodeCount := data.TiDBNodeSetting.TiProxySetting.NodeCount.ValueInt32()
		tiProxyNodeSpecKey := data.TiDBNodeSetting.TiProxySetting.NodeSpecKey.ValueString()
		tiProxySetting.NodeCount = *dedicated.NewNullableInt32(&tiProxyNodeCount)
		tiProxySetting.NodeSpecKey = tiProxyNodeSpecKey
		defaultNodeGroup.TiproxySetting = &tiProxySetting
	}
	nodeGroups := []dedicated.Dedicatedv1beta1TidbNodeGroup{
		defaultNodeGroup,
	}

	// tidb node setting
	tidbNodeSpecKey := data.TiDBNodeSetting.NodeSpecKey.ValueString()
	tidbNodeSetting := dedicated.V1beta1ClusterTidbNodeSetting{
		NodeSpecKey:    &tidbNodeSpecKey,
		TidbNodeGroups: nodeGroups,
	}

	// tikv node setting
	tikvNodeSpecKey := data.TiKVNodeSetting.NodeSpecKey.ValueString()
	tikvNodeCount := int32(data.TiKVNodeSetting.NodeCount.ValueInt32())
	tikvStorageSizeGi := int32(data.TiKVNodeSetting.StorageSizeGi.ValueInt32())
	tikvStorageType := dedicated.ClusterStorageNodeSettingStorageType(data.TiKVNodeSetting.StorageType.ValueString())
	tikvNodeSetting := dedicated.V1beta1ClusterStorageNodeSetting{
		NodeSpecKey:   tikvNodeSpecKey,
		NodeCount:     tikvNodeCount,
		StorageSizeGi: tikvStorageSizeGi,
		StorageType:   &tikvStorageType,
	}
	if IsKnown(data.TiKVNodeSetting.RaftStoreIOPS) {
		tikvRaftStoreIOPS := data.TiKVNodeSetting.RaftStoreIOPS.ValueInt32()
		tikvNodeSetting.RaftStoreIops = *dedicated.NewNullableInt32(&tikvRaftStoreIOPS)
	}

	// tiflash node setting
	var tiflashNodeSetting *dedicated.V1beta1ClusterStorageNodeSetting
	if data.TiFlashNodeSetting != nil {
		tiflashNodeSpecKey := data.TiFlashNodeSetting.NodeSpecKey.ValueString()
		tikvNodeCount := int32(data.TiKVNodeSetting.NodeCount.ValueInt32())
		tiflashStorageSizeGi := int32(data.TiFlashNodeSetting.StorageSizeGi.ValueInt32())
		tiflashStorageType := dedicated.ClusterStorageNodeSettingStorageType(data.TiFlashNodeSetting.StorageType.ValueString())
		tiflashNodeSetting = &dedicated.V1beta1ClusterStorageNodeSetting{
			NodeSpecKey:   tiflashNodeSpecKey,
			NodeCount:     tikvNodeCount,
			StorageSizeGi: tiflashStorageSizeGi,
			StorageType:   &tiflashStorageType,
		}
		if IsKnown(data.TiFlashNodeSetting.RaftStoreIOPS) {
			tiflashRaftStoreIOPS := data.TiFlashNodeSetting.RaftStoreIOPS.ValueInt32()
			tiflashNodeSetting.RaftStoreIops = *dedicated.NewNullableInt32(&tiflashRaftStoreIOPS)
		}
	}

	labels := make(map[string]string)
	if IsKnown(data.ProjectId) {
		labels[LabelsKeyProjectId] = data.ProjectId.ValueString()
	}

	return dedicated.TidbCloudOpenApidedicatedv1beta1Cluster{
		DisplayName:        displayName,
		RegionId:           regionId,
		Labels:             &labels,
		TidbNodeSetting:    tidbNodeSetting,
		TikvNodeSetting:    tikvNodeSetting,
		TiflashNodeSetting: tiflashNodeSetting,
		Port:               data.Port.ValueInt32(),
		RootPassword:       &rootPassword,
	}, nil
}

func updatePublicEndpointSetting(ctx context.Context, client tidbcloud.TiDBCloudDedicatedClient,
	clusterId string, nodeGroupId string, data *publicEndpointSetting) (*publicEndpointSetting, error) {
	if data == nil {
		return nil, nil
	}
	req := dedicated.TidbNodeGroupServiceUpdatePublicEndpointSettingRequest{}
	ipAccessList := make([]dedicated.V1beta1PublicEndpointSettingIpAccessList, 0)
	if IsKnown(data.IPAccessList) {
		for _, v := range data.IPAccessList.Elements() {
			obj := v.(types.Object)
			cidr := obj.Attributes()["cidr_notation"].(types.String)
			desc := obj.Attributes()["description"].(types.String)
			ipAccessList = append(ipAccessList, dedicated.V1beta1PublicEndpointSettingIpAccessList{
				CidrNotation: cidr.ValueString(),
				Description:  desc.ValueStringPointer(),
			})
		}
	}
	if IsKnown(data.Enabled) {
		enabled := data.Enabled.ValueBool()
		req.Enabled = *dedicated.NewNullableBool(&enabled)
	}
	req.IpAccessList = ipAccessList
	publicEndpoint, err := client.UpdatePublicEndpoint(ctx, clusterId, nodeGroupId, &req)
	if err != nil {
		return nil, err
	}
	return convertDedicatedPublicEndpointSetting(publicEndpoint), nil
}
