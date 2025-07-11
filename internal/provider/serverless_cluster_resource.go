package provider

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int32planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"github.com/tidbcloud/terraform-provider-tidbcloud/tidbcloud"
	clusterV1beta1 "github.com/tidbcloud/tidbcloud-cli/pkg/tidbcloud/v1beta1/serverless/cluster"
)

const (
	serverlessClusterCreateTimeout  = 180 * time.Second
	serverlessClusterCreateInterval = 2 * time.Second
)

type mutableField string

const (
	DisplayName            mutableField = "displayName"
	Labels                 mutableField = "labels"
	PublicEndpointDisabled mutableField = "endpoints.public.disabled"
	SpendingLimitMonthly   mutableField = "spendingLimit.monthly"
	AutomatedBackupPolicy  mutableField = "automatedBackupPolicy"
	AutoScaling            mutableField = "autoScaling"
)

const (
	LabelsKeyProjectId = "tidb.cloud/project"
)

type serverlessClusterResourceData struct {
	ProjectId             types.String           `tfsdk:"project_id"`
	ClusterId             types.String           `tfsdk:"cluster_id"`
	DisplayName           types.String           `tfsdk:"display_name"`
	Region                *region                `tfsdk:"region"`
	SpendingLimit         types.Object           `tfsdk:"spending_limit"`
	AutoScaling           types.Object           `tfsdk:"auto_scaling"`
	AutomatedBackupPolicy *automatedBackupPolicy `tfsdk:"automated_backup_policy"`
	Endpoints             *endpoints             `tfsdk:"endpoints"`
	EncryptionConfig      *encryptionConfig      `tfsdk:"encryption_config"`
	Version               types.String           `tfsdk:"version"`
	CreatedBy             types.String           `tfsdk:"created_by"`
	CreateTime            types.String           `tfsdk:"create_time"`
	UpdateTime            types.String           `tfsdk:"update_time"`
	UserPrefix            types.String           `tfsdk:"user_prefix"`
	State                 types.String           `tfsdk:"state"`
	Labels                types.Map              `tfsdk:"labels"`
	Annotations           types.Map              `tfsdk:"annotations"`
}

type region struct {
	Name          types.String `tfsdk:"name"`
	RegionId      types.String `tfsdk:"region_id"`
	CloudProvider types.String `tfsdk:"cloud_provider"`
	DisplayName   types.String `tfsdk:"display_name"`
}

type spendingLimit struct {
	Monthly types.Int32 `tfsdk:"monthly"`
}

var spendingLimitAttrTypes = map[string]attr.Type{
	"monthly": types.Int32Type,
}

type autoScaling struct {
	MinRCU types.Int64 `tfsdk:"min_rcu"`
	MaxRCU types.Int64 `tfsdk:"max_rcu"`
}

var autoScalingAttrTypes = map[string]attr.Type{
	"min_rcu": types.Int64Type,
	"max_rcu": types.Int64Type,
}

type automatedBackupPolicy struct {
	StartTime     types.String `tfsdk:"start_time"`
	RetentionDays types.Int32  `tfsdk:"retention_days"`
}

type endpoints struct {
	Public  *public  `tfsdk:"public"`
	Private *private `tfsdk:"private"`
}

type public struct {
	Host     types.String `tfsdk:"host"`
	Port     types.Int32  `tfsdk:"port"`
	Disabled types.Bool   `tfsdk:"disabled"`
}

type private struct {
	Host types.String `tfsdk:"host"`
	Port types.Int32  `tfsdk:"port"`
	AWS  *aws         `tfsdk:"aws"`
}

type aws struct {
	ServiceName      types.String `tfsdk:"service_name"`
	AvailabilityZone types.List   `tfsdk:"availability_zone"`
}

type encryptionConfig struct {
	EnhancedEncryptionEnabled types.Bool `tfsdk:"enhanced_encryption_enabled"`
}

type serverlessClusterResource struct {
	provider *tidbcloudProvider
}

func NewServerlessClusterResource() resource.Resource {
	return &serverlessClusterResource{}
}

func (r *serverlessClusterResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_serverless_cluster"
}

func (r *serverlessClusterResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *serverlessClusterResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "serverless cluster resource",
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
				MarkdownDescription: "The display name of the cluster.",
				Required:            true,
			},
			"region": schema.SingleNestedAttribute{
				MarkdownDescription: "The region of the cluster.",
				Required:            true,
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.UseStateForUnknown(),
				},
				Attributes: map[string]schema.Attribute{
					"name": schema.StringAttribute{
						MarkdownDescription: "The unique name of the region. The format is `regions/{region-id}`.",
						Required:            true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.RequiresReplace(),
						},
					},
					"region_id": schema.StringAttribute{
						MarkdownDescription: "The ID of the region.",
						Computed:            true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"cloud_provider": schema.StringAttribute{
						MarkdownDescription: "The cloud provider of the region.",
						Computed:            true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"display_name": schema.StringAttribute{
						MarkdownDescription: "The display name of the region.",
						Computed:            true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
				},
			},
			"spending_limit": schema.SingleNestedAttribute{
				MarkdownDescription: "The spending limit of the cluster.",
				Optional:            true,
				Computed:            true,
				Attributes: map[string]schema.Attribute{
					"monthly": schema.Int32Attribute{
						MarkdownDescription: "Maximum monthly spending limit in USD cents.",
						Optional:            true,
						Computed:            true,
					},
				},
			},
			"auto_scaling": schema.SingleNestedAttribute{
				MarkdownDescription: "The auto scaling config of the essential cluster.",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"min_rcu": schema.Int64Attribute{
						MarkdownDescription: "The minimum RCU (Request Capacity Unit) of the cluster.",
						Required:            true,
					},
					"max_rcu": schema.Int64Attribute{
						MarkdownDescription: "The maximum RCU (Request Capacity Unit) of the cluster.",
						Required:            true,
					},
				},
			},
			"automated_backup_policy": schema.SingleNestedAttribute{
				MarkdownDescription: "The automated backup policy of the cluster.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.UseStateForUnknown(),
				},
				Attributes: map[string]schema.Attribute{
					"start_time": schema.StringAttribute{
						MarkdownDescription: "The UTC time of day in HH:mm format when the automated backup will start.",
						Optional:            true,
						Computed:            true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"retention_days": schema.Int32Attribute{
						MarkdownDescription: "The number of days to retain automated backups.",
						Optional:            true,
						Computed:            true,
						PlanModifiers: []planmodifier.Int32{
							int32planmodifier.UseStateForUnknown(),
						},
					},
				},
			},
			"endpoints": schema.SingleNestedAttribute{
				MarkdownDescription: "The endpoints for connecting to the cluster.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.UseStateForUnknown(),
				},
				Attributes: map[string]schema.Attribute{
					"public": schema.SingleNestedAttribute{
						MarkdownDescription: "The public endpoint for connecting to the cluster.",
						Optional:            true,
						Computed:            true,
						Attributes: map[string]schema.Attribute{
							"host": schema.StringAttribute{
								MarkdownDescription: "The host of the public endpoint.",
								Computed:            true,
								PlanModifiers: []planmodifier.String{
									stringplanmodifier.UseStateForUnknown(),
								},
							},
							"port": schema.Int32Attribute{
								MarkdownDescription: "The port of the public endpoint.",
								Computed:            true,
								PlanModifiers: []planmodifier.Int32{
									int32planmodifier.UseStateForUnknown(),
								},
							},
							"disabled": schema.BoolAttribute{
								MarkdownDescription: "Whether the public endpoint is disabled.",
								Optional:            true,
								PlanModifiers: []planmodifier.Bool{
									boolplanmodifier.UseStateForUnknown(),
								},
							},
						},
					},
					"private": schema.SingleNestedAttribute{
						MarkdownDescription: "The private endpoint for connecting to the cluster.",
						Computed:            true,
						PlanModifiers: []planmodifier.Object{
							objectplanmodifier.UseStateForUnknown(),
						},
						Attributes: map[string]schema.Attribute{
							"host": schema.StringAttribute{
								MarkdownDescription: "The host of the private endpoint.",
								Computed:            true,
							},
							"port": schema.Int32Attribute{
								MarkdownDescription: "The port of the private endpoint.",
								Computed:            true,
							},
							"aws": schema.SingleNestedAttribute{
								MarkdownDescription: "Message for AWS PrivateLink information.",
								Computed:            true,
								Attributes: map[string]schema.Attribute{
									"service_name": schema.StringAttribute{
										MarkdownDescription: "The AWS service name for private access.",
										Computed:            true,
									},
									"availability_zone": schema.ListAttribute{
										MarkdownDescription: "The availability zones that the service is available in.",
										Computed:            true,
										ElementType:         types.StringType,
									},
								},
							},
						},
					},
				},
			},
			"encryption_config": schema.SingleNestedAttribute{
				MarkdownDescription: "The encryption settings for the cluster.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.UseStateForUnknown(),
				},
				Attributes: map[string]schema.Attribute{
					"enhanced_encryption_enabled": schema.BoolAttribute{
						MarkdownDescription: "Whether enhanced encryption is enabled.",
						Optional:            true,
						Computed:            true,
						PlanModifiers: []planmodifier.Bool{
							boolplanmodifier.RequiresReplace(),
							boolplanmodifier.UseStateForUnknown(),
						},
					},
				},
			},
			"version": schema.StringAttribute{
				MarkdownDescription: "The version of the cluster.",
				Computed:            true,
			},
			"created_by": schema.StringAttribute{
				MarkdownDescription: "The email of the creator of the cluster.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"create_time": schema.StringAttribute{
				MarkdownDescription: "The time the cluster was created.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"update_time": schema.StringAttribute{
				MarkdownDescription: "The time the cluster was last updated.",
				Computed:            true,
			},
			"user_prefix": schema.StringAttribute{
				MarkdownDescription: "The unique prefix in SQL user name.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"state": schema.StringAttribute{
				MarkdownDescription: "The state of the cluster.",
				Computed:            true,
			},
			"labels": schema.MapAttribute{
				MarkdownDescription: "The labels of the cluster.",
				Computed:            true,
				ElementType:         types.StringType,
			},
			"annotations": schema.MapAttribute{
				MarkdownDescription: "The annotations of the cluster.",
				Computed:            true,
				ElementType:         types.StringType,
			},
		},
	}
}

func (r serverlessClusterResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if !r.provider.configured {
		resp.Diagnostics.AddError(
			"Provider not configured",
			"The provider hasn't been configured before apply, likely because it depends on an unknown value from another resource. This leads to weird stuff happening, so we'd prefer if you didn't do that. Thanks!",
		)
		return
	}

	// get data from config
	var data serverlessClusterResourceData
	diags := req.Config.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Trace(ctx, "create serverless_cluster_resource")
	body, err := buildCreateServerlessClusterBody(ctx, data)
	if err != nil {
		resp.Diagnostics.AddError("Create Error", fmt.Sprintf("Unable to build CreateCluster body, got error: %s", err))
		return
	}
	cluster, err := r.provider.ServerlessClient.CreateCluster(ctx, &body)
	if err != nil {
		resp.Diagnostics.AddError("Create Error", fmt.Sprintf("Unable to call CreateCluster, got error: %s", err))
		return
	}

	clusterId := *cluster.ClusterId
	data.ClusterId = types.StringValue(clusterId)
	tflog.Info(ctx, "wait serverless cluster ready")
	cluster, err = WaitServerlessClusterReady(ctx, serverlessClusterCreateTimeout, serverlessClusterCreateInterval, clusterId, r.provider.ServerlessClient)
	if err != nil {
		resp.Diagnostics.AddError(
			"Cluster creation failed",
			fmt.Sprintf("Cluster is not ready, get error: %s", err),
		)
		return
	}
	cluster, err = r.provider.ServerlessClient.GetCluster(ctx, *cluster.ClusterId, clusterV1beta1.SERVERLESSSERVICEGETCLUSTERVIEWPARAMETER_FULL)
	if err != nil {
		tflog.Error(ctx, fmt.Sprintf("Unable to call GetCluster, error: %s", err))
		return
	}
	err = refreshServerlessClusterResourceData(ctx, cluster, &data)
	if err != nil {
		resp.Diagnostics.AddError("Refresh Error", fmt.Sprintf("Unable to refresh serverless cluster resource data, got error: %s", err))
		return
	}

	// save into the Terraform state.
	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (r serverlessClusterResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// get data from state
	var clusterId string

	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("cluster_id"), &clusterId)...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Debug(ctx, fmt.Sprintf("read serverless_cluster_resource cluster_id: %s", clusterId))

	// call read api
	tflog.Trace(ctx, "read serverless_cluster_resource")
	cluster, err := r.provider.ServerlessClient.GetCluster(ctx, clusterId, clusterV1beta1.SERVERLESSSERVICEGETCLUSTERVIEWPARAMETER_FULL)
	if err != nil {
		tflog.Error(ctx, fmt.Sprintf("Unable to call GetCluster, error: %s", err))
		return
	}
	var data serverlessClusterResourceData
	err = refreshServerlessClusterResourceData(ctx, cluster, &data)
	if err != nil {
		resp.Diagnostics.AddError("Refresh Error", fmt.Sprintf("Unable to refresh serverless cluster resource data, got error: %s", err))
		return
	}

	// save into the Terraform state.
	diags := resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (r serverlessClusterResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var clusterId string

	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("cluster_id"), &clusterId)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Trace(ctx, "delete serverless_cluster_resource")
	_, err := r.provider.ServerlessClient.DeleteCluster(ctx, clusterId)
	if err != nil {
		resp.Diagnostics.AddError("Delete Error", fmt.Sprintf("Unable to call DeleteCluster, got error: %s", err))
		return
	}
}

func (r serverlessClusterResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// get plan
	var plan serverlessClusterResourceData
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	// get state
	var state serverlessClusterResourceData
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := &clusterV1beta1.V1beta1ServerlessServicePartialUpdateClusterBody{
		Cluster: &clusterV1beta1.RequiredTheClusterToBeUpdated{},
	}

	if plan.DisplayName.ValueString() != state.DisplayName.ValueString() {
		displayName := plan.DisplayName.ValueString()
		body.Cluster.DisplayName = &displayName
		body.UpdateMask = string(DisplayName)
		tflog.Trace(ctx, fmt.Sprintf("update serverless_cluster_resource %s", DisplayName))
		_, err := r.provider.ServerlessClient.PartialUpdateCluster(ctx, state.ClusterId.ValueString(), body)
		if err != nil {
			resp.Diagnostics.AddError("Update Error", fmt.Sprintf("Unable to call UpdateCluster, got error: %s", err))
			return
		}
	}

	if plan.Endpoints.Public.Disabled.ValueBool() != state.Endpoints.Public.Disabled.ValueBool() {
		publicEndpointDisabled := plan.Endpoints.Public.Disabled.ValueBool()
		body.Cluster.Endpoints = &clusterV1beta1.V1beta1ClusterEndpoints{
			Public: &clusterV1beta1.EndpointsPublic{
				Disabled: &publicEndpointDisabled,
			},
		}
		body.UpdateMask = string(PublicEndpointDisabled)
		tflog.Trace(ctx, fmt.Sprintf("update serverless_cluster_resource %s", PublicEndpointDisabled))
		_, err := r.provider.ServerlessClient.PartialUpdateCluster(ctx, state.ClusterId.ValueString(), body)
		if err != nil {
			resp.Diagnostics.AddError("Update Error", fmt.Sprintf("Unable to call UpdateCluster, got error: %s", err))
			return
		}
	}

	if IsKnown(plan.SpendingLimit) {
		if IsKnown(plan.AutoScaling) || IsKnown(state.AutoScaling) {
			resp.Diagnostics.AddError("Update Error", "Cannot set both spending limit and capacity for serverless cluster")
			return
		}
		var planLimit spendingLimit
		diags := plan.SpendingLimit.As(ctx, &planLimit, basetypes.ObjectAsOptions{})
		if diags.HasError() {
			resp.Diagnostics.Append(diags...)
			return
		}

		var stateLimit spendingLimit
		if IsKnown(state.SpendingLimit) {
			diags := state.SpendingLimit.As(ctx, &stateLimit, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				resp.Diagnostics.Append(diags...)
				return
			}
		}
		if planLimit.Monthly.ValueInt32() != stateLimit.Monthly.ValueInt32() {
			spendingLimitInt32 := planLimit.Monthly.ValueInt32()
			body.Cluster.SpendingLimit = &clusterV1beta1.ClusterSpendingLimit{
				Monthly: &spendingLimitInt32,
			}
			body.UpdateMask = string(SpendingLimitMonthly)
			tflog.Trace(ctx, fmt.Sprintf("update serverless_cluster_resource %s", SpendingLimitMonthly))

			_, err := r.provider.ServerlessClient.PartialUpdateCluster(ctx, state.ClusterId.ValueString(), body)
			if err != nil {
				resp.Diagnostics.AddError("Update Error", fmt.Sprintf("Unable to call UpdateCluster, got error: %s", err))
				return
			}
		}
	}

	if IsKnown(plan.AutoScaling) {
		if IsKnown(plan.SpendingLimit) || IsKnown(state.SpendingLimit) {
			resp.Diagnostics.AddError("Update Error", "Cannot set both spending limit and capacity for serverless cluster")
			return
		}
		var planCap autoScaling
		diags := plan.AutoScaling.As(ctx, &planCap, basetypes.ObjectAsOptions{})
		if diags.HasError() {
			resp.Diagnostics.Append(diags...)
			return
		}
		var stateCap autoScaling
		if IsKnown(state.AutoScaling) {
			diags := state.AutoScaling.As(ctx, &stateCap, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				resp.Diagnostics.Append(diags...)
				return
			}
		}
		if planCap.MinRCU.ValueInt64() != stateCap.MinRCU.ValueInt64() ||
			planCap.MaxRCU.ValueInt64() != stateCap.MaxRCU.ValueInt64() {
			minRCU := planCap.MinRCU.ValueInt64()
			maxRCU := planCap.MaxRCU.ValueInt64()
			body.Cluster.AutoScaling = &clusterV1beta1.V1beta1ClusterAutoScaling{
				MinRcu: &minRCU,
				MaxRcu: &maxRCU,
			}
			body.UpdateMask = string(AutoScaling)
			tflog.Trace(ctx, fmt.Sprintf("update serverless_cluster_resource %s", AutoScaling))
			_, err := r.provider.ServerlessClient.PartialUpdateCluster(ctx, state.ClusterId.ValueString(), body)
			if err != nil {
				resp.Diagnostics.AddError("Update Error", fmt.Sprintf("Unable to call UpdateCluster, got error: %s", err))
				return
			}
			// wait for the auto scaling to be updated
			time.Sleep(2 * time.Second)
		}
	}

	if plan.AutomatedBackupPolicy != nil {
		if plan.AutomatedBackupPolicy.StartTime.ValueString() != state.AutomatedBackupPolicy.StartTime.ValueString() ||
			plan.AutomatedBackupPolicy.RetentionDays.ValueInt32() != state.AutomatedBackupPolicy.RetentionDays.ValueInt32() {
			automatedBackupPolicyStartTime := plan.AutomatedBackupPolicy.StartTime.ValueString()
			automatedBackupPolicyRetentionDays := plan.AutomatedBackupPolicy.RetentionDays.ValueInt32()
			body.Cluster.AutomatedBackupPolicy = &clusterV1beta1.V1beta1ClusterAutomatedBackupPolicy{
				StartTime:     &automatedBackupPolicyStartTime,
				RetentionDays: &automatedBackupPolicyRetentionDays,
			}
			body.UpdateMask = string(AutomatedBackupPolicy)
			tflog.Trace(ctx, fmt.Sprintf("update serverless_cluster_resource %s", AutomatedBackupPolicy))
			_, err := r.provider.ServerlessClient.PartialUpdateCluster(ctx, state.ClusterId.ValueString(), body)
			if err != nil {
				resp.Diagnostics.AddError("Update Error", fmt.Sprintf("Unable to call UpdateCluster, got error: %s", err))
				return
			}
		}
	}

	// because the update api does not return the annotations, we need to call the get api
	cluster, err := r.provider.ServerlessClient.GetCluster(ctx, state.ClusterId.ValueString(), clusterV1beta1.SERVERLESSSERVICEGETCLUSTERVIEWPARAMETER_FULL)
	if err != nil {
		tflog.Error(ctx, fmt.Sprintf("Unable to call GetCluster, error: %s", err))
		return
	}
	err = refreshServerlessClusterResourceData(ctx, cluster, &state)
	if err != nil {
		resp.Diagnostics.AddError("Refresh Error", fmt.Sprintf("Unable to refresh serverless cluster resource data, got error: %s", err))
		return
	}

	// save into the Terraform state.
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

func (r serverlessClusterResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("cluster_id"), req, resp)
}

func buildCreateServerlessClusterBody(ctx context.Context, data serverlessClusterResourceData) (clusterV1beta1.TidbCloudOpenApiserverlessv1beta1Cluster, error) {
	displayName := data.DisplayName.ValueString()
	regionName := data.Region.Name.ValueString()
	labels := make(map[string]string)
	if IsKnown(data.ProjectId) {
		labels[LabelsKeyProjectId] = data.ProjectId.ValueString()
	}
	body := clusterV1beta1.TidbCloudOpenApiserverlessv1beta1Cluster{
		DisplayName: displayName,
		Region: clusterV1beta1.Commonv1beta1Region{
			Name: &regionName,
		},
		Labels: &labels,
	}

	if IsKnown(data.SpendingLimit) {
		if IsKnown(data.AutoScaling) {
			return clusterV1beta1.TidbCloudOpenApiserverlessv1beta1Cluster{},
				errors.New("cannot set both spending limit and capacity for serverless cluster")
		}
		var limit spendingLimit
		diags := data.SpendingLimit.As(ctx, &limit, basetypes.ObjectAsOptions{})
		if diags.HasError() {
			return clusterV1beta1.TidbCloudOpenApiserverlessv1beta1Cluster{}, errors.New("unable to convert spending limit")
		}
		spendingLimit := limit.Monthly.ValueInt32()
		body.SpendingLimit = &clusterV1beta1.ClusterSpendingLimit{
			Monthly: &spendingLimit,
		}
	}

	if IsKnown(data.AutoScaling) {
		var cap autoScaling
		diags := data.AutoScaling.As(ctx, &cap, basetypes.ObjectAsOptions{})
		if diags.HasError() {
			return clusterV1beta1.TidbCloudOpenApiserverlessv1beta1Cluster{}, errors.New("unable to convert capacity")
		}
		minRCU := cap.MinRCU.ValueInt64()
		maxRCU := cap.MaxRCU.ValueInt64()
		body.AutoScaling = &clusterV1beta1.V1beta1ClusterAutoScaling{
			MinRcu: &minRCU,
			MaxRcu: &maxRCU,
		}
	}

	if data.AutomatedBackupPolicy != nil {
		automatedBackupPolicy := data.AutomatedBackupPolicy
		automatedBackupPolicyStartTime := automatedBackupPolicy.StartTime.ValueString()
		automatedBackupPolicyRetentionDays := automatedBackupPolicy.RetentionDays.ValueInt32()
		body.AutomatedBackupPolicy = &clusterV1beta1.V1beta1ClusterAutomatedBackupPolicy{
			StartTime:     &automatedBackupPolicyStartTime,
			RetentionDays: &automatedBackupPolicyRetentionDays,
		}
	}

	if data.Endpoints != nil {
		publicEndpointsDisabled := data.Endpoints.Public.Disabled.ValueBool()
		body.Endpoints = &clusterV1beta1.V1beta1ClusterEndpoints{
			Public: &clusterV1beta1.EndpointsPublic{
				Disabled: &publicEndpointsDisabled,
			},
		}
	}

	if data.EncryptionConfig != nil {
		encryptionConfig := data.EncryptionConfig
		enhancedEncryptionEnabled := encryptionConfig.EnhancedEncryptionEnabled.ValueBool()
		body.EncryptionConfig = &clusterV1beta1.V1beta1ClusterEncryptionConfig{
			EnhancedEncryptionEnabled: &enhancedEncryptionEnabled,
		}
	}
	return body, nil
}

func refreshServerlessClusterResourceData(ctx context.Context, resp *clusterV1beta1.TidbCloudOpenApiserverlessv1beta1Cluster, data *serverlessClusterResourceData) error {
	labels, diags := types.MapValueFrom(ctx, types.StringType, *resp.Labels)
	if diags.HasError() {
		return errors.New("unable to convert labels")
	}
	annotations, diags := types.MapValueFrom(ctx, types.StringType, *resp.Annotations)
	if diags.HasError() {
		return errors.New("unable to convert annotations")
	}
	data.ClusterId = types.StringValue(*resp.ClusterId)
	data.DisplayName = types.StringValue(resp.DisplayName)
	data.ProjectId = types.StringValue((*resp.Labels)[LabelsKeyProjectId])

	r := resp.Region
	data.Region = &region{
		Name:          types.StringValue(*r.Name),
		RegionId:      types.StringValue(*r.RegionId),
		CloudProvider: types.StringValue(string(*r.CloudProvider)),
		DisplayName:   types.StringValue(*r.DisplayName),
	}

	if resp.SpendingLimit != nil {
		s := spendingLimit{
			Monthly: types.Int32Value(*resp.SpendingLimit.Monthly),
		}
		data.SpendingLimit, diags = types.ObjectValueFrom(ctx, spendingLimitAttrTypes, s)
		if diags.HasError() {
			return errors.New("unable to convert spending limit")
		}
	} else {
		data.SpendingLimit = types.ObjectNull(spendingLimitAttrTypes)
	}

	if resp.AutoScaling != nil {
		as := autoScaling{
			MinRCU: types.Int64Value(*resp.AutoScaling.MinRcu),
			MaxRCU: types.Int64Value(*resp.AutoScaling.MaxRcu),
		}
		data.AutoScaling, diags = types.ObjectValueFrom(ctx, autoScalingAttrTypes, as)
		if diags.HasError() {
			return errors.New("unable to convert auto scaling")
		}
	} else {
		data.AutoScaling = types.ObjectNull(autoScalingAttrTypes)
	}

	a := resp.AutomatedBackupPolicy
	data.AutomatedBackupPolicy = &automatedBackupPolicy{
		StartTime:     types.StringValue(*a.StartTime),
		RetentionDays: types.Int32Value(*a.RetentionDays),
	}

	e := resp.Endpoints
	var pe private
	if e.Private.Aws != nil {
		awsAvailabilityZone, diag := types.ListValueFrom(ctx, types.StringType, e.Private.Aws.AvailabilityZone)
		if diag.HasError() {
			return errors.New("unable to convert aws availability zone")
		}
		pe = private{
			Host: types.StringValue(*e.Private.Host),
			Port: types.Int32Value(*e.Private.Port),
			AWS: &aws{
				ServiceName:      types.StringValue(*e.Private.Aws.ServiceName),
				AvailabilityZone: awsAvailabilityZone,
			},
		}
	}

	data.Endpoints = &endpoints{
		Public: &public{
			Host:     types.StringValue(*e.Public.Host),
			Port:     types.Int32Value(*e.Public.Port),
			Disabled: types.BoolValue(*e.Public.Disabled),
		},
		Private: &pe,
	}

	en := resp.EncryptionConfig
	data.EncryptionConfig = &encryptionConfig{
		EnhancedEncryptionEnabled: types.BoolValue(*en.EnhancedEncryptionEnabled),
	}

	data.Version = types.StringValue(*resp.Version)
	data.CreatedBy = types.StringValue(*resp.CreatedBy)
	data.CreateTime = types.StringValue(resp.CreateTime.Format(time.RFC3339))
	data.UpdateTime = types.StringValue(resp.UpdateTime.Format(time.RFC3339))
	data.UserPrefix = types.StringValue(*resp.UserPrefix)
	data.State = types.StringValue(string(*resp.State))
	data.Labels = labels
	data.Annotations = annotations
	return nil
}

func WaitServerlessClusterReady(ctx context.Context, timeout time.Duration, interval time.Duration, clusterId string,
	client tidbcloud.TiDBCloudServerlessClient) (*clusterV1beta1.TidbCloudOpenApiserverlessv1beta1Cluster, error) {
	stateConf := &retry.StateChangeConf{
		Pending: []string{
			string(clusterV1beta1.COMMONV1BETA1CLUSTERSTATE_CREATING),
			string(clusterV1beta1.COMMONV1BETA1CLUSTERSTATE_RESTORING),
		},
		Target: []string{
			string(clusterV1beta1.COMMONV1BETA1CLUSTERSTATE_ACTIVE),
			string(clusterV1beta1.COMMONV1BETA1CLUSTERSTATE_DELETED),
		},
		Timeout:      timeout,
		MinTimeout:   500 * time.Millisecond,
		PollInterval: interval,
		Refresh:      serverlessClusterStateRefreshFunc(ctx, clusterId, client),
	}

	outputRaw, err := stateConf.WaitForStateContext(ctx)

	if output, ok := outputRaw.(*clusterV1beta1.TidbCloudOpenApiserverlessv1beta1Cluster); ok {
		return output, err
	}
	return nil, err
}

func serverlessClusterStateRefreshFunc(ctx context.Context, clusterId string,
	client tidbcloud.TiDBCloudServerlessClient) retry.StateRefreshFunc {
	return func() (interface{}, string, error) {
		tflog.Trace(ctx, fmt.Sprintf("Waiting for serverless cluster %s ready", clusterId))
		cluster, err := client.GetCluster(ctx, clusterId, clusterV1beta1.SERVERLESSSERVICEGETCLUSTERVIEWPARAMETER_BASIC)
		if err != nil {
			return nil, "", err
		}
		return cluster, string(*cluster.State), nil
	}
}
