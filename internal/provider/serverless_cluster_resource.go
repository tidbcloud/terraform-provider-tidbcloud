package provider

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"github.com/tidbcloud/terraform-provider-tidbcloud/tidbcloud"
	clusterV1beta1 "github.com/tidbcloud/tidbcloud-cli/pkg/tidbcloud/v1beta1/serverless/cluster"
)

type mutableField string

const (
	DisplayName                   mutableField = "displayName"
	Labels                        mutableField = "labels"
	PublicEndpointDisabled        mutableField = "endpoints.public.disabled"
	SpendingLimitMonthly          mutableField = "spendingLimit.monthly"
	AutomatedBackupPolicySchedule mutableField = "automatedBackupPolicy.schedule"
)

type updatableField string

const (
	UpdateDisplayName updatableField = "display_name"
)

const (
	LabelsKeyProjectId = "tidb.cloud/project"
)

type serverlessClusterResourceData struct {
	ProjectId             types.String           `tfsdk:"project_id"`
	ClusterId             types.String           `tfsdk:"cluster_id"`
	DisplayName           types.String           `tfsdk:"display_name"`
	Region                *region                `tfsdk:"region"`
	SpendingLimit         *spendingLimit         `tfsdk:"spending_limit"`
	AutomatedBackupPolicy *automatedBackupPolicy `tfsdk:"automated_backup_policy"`
	Endpoints             *endpoints             `tfsdk:"endpoints"`
	EncryptionConfig      *encryptionConfig      `tfsdk:"encryption_config"`
	Version               types.String           `tfsdk:"version"`
	CreatedBy             types.String           `tfsdk:"created_by"`
	CreateTime            types.String           `tfsdk:"create_time"`
	UpdateTime            types.String           `tfsdk:"update_time"`
	UserPrefix            types.String           `tfsdk:"user_prefix"`
	State                 types.String           `tfsdk:"state"`
	Usage                 *usage                 `tfsdk:"usage"`
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
	Monthly types.Int64 `tfsdk:"monthly"`
}

type automatedBackupPolicy struct {
	StartTime     types.String `tfsdk:"start_time"`
	RetentionDays types.Int64  `tfsdk:"retention_days"`
}

type endpoints struct {
	PublicEndpoint  *publicEndpoint  `tfsdk:"public_endpoint"`
	PrivateEndpoint *privateEndpoint `tfsdk:"private_endpoint"`
}

type publicEndpoint struct {
	Host     types.String `tfsdk:"host"`
	Port     types.Int64  `tfsdk:"port"`
	Disabled types.Bool   `tfsdk:"disabled"`
}

type privateEndpoint struct {
	Host        types.String `tfsdk:"host"`
	Port        types.Int64  `tfsdk:"port"`
	AWSEndpoint *awsEndpoint `tfsdk:"aws_endpoint"`
	GCPEndpoint *gcpEndpoint `tfsdk:"gcp_endpoint"`
}

type awsEndpoint struct {
	ServiceName      types.String `tfsdk:"service_name"`
	AvailabilityZone types.List   `tfsdk:"availability_zone"`
}

type gcpEndpoint struct {
	ServiceAttachmentName types.String `tfsdk:"service_attachment_name"`
}

type encryptionConfig struct {
	EnhancedEncryptionEnabled types.Bool `tfsdk:"enhanced_encryption_enabled"`
}

type usage struct {
	RequestUnit     types.String `tfsdk:"request_unit"`
	RowBasedStorage types.Int64  `tfsdk:"row_based_storage"`
	ColumnarStorage types.Int64  `tfsdk:"columnar_storage"`
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
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.UseStateForUnknown(),
				},
				Attributes: map[string]schema.Attribute{
					"monthly": schema.Int64Attribute{
						MarkdownDescription: "Maximum monthly spending limit in USD cents.",
						Optional:            true,
						Computed:            true,
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
					"retention_days": schema.Int64Attribute{
						MarkdownDescription: "The number of days to retain automated backups.",
						Computed:            true,
						PlanModifiers: []planmodifier.Int64{
							int64planmodifier.UseStateForUnknown(),
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
					"public_endpoint": schema.SingleNestedAttribute{
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
							"port": schema.Int64Attribute{
								MarkdownDescription: "The port of the public endpoint.",
								Computed:            true,
								PlanModifiers: []planmodifier.Int64{
									int64planmodifier.UseStateForUnknown(),
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
					"private_endpoint": schema.SingleNestedAttribute{
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
							"port": schema.Int64Attribute{
								MarkdownDescription: "The port of the private endpoint.",
								Computed:            true,
							},
							"aws_endpoint": schema.SingleNestedAttribute{
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
							"gcp_endpoint": schema.SingleNestedAttribute{
								MarkdownDescription: "Message for GCP PrivateLink information.",
								Computed:            true,
								Attributes: map[string]schema.Attribute{
									"service_attachment_name": schema.StringAttribute{
										MarkdownDescription: "The target GCP service attachment name for private access.",
										Computed:            true,
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
						},
					},
				},
			},
			"version": schema.StringAttribute{
				MarkdownDescription: "The version of the cluster.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
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
			"usage": schema.SingleNestedAttribute{
				MarkdownDescription: "The usage of the cluster.",
				Computed:            true,
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.UseStateForUnknown(),
				},
				Attributes: map[string]schema.Attribute{
					"request_unit": schema.StringAttribute{
						MarkdownDescription: "The request unit of the cluster.",
						Computed:            true,
					},
					"row_based_storage": schema.Int64Attribute{
						MarkdownDescription: "The row-based storage of the cluster.",
						Computed:            true,
					},
					"columnar_storage": schema.Int64Attribute{
						MarkdownDescription: "The columnar storage of the cluster.",
						Computed:            true,
					},
				},
			},
			"labels": schema.MapAttribute{
				MarkdownDescription: "The labels of the cluster.",
				Computed:            true,
				ElementType:         types.StringType,
				PlanModifiers: []planmodifier.Map{
					mapplanmodifier.UseStateForUnknown(),
				},
			},
			"annotations": schema.MapAttribute{
				MarkdownDescription: "The annotations of the cluster.",
				Computed:            true,
				ElementType:         types.StringType,
				PlanModifiers: []planmodifier.Map{
					mapplanmodifier.UseStateForUnknown(),
				},
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
	body, err := buildCreateServerlessClusterBody(data)
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
	cluster, err = WaitServerlessClusterReady(ctx, clusterServerlessCreateTimeout, clusterServerlessCreateInterval, clusterId, r.provider.ServerlessClient)
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
	var data serverlessClusterResourceData
	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Debug(ctx, fmt.Sprintf("read serverless_cluster_resource clusterid: %s", data.ClusterId.ValueString()))

	// call read api
	tflog.Trace(ctx, "read serverless_cluster_resource")
	cluster, err := r.provider.ServerlessClient.GetCluster(ctx, data.ClusterId.ValueString(), clusterV1beta1.SERVERLESSSERVICEGETCLUSTERVIEWPARAMETER_FULL)
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

	var fieldName string
	body := &clusterV1beta1.V1beta1ServerlessServicePartialUpdateClusterBody{
		Cluster: &clusterV1beta1.RequiredTheClusterToBeUpdated{},
	}

	if plan.DisplayName.ValueString() != state.DisplayName.ValueString() {
		displayName := plan.DisplayName.ValueString()
		body.Cluster.DisplayName = &displayName
		fieldName = string(DisplayName)
	}

	if plan.Endpoints.PublicEndpoint.Disabled.ValueBool() != state.Endpoints.PublicEndpoint.Disabled.ValueBool() {
		if fieldName != "" {
			resp.Diagnostics.AddError("Update Error", fmt.Sprintf("Unable to change more than one filed at the same time: %s and %s is changed", fieldName, string(PublicEndpointDisabled)))
			return
		}
		publicEndpointDisabled := plan.Endpoints.PublicEndpoint.Disabled.ValueBool()
		body.Cluster.Endpoints = &clusterV1beta1.V1beta1ClusterEndpoints{
			Public: &clusterV1beta1.EndpointsPublic{
				Disabled: &publicEndpointDisabled,
			},
		}
		fieldName = string(PublicEndpointDisabled)
	}

	if plan.SpendingLimit != nil {
		if plan.SpendingLimit.Monthly.ValueInt64() != state.SpendingLimit.Monthly.ValueInt64() {
			if fieldName != "" {
				resp.Diagnostics.AddError("Update Error", fmt.Sprintf("Unable to change %s and %s at the same time", fieldName, string(SpendingLimitMonthly)))
				return
			}
			spendingLimit := plan.SpendingLimit.Monthly.ValueInt64()
			spendingLimitInt32 := int32(spendingLimit)
			body.Cluster.SpendingLimit = &clusterV1beta1.ClusterSpendingLimit{
				Monthly: &spendingLimitInt32,
			}
			fieldName = string(SpendingLimitMonthly)
		}
	}

	if plan.AutomatedBackupPolicy != nil {
		if plan.AutomatedBackupPolicy.StartTime.ValueString() != state.AutomatedBackupPolicy.StartTime.ValueString() {
			if fieldName != "" {
				resp.Diagnostics.AddError("Update Error", fmt.Sprintf("Unable to change %s and %s at the same time", fieldName, string(AutomatedBackupPolicySchedule)))
				return
			}
			automatedBackupPolicyStartTime := plan.AutomatedBackupPolicy.StartTime.ValueString()
			body.Cluster.AutomatedBackupPolicy = &clusterV1beta1.V1beta1ClusterAutomatedBackupPolicy{
				StartTime: &automatedBackupPolicyStartTime,
			}
			fieldName = string(AutomatedBackupPolicySchedule)
		}
	}

	if fieldName == "" {
		tflog.Debug(ctx, "no field need to update")
		return
	}

	body.UpdateMask = fieldName
	// call update api
	tflog.Trace(ctx, "update serverless_cluster_resource")
	_, err := r.provider.ServerlessClient.PartialUpdateCluster(ctx, state.ClusterId.ValueString(), body)
	if err != nil {
		resp.Diagnostics.AddError("Update Error", fmt.Sprintf("Unable to call UpdateCluster, got error: %s", err))
		return
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

func buildCreateServerlessClusterBody(data serverlessClusterResourceData) (clusterV1beta1.TidbCloudOpenApiserverlessv1beta1Cluster, error) {
	displayName := data.DisplayName.ValueString()
	regionName := data.Region.Name.ValueString()
	labels := make(map[string]string)
	if !data.ProjectId.IsUnknown() && !data.ProjectId.IsNull() {
		labels[LabelsKeyProjectId] = data.ProjectId.ValueString()
	}
	body := clusterV1beta1.TidbCloudOpenApiserverlessv1beta1Cluster{
		DisplayName: displayName,
		Region: clusterV1beta1.Commonv1beta1Region{
			Name: &regionName,
		},
		Labels: &labels,
	}

	if data.SpendingLimit != nil {
		spendingLimit := data.SpendingLimit.Monthly.ValueInt64()
		spendingLimitInt32 := int32(spendingLimit)
		body.SpendingLimit = &clusterV1beta1.ClusterSpendingLimit{
			Monthly: &spendingLimitInt32,
		}
	}

	if data.AutomatedBackupPolicy != nil {
		automatedBackupPolicy := data.AutomatedBackupPolicy
		automatedBackupPolicyStartTime := automatedBackupPolicy.StartTime.ValueString()
		automatedBackupPolicyRetentionDays := automatedBackupPolicy.RetentionDays.ValueInt64()
		automatedBackupPolicyRetentionDaysInt32 := int32(automatedBackupPolicyRetentionDays)
		body.AutomatedBackupPolicy = &clusterV1beta1.V1beta1ClusterAutomatedBackupPolicy{
			StartTime:     &automatedBackupPolicyStartTime,
			RetentionDays: &automatedBackupPolicyRetentionDaysInt32,
		}
	}

	if data.Endpoints != nil {
		publicEndpointsDisabled := data.Endpoints.PublicEndpoint.Disabled.ValueBool()
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

	s := resp.SpendingLimit
	data.SpendingLimit = &spendingLimit{
		Monthly: types.Int64Value(int64(*s.Monthly)),
	}

	a := resp.AutomatedBackupPolicy
	data.AutomatedBackupPolicy = &automatedBackupPolicy{
		StartTime:     types.StringValue(*a.StartTime),
		RetentionDays: types.Int64Value(int64(*a.RetentionDays)),
	}

	e := resp.Endpoints
	var pe privateEndpoint
	if e.Private.Aws != nil {
		awsAvailabilityZone, diag := types.ListValueFrom(ctx, types.StringType, e.Private.Aws.AvailabilityZone)
		if diag.HasError() {
			return errors.New("unable to convert aws availability zone")
		}
		pe = privateEndpoint{
			Host: types.StringValue(*e.Private.Host),
			Port: types.Int64Value(int64(*e.Private.Port)),
			AWSEndpoint: &awsEndpoint{
				ServiceName:      types.StringValue(*e.Private.Aws.ServiceName),
				AvailabilityZone: awsAvailabilityZone,
			},
		}
	}

	if e.Private.Gcp != nil {
		pe = privateEndpoint{
			Host: types.StringValue(*e.Private.Host),
			Port: types.Int64Value(int64(*e.Private.Port)),
			GCPEndpoint: &gcpEndpoint{
				ServiceAttachmentName: types.StringValue(*e.Private.Gcp.ServiceAttachmentName),
			},
		}
	}

	data.Endpoints = &endpoints{
		PublicEndpoint: &publicEndpoint{
			Host:     types.StringValue(*e.Public.Host),
			Port:     types.Int64Value(int64(*e.Public.Port)),
			Disabled: types.BoolValue(*e.Public.Disabled),
		},
		PrivateEndpoint: &pe,
	}

	en := resp.EncryptionConfig
	data.EncryptionConfig = &encryptionConfig{
		EnhancedEncryptionEnabled: types.BoolValue(*en.EnhancedEncryptionEnabled),
	}

	data.Version = types.StringValue(*resp.Version)
	data.CreatedBy = types.StringValue(*resp.CreatedBy)
	data.CreateTime = types.StringValue(resp.CreateTime.String())
	data.UpdateTime = types.StringValue(resp.UpdateTime.String())
	data.UserPrefix = types.StringValue(*resp.UserPrefix)
	data.State = types.StringValue(string(*resp.State))

	u := resp.Usage
	data.Usage = &usage{
		RequestUnit:     types.StringValue(*u.RequestUnit),
		RowBasedStorage: types.Int64Value(int64(*u.RowBasedStorage)),
		ColumnarStorage: types.Int64Value(int64(*u.ColumnarStorage)),
	}

	data.Labels = labels
	data.Annotations = annotations
	return nil
}

func WaitServerlessClusterReady(ctx context.Context, timeout time.Duration, interval time.Duration, clusterId string,
	client tidbcloud.TiDBCloudServerlessClient) (*clusterV1beta1.TidbCloudOpenApiserverlessv1beta1Cluster, error) {
	stateConf := &retry.StateChangeConf{
		Pending: []string{
			string(clusterV1beta1.COMMONV1BETA1CLUSTERSTATE_CREATING),
			string(clusterV1beta1.COMMONV1BETA1CLUSTERSTATE_DELETING),
			string(clusterV1beta1.COMMONV1BETA1CLUSTERSTATE_RESTORING),
			string(clusterV1beta1.COMMONV1BETA1CLUSTERSTATE_INACTIVE),
			string(clusterV1beta1.COMMONV1BETA1CLUSTERSTATE_UPGRADING),
			string(clusterV1beta1.COMMONV1BETA1CLUSTERSTATE_IMPORTING),
			string(clusterV1beta1.COMMONV1BETA1CLUSTERSTATE_MODIFYING),
			string(clusterV1beta1.COMMONV1BETA1CLUSTERSTATE_PAUSING),
			string(clusterV1beta1.COMMONV1BETA1CLUSTERSTATE_RESUMING),
		},
		Target: []string{
			string(clusterV1beta1.COMMONV1BETA1CLUSTERSTATE_ACTIVE),
			string(clusterV1beta1.COMMONV1BETA1CLUSTERSTATE_PAUSED),
			string(clusterV1beta1.COMMONV1BETA1CLUSTERSTATE_DELETED),
			string(clusterV1beta1.COMMONV1BETA1CLUSTERSTATE_MAINTENANCE),
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
		tflog.Trace(ctx, "Waiting for serverless cluster ready")
		cluster, err := client.GetCluster(ctx, clusterId, clusterV1beta1.SERVERLESSSERVICEGETCLUSTERVIEWPARAMETER_BASIC)
		if err != nil {
			return nil, "", err
		}
		return cluster, string(*cluster.State), nil
	}
}
