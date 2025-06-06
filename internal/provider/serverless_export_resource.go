package provider

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	exportV1beta1 "github.com/tidbcloud/tidbcloud-cli/pkg/tidbcloud/v1beta1/serverless/export"
)

type serverlessExportResourceData struct {
	ExportId      types.String   `tfsdk:"export_id"`
	ClusterId     types.String   `tfsdk:"cluster_id"`
	DisplayName   types.String   `tfsdk:"display_name"`
	State         types.String   `tfsdk:"state"`
	CreateTime    types.String   `tfsdk:"create_time"`
	CreatedBy     types.String   `tfsdk:"created_by"`
	UpdateTime    types.String   `tfsdk:"update_time"`
	CompleteTime  types.String   `tfsdk:"complete_time"`
	SnapshotTime  types.String   `tfsdk:"snapshot_time"`
	ExpireTime    types.String   `tfsdk:"expire_time"`
	ExportOptions *exportOptions `tfsdk:"export_options"`
	Target        *exportTarget  `tfsdk:"target"`
	Reason        types.String   `tfsdk:"reason"`
}

type exportOptions struct {
	FileType      types.String   `tfsdk:"file_type"`
	Compression   types.String   `tfsdk:"compression"`
	Filter        *exportFilter  `tfsdk:"filter"`
	CsvFormat     *csvFormat     `tfsdk:"csv_format"`
	ParquetFormat *parquetFormat `tfsdk:"parquet_format"`
}

type exportFilter struct {
	Sql   types.String `tfsdk:"sql"`
	Table *tableFilter `tfsdk:"table"`
}

type tableFilter struct {
	Patterns types.List   `tfsdk:"patterns"`
	Where    types.String `tfsdk:"where"`
}

type csvFormat struct {
	Separator  types.String `tfsdk:"separator"`
	Delimiter  types.String `tfsdk:"delimiter"`
	NullValue  types.String `tfsdk:"null_value"`
	SkipHeader types.Bool   `tfsdk:"skip_header"`
}

type parquetFormat struct {
	Compression types.String `tfsdk:"compression"`
}

type exportTarget struct {
	Type      types.String     `tfsdk:"type"`
	S3        *s3Target        `tfsdk:"s3"`
	Gcs       *gcsTarget       `tfsdk:"gcs"`
	AzureBlob *azureBlobTarget `tfsdk:"azure_blob"`
}

type s3Target struct {
	Uri       types.String `tfsdk:"uri"`
	AuthType  types.String `tfsdk:"auth_type"`
	AccessKey *accessKey   `tfsdk:"access_key"`
	RoleArn   types.String `tfsdk:"role_arn"`
}

type accessKey struct {
	Id     types.String `tfsdk:"id"`
	Secret types.String `tfsdk:"secret"`
}

type gcsTarget struct {
	Uri               types.String `tfsdk:"uri"`
	AuthType          types.String `tfsdk:"auth_type"`
	ServiceAccountKey types.String `tfsdk:"service_account_key"`
}

type azureBlobTarget struct {
	Uri      types.String `tfsdk:"uri"`
	AuthType types.String `tfsdk:"auth_type"`
	SasToken types.String `tfsdk:"sas_token"`
}
type serverlessExportResource struct {
	provider *tidbcloudProvider
}

func NewServerlessExportResource() resource.Resource {
	return &serverlessExportResource{}
}

func (r *serverlessExportResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_serverless_export"
}

func (r *serverlessExportResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *serverlessExportResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Serverless Export Resource",
		Attributes: map[string]schema.Attribute{
			"export_id": schema.StringAttribute{
				MarkdownDescription: "The unique ID of the export.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"cluster_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the cluster.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"display_name": schema.StringAttribute{
				MarkdownDescription: "The display name of the export.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
			},
			"state": schema.StringAttribute{
				MarkdownDescription: "The state of the export.",
				Computed:            true,
			},
			"create_time": schema.StringAttribute{
				MarkdownDescription: "Timestamp when the export was created.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"created_by": schema.StringAttribute{
				MarkdownDescription: "The user who created the export.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"update_time": schema.StringAttribute{
				MarkdownDescription: "Timestamp when the export was updated.",
				Computed:            true,
			},
			"complete_time": schema.StringAttribute{
				MarkdownDescription: "Timestamp when the export was completed.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"snapshot_time": schema.StringAttribute{
				MarkdownDescription: "Snapshot time of the export.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"expire_time": schema.StringAttribute{
				MarkdownDescription: "Expire time of the export.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"export_options": schema.SingleNestedAttribute{
				MarkdownDescription: "The options of the export.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.UseStateForUnknown(),
					objectplanmodifier.RequiresReplace(),
				},
				Attributes: map[string]schema.Attribute{
					"file_type": schema.StringAttribute{
						MarkdownDescription: "The exported file type. Available values are SQL, CSV and Parquet. Default is CSV.",
						Optional:            true,
						Computed:            true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"compression": schema.StringAttribute{
						MarkdownDescription: "The compression of the export.",
						Optional:            true,
						Computed:            true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"filter": schema.SingleNestedAttribute{
						MarkdownDescription: "The filter of the export.",
						Optional:            true,
						Computed:            true,
						PlanModifiers: []planmodifier.Object{
							objectplanmodifier.UseStateForUnknown(),
						},
						Attributes: map[string]schema.Attribute{
							"sql": schema.StringAttribute{
								MarkdownDescription: "Use SQL to filter the export.",
								Optional:            true,
							},
							"table": schema.SingleNestedAttribute{
								MarkdownDescription: "Use table-filter to filter the export.",
								Optional:            true,
								Computed:            true,
								PlanModifiers: []planmodifier.Object{
									objectplanmodifier.UseStateForUnknown(),
								},
								Attributes: map[string]schema.Attribute{
									"patterns": schema.ListAttribute{
										MarkdownDescription: "The table-filter expressions.",
										Optional:            true,
										Computed:            true,
										ElementType:         types.StringType,
										PlanModifiers: []planmodifier.List{
											listplanmodifier.UseStateForUnknown(),
										},
									},
									"where": schema.StringAttribute{
										MarkdownDescription: "Export only selected records.",
										Optional:            true,
										Computed:            true,
										PlanModifiers: []planmodifier.String{
											stringplanmodifier.UseStateForUnknown(),
										},
									},
								},
							},
						},
					},
					"csv_format": schema.SingleNestedAttribute{
						MarkdownDescription: "The format of the csv.",
						Optional:            true,
						Computed:            true,
						PlanModifiers: []planmodifier.Object{
							objectplanmodifier.UseStateForUnknown(),
						},
						Attributes: map[string]schema.Attribute{
							"separator": schema.StringAttribute{
								MarkdownDescription: "Separator of each value in CSV files.",
								Optional:            true,
								Computed:            true,
								PlanModifiers: []planmodifier.String{
									stringplanmodifier.UseStateForUnknown(),
								},
							},
							"delimiter": schema.StringAttribute{
								MarkdownDescription: "Delimiter of string type variables in CSV files.",
								Optional:            true,
								Computed:            true,
								PlanModifiers: []planmodifier.String{
									stringplanmodifier.UseStateForUnknown(),
								},
							},
							"null_value": schema.StringAttribute{
								MarkdownDescription: "Representation of null values in CSV files.",
								Optional:            true,
								Computed:            true,
								PlanModifiers: []planmodifier.String{
									stringplanmodifier.UseStateForUnknown(),
								},
							},
							"skip_header": schema.BoolAttribute{
								MarkdownDescription: "Export CSV files of the tables without header.",
								Optional:            true,
								Computed:            true,
								PlanModifiers: []planmodifier.Bool{
									boolplanmodifier.UseStateForUnknown(),
								},
							},
						},
					},
					"parquet_format": schema.SingleNestedAttribute{
						MarkdownDescription: "The format of the parquet.",
						Optional:            true,
						Computed:            true,
						PlanModifiers: []planmodifier.Object{
							objectplanmodifier.UseStateForUnknown(),
						},
						Attributes: map[string]schema.Attribute{
							"compression": schema.StringAttribute{
								MarkdownDescription: "The compression of the parquet.",
								Optional:            true,
								Computed:            true,
								PlanModifiers: []planmodifier.String{
									stringplanmodifier.UseStateForUnknown(),
								},
							},
						},
					},
				},
			},
			"target": schema.SingleNestedAttribute{
				MarkdownDescription: "The target type of the export.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.UseStateForUnknown(),
					objectplanmodifier.RequiresReplace(),
				},
				Attributes: map[string]schema.Attribute{
					"type": schema.StringAttribute{
						MarkdownDescription: "The exported target type.",
						Optional:            true,
						Computed:            true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"s3": schema.SingleNestedAttribute{
						MarkdownDescription: "S3 target.",
						Optional:            true,
						Attributes: map[string]schema.Attribute{
							"uri": schema.StringAttribute{
								MarkdownDescription: "The URI of the S3 folder.",
								Required:            true,
							},
							"auth_type": schema.StringAttribute{
								MarkdownDescription: "The auth method of the export S3.",
								Required:            true,
							},
							"access_key": schema.SingleNestedAttribute{
								MarkdownDescription: "The access key of the S3.",
								Optional:            true,
								Attributes: map[string]schema.Attribute{
									"id": schema.StringAttribute{
										MarkdownDescription: "The access key ID of the S3.",
										Required:            true,
									},
									"secret": schema.StringAttribute{
										MarkdownDescription: "The secret access key of the S3.",
										Required:            true,
										Sensitive:           true,
									},
								},
							},
							"role_arn": schema.StringAttribute{
								MarkdownDescription: "The role arn of the S3.",
								Optional:            true,
							},
						},
					},
					"gcs": schema.SingleNestedAttribute{
						MarkdownDescription: "GCS target.",
						Optional:            true,
						Attributes: map[string]schema.Attribute{
							"uri": schema.StringAttribute{
								MarkdownDescription: "The GCS URI of the export target.",
								Required:            true,
							},
							"auth_type": schema.StringAttribute{
								MarkdownDescription: "The auth method of the export target.",
								Required:            true,
							},
							"service_account_key": schema.StringAttribute{
								MarkdownDescription: "The service account key.",
								Optional:            true,
								Sensitive:           true,
							},
						},
					},
					"azure_blob": schema.SingleNestedAttribute{
						MarkdownDescription: "Azure Blob target.",
						Optional:            true,
						Attributes: map[string]schema.Attribute{
							"uri": schema.StringAttribute{
								MarkdownDescription: "The Azure Blob URI of the export target.",
								Required:            true,
							},
							"auth_type": schema.StringAttribute{
								MarkdownDescription: "The auth method of the export target.",
								Required:            true,
							},
							"sas_token": schema.StringAttribute{
								MarkdownDescription: "The sas token.",
								Optional:            true,
								Sensitive:           true,
							},
						},
					},
				},
			},
			"reason": schema.StringAttribute{
				MarkdownDescription: "The failed reason of the export.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *serverlessExportResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if !r.provider.configured {
		resp.Diagnostics.AddError(
			"Provider not configured",
			"The provider hasn't been configured before apply, likely because it depends on an unknown value from another resource. This leads to weird stuff happening, so we'd prefer if you didn't do that. Thanks!",
		)
		return
	}

	// get data from config
	var data serverlessExportResourceData
	diags := req.Config.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Trace(ctx, "create serverless_export_resource")
	body, err := buildCreateServerlessExportBody(ctx, data)
	if err != nil {
		resp.Diagnostics.AddError("Create Error", fmt.Sprintf("Unable to build CreateExport body, got error: %s", err))
		return
	}

	export, err := r.provider.ServerlessClient.CreateExport(ctx, data.ClusterId.ValueString(), &body)
	if err != nil {
		resp.Diagnostics.AddError("Create Error", fmt.Sprintf("Unable to create export, got error: %s", err))
		return
	}

	data.ExportId = types.StringValue(*export.ExportId)

	refreshServerlessExportResourceData(ctx, export, &data)

	// save to terraform state
	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (r *serverlessExportResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data serverlessExportResourceData
	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	export, err := r.provider.ServerlessClient.GetExport(ctx, data.ClusterId.ValueString(), data.ExportId.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Read Error", fmt.Sprintf("Unable to read export, got error: %s", err))
		return
	}

	refreshServerlessExportResourceData(ctx, export, &data)

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (r *serverlessExportResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
}

func (r *serverlessExportResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var clusterId string
	var exportId string

	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("cluster_id"), &clusterId)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("export_id"), &exportId)...)
	if resp.Diagnostics.HasError() {
		return
	}

	export, err := r.provider.ServerlessClient.GetExport(ctx, clusterId, exportId)
	if err != nil {
		resp.Diagnostics.AddError("Delete Error", fmt.Sprintf("Unable to get serverless export, got error: %s", err))
		return
	}
	if *export.State == exportV1beta1.EXPORTSTATEENUM_RUNNING {
		tflog.Trace(ctx, "serverless_export_resource is running, cancel it before delete")
		_, err := r.provider.ServerlessClient.CancelExport(ctx, clusterId, exportId)
		if err != nil {
			resp.Diagnostics.AddError("Cancel Error", fmt.Sprintf("Unable to cancel serverless export before delete, got error: %s", err))
			return
		}
	}

	tflog.Trace(ctx, "delete serverless_export_resource")
	_, err = r.provider.ServerlessClient.DeleteExport(ctx, clusterId, exportId)
	if err != nil {
		resp.Diagnostics.AddError("Delete Error", fmt.Sprintf("Unable to delete serverless export, got error: %s", err))
		return
	}
}

func (r *serverlessExportResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	idParts := strings.Split(req.ID, ",")

	if len(idParts) != 2 || idParts[0] == "" || idParts[1] == "" {
		resp.Diagnostics.AddError(
			"Unexpected Import Identifier",
			fmt.Sprintf("Expected import identifier with format: cluster_id, export_id. Got: %q", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("cluster_id"), idParts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("export_id"), idParts[1])...)
}

func buildCreateServerlessExportBody(ctx context.Context, data serverlessExportResourceData) (exportV1beta1.ExportServiceCreateExportBody, error) {
	body := exportV1beta1.ExportServiceCreateExportBody{}

	if IsKnown(data.DisplayName) {
		displayName := data.DisplayName.ValueString()
		body.DisplayName = &displayName
	}
	if data.ExportOptions != nil {
		body.ExportOptions = &exportV1beta1.ExportOptions{}
		if IsKnown(data.ExportOptions.FileType) {
			fileType := exportV1beta1.ExportFileTypeEnum(data.ExportOptions.FileType.ValueString())
			body.ExportOptions.FileType = &fileType
		}
		if IsKnown(data.ExportOptions.Compression) {
			compression := exportV1beta1.ExportCompressionTypeEnum(data.ExportOptions.Compression.ValueString())
			body.ExportOptions.Compression = &compression
		}
		if data.ExportOptions.Filter != nil {
			body.ExportOptions.Filter = &exportV1beta1.ExportOptionsFilter{}
			if IsKnown(data.ExportOptions.Filter.Sql) {
				sql := data.ExportOptions.Filter.Sql.ValueString()
				body.ExportOptions.Filter.Sql = &sql
			}
			if data.ExportOptions.Filter.Table != nil {
				body.ExportOptions.Filter.Table = &exportV1beta1.ExportOptionsFilterTable{}
				if IsKnown(data.ExportOptions.Filter.Table.Patterns) {
					var patterns []string
					diag := data.ExportOptions.Filter.Table.Patterns.ElementsAs(ctx, &patterns, false)
					if diag.HasError() {
						return exportV1beta1.ExportServiceCreateExportBody{}, errors.New("unable to get patterns")
					}
					body.ExportOptions.Filter.Table.Patterns = patterns
				}
				if IsKnown(data.ExportOptions.Filter.Table.Where) {
					where := data.ExportOptions.Filter.Table.Where.ValueString()
					body.ExportOptions.Filter.Table.Where = &where
				}
			}
		}

		if data.ExportOptions.CsvFormat != nil {
			csvFormat := exportV1beta1.ExportOptionsCSVFormat{}
			if IsKnown(data.ExportOptions.CsvFormat.Separator) {
				separator := data.ExportOptions.CsvFormat.Separator.ValueString()
				csvFormat.Separator = &separator
			}
			if IsKnown(data.ExportOptions.CsvFormat.Delimiter) {
				delimiter := data.ExportOptions.CsvFormat.Delimiter.ValueString()
				csvFormat.Delimiter = *exportV1beta1.NewNullableString(&delimiter)
			}
			if IsKnown(data.ExportOptions.CsvFormat.NullValue) {
				nullValue := data.ExportOptions.CsvFormat.NullValue.ValueString()
				csvFormat.NullValue = *exportV1beta1.NewNullableString(&nullValue)
			}
			if IsKnown(data.ExportOptions.CsvFormat.SkipHeader) {
				skipHeader := data.ExportOptions.CsvFormat.SkipHeader.ValueBool()
				csvFormat.SkipHeader = &skipHeader
			}
			body.ExportOptions.CsvFormat = &csvFormat
		}

		if data.ExportOptions.ParquetFormat != nil {
			compression := exportV1beta1.ExportCompressionTypeEnum(data.ExportOptions.ParquetFormat.Compression.ValueString())
			body.ExportOptions.ParquetFormat = &exportV1beta1.ExportOptionsParquetFormat{
				Compression: (*exportV1beta1.ExportParquetCompressionTypeEnum)(&compression),
			}
		}
	}
	if data.Target != nil {
		targetType := exportV1beta1.ExportTargetTypeEnum(data.Target.Type.ValueString())
		body.Target = &exportV1beta1.ExportTarget{
			Type: &targetType,
		}
		if data.Target.S3 != nil {
			uri := data.Target.S3.Uri.ValueString()
			authType := exportV1beta1.ExportS3AuthTypeEnum(data.Target.S3.AuthType.ValueString())
			body.Target.S3 = &exportV1beta1.S3Target{
				Uri:      &uri,
				AuthType: authType,
			}
			if data.Target.S3.AccessKey != nil {
				body.Target.S3.AccessKey = &exportV1beta1.S3TargetAccessKey{
					Id:     data.Target.S3.AccessKey.Id.ValueString(),
					Secret: data.Target.S3.AccessKey.Secret.ValueString(),
				}
			}
			if IsKnown(data.Target.S3.RoleArn) {
				roleArn := data.Target.S3.RoleArn.ValueString()
				body.Target.S3.RoleArn = &roleArn
			}
		}

		if data.Target.Gcs != nil {
			authType := exportV1beta1.ExportGcsAuthTypeEnum(data.Target.Gcs.AuthType.ValueString())
			body.Target.Gcs = &exportV1beta1.GCSTarget{
				Uri:      data.Target.Gcs.Uri.ValueString(),
				AuthType: authType,
			}
			if IsKnown(data.Target.Gcs.ServiceAccountKey) {
				serviceAccountKey := data.Target.Gcs.ServiceAccountKey.ValueString()
				body.Target.Gcs.ServiceAccountKey = &serviceAccountKey
			}
		}

		if data.Target.AzureBlob != nil {
			authType := exportV1beta1.ExportAzureBlobAuthTypeEnum(data.Target.AzureBlob.AuthType.ValueString())
			body.Target.AzureBlob = &exportV1beta1.AzureBlobTarget{
				Uri:      data.Target.AzureBlob.Uri.ValueString(),
				AuthType: authType,
			}
			if IsKnown(data.Target.AzureBlob.SasToken) {
				sasToken := data.Target.AzureBlob.SasToken.ValueString()
				body.Target.AzureBlob.SasToken = &sasToken
			}
		}
	}

	return body, nil
}

func refreshServerlessExportResourceData(ctx context.Context, resp *exportV1beta1.Export, data *serverlessExportResourceData) error {
	data.DisplayName = types.StringValue(*resp.DisplayName)
	data.State = types.StringValue(string(*resp.State))
	data.CreateTime = types.StringValue(resp.CreateTime.Format(time.RFC3339))
	data.CreatedBy = types.StringValue(*resp.CreatedBy)
	if resp.Reason.IsSet() {
		data.Reason = types.StringValue(*resp.Reason.Get())
	}
	if resp.UpdateTime.IsSet() {
		data.UpdateTime = types.StringValue(resp.UpdateTime.Get().Format(time.RFC3339))
	}
	if resp.CompleteTime.IsSet() {
		data.CompleteTime = types.StringValue(resp.CompleteTime.Get().Format(time.RFC3339))
	}
	if resp.SnapshotTime.IsSet() {
		data.SnapshotTime = types.StringValue(resp.SnapshotTime.Get().Format(time.RFC3339))
	}
	if resp.ExpireTime.IsSet() {
		data.ExpireTime = types.StringValue(resp.ExpireTime.Get().Format(time.RFC3339))
	}

	exportOptionsFileType := *resp.ExportOptions.FileType
	eo := exportOptions{
		FileType: types.StringValue(string(exportOptionsFileType)),
	}
	if resp.ExportOptions.Filter != nil {
		if resp.ExportOptions.Filter.Sql != nil {
			eo.Filter = &exportFilter{
				Sql: types.StringValue(*resp.ExportOptions.Filter.Sql),
			}
		} else {
			patterns, diag := types.ListValueFrom(ctx, types.StringType, resp.ExportOptions.Filter.Table.Patterns)
			if diag.HasError() {
				return errors.New("unable to convert export options filter table patterns")
			}
			eo.Filter = &exportFilter{
				Table: &tableFilter{
					Patterns: patterns,
					Where:    types.StringValue(*resp.ExportOptions.Filter.Table.Where),
				},
			}
		}
	}
	switch exportOptionsFileType {
	case exportV1beta1.EXPORTFILETYPEENUM_SQL:
		eo.Compression = types.StringValue(string(*resp.ExportOptions.Compression))
	case exportV1beta1.EXPORTFILETYPEENUM_CSV:
		eo.Compression = types.StringValue(string(*resp.ExportOptions.Compression))
		if resp.ExportOptions.CsvFormat != nil {
			eo.CsvFormat = &csvFormat{
				Separator:  types.StringValue(*resp.ExportOptions.CsvFormat.Separator),
				Delimiter:  types.StringValue(*resp.ExportOptions.CsvFormat.Delimiter.Get()),
				NullValue:  types.StringValue(*resp.ExportOptions.CsvFormat.NullValue.Get()),
				SkipHeader: types.BoolValue(*resp.ExportOptions.CsvFormat.SkipHeader),
			}
		}
	case exportV1beta1.EXPORTFILETYPEENUM_PARQUET:
		eo.ParquetFormat = &parquetFormat{
			Compression: types.StringValue(string(*resp.ExportOptions.ParquetFormat.Compression)),
		}
	}
	data.ExportOptions = &eo

	exportTargetType := *resp.Target.Type
	et := exportTarget{
		Type: types.StringValue(string(exportTargetType)),
	}
	switch exportTargetType {
	case exportV1beta1.EXPORTTARGETTYPEENUM_LOCAL:
	case exportV1beta1.EXPORTTARGETTYPEENUM_S3:
		et.S3 = &s3Target{
			Uri:      types.StringValue(*resp.Target.S3.Uri),
			AuthType: types.StringValue(string(resp.Target.S3.AuthType)),
			AccessKey: &accessKey{
				Id: types.StringValue(resp.Target.S3.AccessKey.Id),
			},
		}
	case exportV1beta1.EXPORTTARGETTYPEENUM_GCS:
		et.Gcs = &gcsTarget{
			Uri:      types.StringValue(resp.Target.Gcs.Uri),
			AuthType: types.StringValue(string(resp.Target.Gcs.AuthType)),
		}
	case exportV1beta1.EXPORTTARGETTYPEENUM_AZURE_BLOB:
		et.AzureBlob = &azureBlobTarget{
			Uri:      types.StringValue(resp.Target.AzureBlob.Uri),
			AuthType: types.StringValue(string(resp.Target.AzureBlob.AuthType)),
		}
	}
	data.Target = &et
	return nil
}
