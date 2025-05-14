package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ datasource.DataSource = &dedicatedAuditLogConfigDataSource{}

type dedicatedAuditLogConfigDataSource struct {
	provider *tidbcloudProvider
}

type dedicatedAuditLogConfigDataSourceData struct {
	ClusterId        types.String      `tfsdk:"cluster_id"`
	Enabled          types.Bool        `tfsdk:"enabled"`
	BucketUri        types.String      `tfsdk:"bucket_uri"`
	BucketRegionId   types.String      `tfsdk:"bucket_region_id"`
	AWSRoleArn       types.String      `tfsdk:"aws_role_arn"`
	AzureSasToken    types.String      `tfsdk:"azure_sas_token"`
	BucketWriteCheck *bucketWriteCheck `tfsdk:"bucket_write_check"`
	BucketManager    types.String      `tfsdk:"bucket_manager"`
}

func NewDedicatedAuditLogConfigDataSource() datasource.DataSource {
	return &dedicatedAuditLogConfigDataSource{}
}

func (d *dedicatedAuditLogConfigDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dedicated_audit_log_config"
}

func (d *dedicatedAuditLogConfigDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	var ok bool
	if d.provider, ok = req.ProviderData.(*tidbcloudProvider); !ok {
		resp.Diagnostics.AddError(
			"Unexpected DataSource Configure Type",
			fmt.Sprintf("Expected *tidbcloudProvider, got: %T", req.ProviderData),
		)
	}
}

func (d *dedicatedAuditLogConfigDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Get the audit log configuration of a dedicated TiDB cluster.",
		Attributes: map[string]schema.Attribute{
			"cluster_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the cluster.",
				Required:            true,
			},
			"enabled": schema.BoolAttribute{
				MarkdownDescription: "Whether the audit log is enabled.",
				Computed:            true,
			},
			"bucket_uri": schema.StringAttribute{
				MarkdownDescription: "The URI of the bucket where audit logs are stored.",
				Computed:            true,
			},
			"bucket_region_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the bucket region.",
				Computed:            true,
			},
			"aws_role_arn": schema.StringAttribute{
				MarkdownDescription: "The ARN of the AWS role for bucket access.",
				Computed:            true,
			},
			"azure_sas_token": schema.StringAttribute{
				MarkdownDescription: "The SAS token for Azure bucket access.",
				Computed:            true,
			},
			"bucket_write_check": schema.SingleNestedAttribute{
				MarkdownDescription: "The result of the bucket write permission check.",
				Computed:            true,
				Attributes: map[string]schema.Attribute{
					"writable": schema.BoolAttribute{
						MarkdownDescription: "Whether the bucket is writable.",
						Computed:            true,
					},
					"error_reason": schema.StringAttribute{
						MarkdownDescription: "Error reason if the bucket is not writable.",
						Computed:            true,
					},
				},
			},
			"bucket_manager": schema.StringAttribute{
				MarkdownDescription: "The cloud provider managing the bucket (AWS/Azure).",
				Computed:            true,
			},
		},
	}
}

func (d *dedicatedAuditLogConfigDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data dedicatedAuditLogConfigDataSourceData
	diags := req.Config.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Trace(ctx, "reading audit log configuration")
	auditLogConfig, err := d.provider.DedicatedClient.GetAuditLogConfig(ctx, data.ClusterId.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Read Error", fmt.Sprintf("Unable to get audit log config, got error: %s", err))
		return
	}

	// Map API response to Terraform data model
	data.Enabled = types.BoolValue(*auditLogConfig.Enabled)
	data.BucketUri = types.StringValue(*auditLogConfig.BucketUri)
	data.BucketRegionId = types.StringValue(*auditLogConfig.BucketRegionId)
	if auditLogConfig.AwsRoleArn != nil {
		data.AWSRoleArn = types.StringValue(*auditLogConfig.AwsRoleArn)
	}
	if auditLogConfig.AzureSasToken != nil {
		data.AzureSasToken = types.StringValue(*auditLogConfig.AzureSasToken)
	}
	data.BucketManager = types.StringValue(string(*auditLogConfig.BucketManager))

	// Handle bucket_write_check
	if auditLogConfig.BucketWriteCheck != nil {
		data.BucketWriteCheck = &bucketWriteCheck{
			Writable:    types.BoolValue(*auditLogConfig.BucketWriteCheck.Writable),
			ErrorReason: types.StringValue(*auditLogConfig.BucketWriteCheck.ErrorReason),
		}
	}

	// Save data into Terraform state
	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}
