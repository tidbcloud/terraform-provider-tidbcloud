package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/tidbcloud/tidbcloud-cli/pkg/tidbcloud/v1beta1/dedicated"
)

var (
	_ resource.Resource = &DedicatedAuditLogConfigResource{}
)

type DedicatedAuditLogConfigResource struct {
	provider *tidbcloudProvider
}

type dedicatedAuditLogConfigResourceData struct {
	ClusterId        types.String      `tfsdk:"cluster_id"`
	Enabled          types.Bool        `tfsdk:"enabled"`
	BucketUri        types.String      `tfsdk:"bucket_uri"`
	BucketRegionId   types.String      `tfsdk:"bucket_region_id"`
	AWSRoleArn       types.String      `tfsdk:"aws_role_arn"`
	AzureSasToken    types.String      `tfsdk:"azure_sas_token"`
	BucketWriteCheck *bucketWriteCheck `tfsdk:"bucket_write_check"`
}

type bucketWriteCheck struct {
	Writable    types.Bool   `tfsdk:"writable"`
	ErrorReason types.String `tfsdk:"error_reason"`
}

func NewDedicatedAuditLogConfigResource() resource.Resource {
	return &DedicatedAuditLogConfigResource{}
}

func (r *DedicatedAuditLogConfigResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dedicated_audit_log_config"
}

func (r *DedicatedAuditLogConfigResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "dedicated audit log config",
		Attributes: map[string]schema.Attribute{
			"cluster_id": schema.StringAttribute{
				Description: "The ID of the cluster",
				Required:    true,
			},
			"enabled": schema.BoolAttribute{
				Description: "Whether the audit log is enabled",
				Required:    true,
			},
			"bucket_uri": schema.StringAttribute{
				Description: "The URI of the bucket",
				Required:    true,
			},
			"bucket_region_id": schema.StringAttribute{
				Description: "The ID of the bucket region",
				Required:    true,
			},
			"aws_role_arn": schema.StringAttribute{
				Description: "The ARN of the AWS role",
				Optional:    true,
			},
			"azure_sas_token": schema.StringAttribute{
				Description: "The SAS token of the Azure",
				Optional:    true,
			},
			"bucket_write_check": schema.SingleNestedAttribute{
				Description: "The bucket write check",
				Computed:    true,
				Attributes: map[string]schema.Attribute{
					"writable": schema.BoolAttribute{
						Description: "Whether the bucket is writable",
						Computed:    true,
					},
					"error_reason": schema.StringAttribute{
						Description: "The error reason",
						Computed:    true,
					},
				},
			},
		},
	}
}

func (r *DedicatedAuditLogConfigResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	var ok bool
	r.provider, ok = req.ProviderData.(*tidbcloudProvider)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *tidbcloudProvider, got: %T", req.ProviderData),
		)
	}
}

func (r *DedicatedAuditLogConfigResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if !r.provider.configured {
		resp.Diagnostics.AddError(
			"Provider not configured",
			"The provider hasn't been configured before apply, likely because it depends on an unknown value from another resource. This leads to weird stuff happening, so we'd prefer if you didn't do that. Thanks!",
		)
		return
	}

	var data dedicatedAuditLogConfigResourceData
	diags := req.Plan.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Trace(ctx, "create dedicated_audit_log_config_resource")
	body, err := buildCreateDedicatedAuditLogConfigBody(data)
	if err != nil {
		resp.Diagnostics.AddError("Create Error", fmt.Sprintf("Unable to build create body, got error: %s", err))
		return
	}
	AuditLogConfig, err := r.provider.DedicatedClient.CreateAuditLogConfig(ctx, data.ClusterId.ValueString(), &body)
	if err != nil {
		resp.Diagnostics.AddError("Create Error", fmt.Sprintf("Unable to call CreateAuditLogConfig, got error: %s", err))
		return
	}

	refreshDedicatedAuditLogConfigResourceData(AuditLogConfig, &data)

	// save into the Terraform state.
	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (r *DedicatedAuditLogConfigResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data dedicatedAuditLogConfigResourceData
	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, fmt.Sprintf("read dedicated_audit_log_config_resource cluster_id: %s", data.ClusterId.ValueString()))

	// call read api
	tflog.Trace(ctx, "read dedicated_audit_log_config_resource")
	AuditLogConfig, err := r.provider.DedicatedClient.GetAuditLogConfig(ctx, data.ClusterId.ValueString())
	if err != nil {
		tflog.Error(ctx, fmt.Sprintf("Unable to call GetAuditLogConfig, error: %s", err))
		return
	}
	refreshDedicatedAuditLogConfigResourceData(AuditLogConfig, &data)

	// save into the Terraform state.
	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (r *DedicatedAuditLogConfigResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// get plan
	var plan dedicatedAuditLogConfigResourceData
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	// get state
	var state dedicatedAuditLogConfigResourceData
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := dedicated.DatabaseAuditLogServiceUpdateAuditLogConfigRequest{}
	if plan.Enabled.ValueBool() != state.Enabled.ValueBool() {
		enabled := plan.Enabled.ValueBool()
		body.Enabled = *dedicated.NewNullableBool(&enabled)
	}
	if plan.BucketUri.ValueString() != state.BucketUri.ValueString() {
		bucketUri := plan.BucketUri.ValueString()
		body.BucketUri = &bucketUri
	}
	if plan.BucketRegionId.ValueString() != state.BucketRegionId.ValueString() {
		bucketRegionId := plan.BucketRegionId.ValueString()
		body.BucketRegionId = &bucketRegionId
	}
	if plan.AWSRoleArn.ValueString() != state.AWSRoleArn.ValueString() {
		awsRoleArn := plan.AWSRoleArn.ValueString()
		body.AwsRoleArn = &awsRoleArn
	}
	if plan.AzureSasToken.ValueString() != state.AzureSasToken.ValueString() {
		azureSasToken := plan.AzureSasToken.ValueString()
		body.AzureSasToken = &azureSasToken
	}

	// call update api
	tflog.Trace(ctx, "update dedicated_audit_log_config_resource")
	auditLogConfig, err := r.provider.DedicatedClient.UpdateAuditLogConfig(ctx, state.ClusterId.ValueString(), &body)
	if err != nil {
		resp.Diagnostics.AddError("Update Error", fmt.Sprintf("Unable to call UpdateAuditLogConfig, got error: %s", err))
		return
	}

	refreshDedicatedAuditLogConfigResourceData(auditLogConfig, &state)

	// save into the Terraform state.
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)

	return
}

func (r *DedicatedAuditLogConfigResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	resp.Diagnostics.AddError("Delete Error", "Delete is not supported for dedicated audit log config, you can only disable it")
	return
}

func buildCreateDedicatedAuditLogConfigBody(data dedicatedAuditLogConfigResourceData) (dedicated.DatabaseAuditLogServiceCreateAuditLogConfigRequest, error) {
	enabled := data.Enabled.ValueBool()
	bucketUri := data.BucketUri.ValueString()
	bucketRegionId := data.BucketRegionId.ValueString()
	awsRoleArn := data.AWSRoleArn.ValueString()
	azureSasToken := data.AzureSasToken.ValueString()

	return dedicated.DatabaseAuditLogServiceCreateAuditLogConfigRequest{
		Enabled:        &enabled,
		BucketUri:      bucketUri,
		BucketRegionId: &bucketRegionId,
		AwsRoleArn:     &awsRoleArn,
		AzureSasToken:  &azureSasToken,
	}, nil
}

func refreshDedicatedAuditLogConfigResourceData(auditLogConfig *dedicated.Dedicatedv1beta1AuditLogConfig, data *dedicatedAuditLogConfigResourceData) {
	data.Enabled = types.BoolValue(*auditLogConfig.Enabled)
	data.BucketUri = types.StringValue(auditLogConfig.BucketUri)
	data.BucketRegionId = types.StringValue(*auditLogConfig.BucketRegionId)
	data.AWSRoleArn = types.StringValue(*auditLogConfig.AwsRoleArn)
	data.AzureSasToken = types.StringValue(*auditLogConfig.AzureSasToken)
	if auditLogConfig.BucketWriteCheck != nil {
		data.BucketWriteCheck = &bucketWriteCheck{
			Writable:    types.BoolValue(*auditLogConfig.BucketWriteCheck.Writable),
			ErrorReason: types.StringValue(*auditLogConfig.BucketWriteCheck.ErrorReason),
		}
	}
}
