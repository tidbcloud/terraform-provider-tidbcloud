package provider

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"github.com/tidbcloud/terraform-provider-tidbcloud/tidbcloud"
	clusterV1beta1 "github.com/tidbcloud/tidbcloud-cli/pkg/tidbcloud/v1beta1/serverless/cluster"
)

type serverlessSQLUserStatus string

const (
	serverlessSQLUserStatusCreating    serverlessSQLUserStatus = "CREATING"
	serverlessSQLUserStatusDeleting    serverlessSQLUserStatus = "DELETING"
	serverlessSQLUserStatusActive      serverlessSQLUserStatus = "ACTIVE"
	serverlessSQLUserStatusRestoring   serverlessSQLUserStatus = "RESTORING"
	serverlessSQLUserStatusMaintenance serverlessSQLUserStatus = "MAINTENANCE"
	serverlessSQLUserStatusDeleted     serverlessSQLUserStatus = "DELETED"
	serverlessSQLUserStatusInactive    serverlessSQLUserStatus = "INACTIVE"
	serverlessSQLUserStatusUpgrading   serverlessSQLUserStatus = "UPGRADING"
	serverlessSQLUserStatusImporting   serverlessSQLUserStatus = "IMPORTING"
	serverlessSQLUserStatusModifying   serverlessSQLUserStatus = "MODIFYING"
	serverlessSQLUserStatusPausing     serverlessSQLUserStatus = "PAUSING"
	serverlessSQLUserStatusPaused      serverlessSQLUserStatus = "PAUSED"
	serverlessSQLUserStatusResuming    serverlessSQLUserStatus = "RESUMING"
)

const (
	MYSQLNATIVEPASSWORD = "mysql_native_password"
)

type serverlessSQLUserResourceData struct {
	ClusterId   types.String `tfsdk:"cluster_id"`
	AuthMethod  types.String `tfsdk:"auth_method"`
	UserName    types.String `tfsdk:"user_name"`
	BuiltinRole types.String `tfsdk:"builtin_role"`
	CustomRoles types.List   `tfsdk:"custom_roles"`
}

type serverlessSQLUserResource struct {
	provider *tidbcloudProvider
}

func NewServerlessSQLUserResource() resource.Resource {
	return &serverlessSQLUserResource{}
}

func (r *serverlessSQLUserResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_serverless_sql_user"
}

func (r *serverlessSQLUserResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *serverlessSQLUserResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "serverless cluster resource",
		Attributes: map[string]schema.Attribute{
			"cluster_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the cluster.",
				Required:            true,
			},
			"auth_method": schema.StringAttribute{
				MarkdownDescription: "The authentication method of the user.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Default: stringdefault.StaticString(MYSQLNATIVEPASSWORD),
			},
			"user_name": schema.StringAttribute{
				MarkdownDescription: "The name of the user.",
				Required:            true,
			},
			"builtin_role": schema.StringAttribute{
				MarkdownDescription: "The builtin role of the user.",
				Required: 		  true,
			},
			"custom_roles": schema.ListAttribute{
				MarkdownDescription: "The custom roles of the user.",
				Optional:            true,
				ElementType:         types.StringType,
			},
		},
	}
}

func (r serverlessSQLUserResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if !r.provider.configured {
		resp.Diagnostics.AddError(
			"Provider not configured",
			"The provider hasn't been configured before apply, likely because it depends on an unknown value from another resource. This leads to weird stuff happening, so we'd prefer if you didn't do that. Thanks!",
		)
		return
	}

	// get data from config
	var data serverlessSQLUserResourceData
	diags := req.Config.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Trace(ctx, "create serverless_sql_user_resource")
	body, err := buildCreateServerlessSQLUserBody(ctx, data)
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
	cluster, err = WaitServerlessSQLUserReady(ctx, clusterServerlessCreateTimeout, clusterServerlessCreateInterval, clusterId, r.provider.ServerlessClient)
	if err != nil {
		resp.Diagnostics.AddError(
			"Cluster creation failed",
			fmt.Sprintf("Cluster is not ready, get error: %s", err),
		)
		return
	}
	refreshServerlessSQLUserResourceData(ctx, cluster, &data)

	// save into the Terraform state.
	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (r serverlessSQLUserResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// get data from state
	var data serverlessSQLUserResourceData
	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Debug(ctx, fmt.Sprintf("read serverless_sql_user_resource clusterid: %s", data.ClusterId.ValueString()))

	// call read api
	tflog.Trace(ctx, "read serverless_sql_user_resource")
	cluster, err := r.provider.ServerlessClient.GetCluster(ctx, data.ClusterId.ValueString(), clusterV1beta1.SERVERLESSSERVICEGETCLUSTERVIEWPARAMETER_FULL)
	if err != nil {
		tflog.Error(ctx, fmt.Sprintf("Unable to call GetCluster, error: %s", err))
		return
	}
	refreshServerlessSQLUserResourceData(ctx, cluster, &data)

	// save into the Terraform state.
	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (r serverlessSQLUserResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var clusterId string

	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("cluster_id"), &clusterId)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Trace(ctx, "delete serverless_sql_user_resource")
	_, err := r.provider.ServerlessClient.DeleteCluster(ctx, clusterId)
	if err != nil {
		resp.Diagnostics.AddError("Delete Error", fmt.Sprintf("Unable to call DeleteCluster, got error: %s", err))
		return
	}
}

func (r serverlessSQLUserResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// get plan
	var plan serverlessSQLUserResourceData
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	// get state
	var state serverlessSQLUserResourceData
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
			resp.Diagnostics.AddError("Update Error", fmt.Sprintf("Unable to change %s and %s at the same time", fieldName, string(PublicEndpointDisabled)))
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

	body.UpdateMask = fieldName
	// call update api
	tflog.Trace(ctx, "update serverless_sql_user_resource")
	_, err := r.provider.ServerlessClient.PartialUpdateCluster(ctx, state.ClusterId.ValueString(), body)
	if err != nil {
		resp.Diagnostics.AddError("Update Error", fmt.Sprintf("Unable to call UpdateCluster, got error: %s", err))
		return
	}

	tflog.Info(ctx, "wait cluster ready")
	cluster, err := WaitServerlessSQLUserReady(ctx, clusterUpdateTimeout, clusterUpdateInterval, state.ClusterId.ValueString(), r.provider.ServerlessClient)
	if err != nil {
		resp.Diagnostics.AddError(
			"Cluster update failed",
			fmt.Sprintf("Cluster is not ready, get error: %s", err),
		)
		return
	}
	refreshServerlessSQLUserResourceData(ctx, cluster, &state)

	// save into the Terraform state.
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)

}

func buildCreateServerlessSQLUserBody(ctx context.Context, data serverlessSQLUserResourceData) (clusterV1beta1.TidbCloudOpenApiserverlessv1beta1Cluster, error) {
	displayName := data.DisplayName.ValueString()
	regionName := data.Region.Name.ValueString()
	rootPassword := data.RootPassword.ValueString()
	highAvailabilityType := clusterV1beta1.ClusterHighAvailabilityType(data.HighAvailabilityType.ValueString())
	var labels map[string]string
	diag := data.Labels.ElementsAs(ctx, &labels, false)
	if diag.HasError() {
		return clusterV1beta1.TidbCloudOpenApiserverlessv1beta1Cluster{}, errors.New("unable to convert labels")
	}
	body := clusterV1beta1.TidbCloudOpenApiserverlessv1beta1Cluster{
		DisplayName: displayName,
		Region: clusterV1beta1.Commonv1beta1Region{
			Name: &regionName,
		},
		RootPassword:         &rootPassword,
		HighAvailabilityType: &highAvailabilityType,
		Labels:               &labels,
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

func refreshServerlessSQLUserResourceData(ctx context.Context, resp *clusterV1beta1.TidbCloudOpenApiserverlessv1beta1Cluster, data *serverlessSQLUserResourceData) {
	labels, diag := types.MapValueFrom(ctx, types.StringType, *resp.Labels)
	if diag.HasError() {
		return
	}
	annotations, diag := types.MapValueFrom(ctx, types.StringType, *resp.Annotations)
	if diag.HasError() {
		return
	}
	data.ClusterId = types.StringValue(*resp.ClusterId)
	data.DisplayName = types.StringValue(resp.DisplayName)

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
		awsAvailabilityZone, diags := types.ListValueFrom(ctx, types.StringType, e.Private.Aws.AvailabilityZone)
		if diags.HasError() {
			return
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
		PublicEndpoint: publicEndpoint{
			Host:     types.StringValue(*e.Public.Host),
			Port:     types.Int64Value(int64(*e.Public.Port)),
			Disabled: types.BoolValue(*e.Public.Disabled),
		},
		PrivateEndpoint: pe,
	}

	en := resp.EncryptionConfig
	data.EncryptionConfig = &encryptionConfig{
		EnhancedEncryptionEnabled: types.BoolValue(*en.EnhancedEncryptionEnabled),
	}

	data.HighAvailabilityType = types.StringValue(string(*resp.HighAvailabilityType))
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
}

func WaitServerlessSQLUserReady(ctx context.Context, timeout time.Duration, interval time.Duration, clusterId string,
	client tidbcloud.TiDBCloudServerlessClient) (*clusterV1beta1.TidbCloudOpenApiserverlessv1beta1Cluster, error) {
	stateConf := &retry.StateChangeConf{
		Pending: []string{
			string(serverlessSQLUserStatusCreating),
			string(serverlessSQLUserStatusDeleting),
			string(serverlessSQLUserStatusRestoring),
			string(serverlessSQLUserStatusInactive),
			string(serverlessSQLUserStatusUpgrading),
			string(serverlessSQLUserStatusImporting),
			string(serverlessSQLUserStatusModifying),
			string(serverlessSQLUserStatusPausing),
			string(serverlessSQLUserStatusResuming),
		},
		Target: []string{
			string(serverlessSQLUserStatusActive),
			string(serverlessSQLUserStatusPaused),
			string(serverlessSQLUserStatusDeleted),
			string(serverlessSQLUserStatusMaintenance),
		},
		Timeout:      timeout,
		MinTimeout:   500 * time.Millisecond,
		PollInterval: interval,
		Refresh:      serverlessSQLUserStateRefreshFunc(ctx, clusterId, client),
	}

	outputRaw, err := stateConf.WaitForStateContext(ctx)

	if output, ok := outputRaw.(*clusterV1beta1.TidbCloudOpenApiserverlessv1beta1Cluster); ok {
		return output, err
	}
	return nil, err
}

func serverlessSQLUserStateRefreshFunc(ctx context.Context, clusterId string,
	client tidbcloud.TiDBCloudServerlessClient) retry.StateRefreshFunc {
	return func() (interface{}, string, error) {
		tflog.Trace(ctx, "Waiting for serverless cluster ready")
		cluster, err := client.GetCluster(ctx, clusterId, clusterV1beta1.SERVERLESSSERVICEGETCLUSTERVIEWPARAMETER_FULL)
		if err != nil {
			return nil, "", err
		}
		return cluster, string(*cluster.State), nil
	}
}
