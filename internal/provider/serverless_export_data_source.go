package provider

import (
	"context"
	"errors"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	exportV1beta1 "github.com/tidbcloud/tidbcloud-cli/pkg/tidbcloud/v1beta1/serverless/export"
)

type serverlessExportDataSourceData struct {
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

var _ datasource.DataSource = &serverlessExportDataSource{}

type serverlessExportDataSource struct {
	provider *tidbcloudProvider
}

func NewServerlessExportDataSource() datasource.DataSource {
	return &serverlessExportDataSource{}
}

func (d *serverlessExportDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_serverless_export"
}

func (d *serverlessExportDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	var ok bool
	if d.provider, ok = req.ProviderData.(*tidbcloudProvider); !ok {
		resp.Diagnostics.AddError("Internal provider error",
			fmt.Sprintf("Error in Configure: expected %T but got %T", tidbcloudProvider{}, req.ProviderData))
	}
}

func (d *serverlessExportDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Serverless Export Resource",
		Attributes: map[string]schema.Attribute{
			"export_id": schema.StringAttribute{
				MarkdownDescription: "The unique ID of the export.",
				Required:            true,
			},
			"cluster_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the cluster.",
				Required:            true,
			},
			"display_name": schema.StringAttribute{
				MarkdownDescription: "The display name of the export.",
				Computed:            true,
			},
			"state": schema.StringAttribute{
				MarkdownDescription: "The state of the export.",
				Computed:            true,
			},
			"create_time": schema.StringAttribute{
				MarkdownDescription: "Timestamp when the export was created.",
				Computed:            true,
			},
			"created_by": schema.StringAttribute{
				MarkdownDescription: "The user who created the export.",
				Computed:            true,
			},
			"update_time": schema.StringAttribute{
				MarkdownDescription: "Timestamp when the export was updated.",
				Computed:            true,
			},
			"complete_time": schema.StringAttribute{
				MarkdownDescription: "Timestamp when the export was completed.",
				Computed:            true,
			},
			"snapshot_time": schema.StringAttribute{
				MarkdownDescription: "Snapshot time of the export.",
				Computed:            true,
			},
			"expire_time": schema.StringAttribute{
				MarkdownDescription: "Expire time of the export.",
				Computed:            true,
			},
			"export_options": schema.SingleNestedAttribute{
				MarkdownDescription: "The options of the export.",
				Computed:            true,
				Attributes: map[string]schema.Attribute{
					"file_type": schema.StringAttribute{
						MarkdownDescription: "The exported file type. Available values are SQL, CSV and Parquet. Default is CSV.",
						Computed:            true,
					},
					"compression": schema.StringAttribute{
						MarkdownDescription: "The compression of the export.",
						Computed:            true,
					},
					"filter": schema.SingleNestedAttribute{
						MarkdownDescription: "The filter of the export.",
						Computed:            true,
						Attributes: map[string]schema.Attribute{
							"sql": schema.StringAttribute{
								MarkdownDescription: "Use SQL to filter the export.",
								Computed:            true,
							},
							"table": schema.SingleNestedAttribute{
								MarkdownDescription: "Use table-filter to filter the export.",
								Computed:            true,
								Attributes: map[string]schema.Attribute{
									"patterns": schema.ListAttribute{
										MarkdownDescription: "The table-filter expressions.",
										Computed:            true,
										ElementType:         types.StringType,
									},
									"where": schema.StringAttribute{
										MarkdownDescription: "Export only selected records.",
										Computed:            true,
									},
								},
							},
						},
					},
					"csv_format": schema.SingleNestedAttribute{
						MarkdownDescription: "The format of the csv.",
						Computed:            true,
						Attributes: map[string]schema.Attribute{
							"separator": schema.StringAttribute{
								MarkdownDescription: "Separator of each value in CSV files.",
								Computed:            true,
							},
							"delimiter": schema.StringAttribute{
								MarkdownDescription: "Delimiter of string type variables in CSV files.",
								Computed:            true,
							},
							"null_value": schema.StringAttribute{
								MarkdownDescription: "Representation of null values in CSV files.",
								Computed:            true,
							},
							"skip_header": schema.BoolAttribute{
								MarkdownDescription: "Export CSV files of the tables without header.",
								Computed:            true,
							},
						},
					},
					"parquet_format": schema.SingleNestedAttribute{
						MarkdownDescription: "The format of the parquet.",
						Computed:            true,
						Attributes: map[string]schema.Attribute{
							"compression": schema.StringAttribute{
								MarkdownDescription: "The compression of the parquet.",
								Computed:            true,
							},
						},
					},
				},
			},
			"target": schema.SingleNestedAttribute{
				MarkdownDescription: "The target of the export.",
				Computed:            true,
				Attributes: map[string]schema.Attribute{
					"type": schema.StringAttribute{
						MarkdownDescription: "The exported file type.",
						Computed:            true,
					},
					"s3": schema.SingleNestedAttribute{
						MarkdownDescription: "S3 target.",
						Computed:            true,
						Attributes: map[string]schema.Attribute{
							"uri": schema.StringAttribute{
								MarkdownDescription: "The URI of the s3 folder.",
								Computed:            true,
							},
							"auth_type": schema.StringAttribute{
								MarkdownDescription: "The auth method of the export s3.",
								Computed:            true,
							},
							"access_key": schema.SingleNestedAttribute{
								MarkdownDescription: "The access key of the s3.",
								Computed:            true,
								Attributes: map[string]schema.Attribute{
									"id": schema.StringAttribute{
										MarkdownDescription: "The access key id of the s3.",
										Computed:            true,
									},
								},
							},
						},
					},
					"gcs": schema.SingleNestedAttribute{
						MarkdownDescription: "GCS target.",
						Computed:            true,
						Attributes: map[string]schema.Attribute{
							"uri": schema.StringAttribute{
								MarkdownDescription: "The GCS URI of the export target.",
								Computed:            true,
							},
							"auth_type": schema.StringAttribute{
								MarkdownDescription: "The auth method of the export target.",
								Computed:            true,
							},
						},
					},
					"azure_blob": schema.SingleNestedAttribute{
						MarkdownDescription: "Azure Blob target.",
						Computed:            true,
						Attributes: map[string]schema.Attribute{
							"uri": schema.StringAttribute{
								MarkdownDescription: "The Azure Blob URI of the export target.",
								Computed:            true,
							},
							"auth_type": schema.StringAttribute{
								MarkdownDescription: "The auth method of the export target.",
								Computed:            true,
							},
						},
					},
				},
			},
			"reason": schema.StringAttribute{
				MarkdownDescription: "The failed reason of the export.",
				Computed:            true,
			},
		},
	}
}

func (d *serverlessExportDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data serverlessExportDataSourceData
	diags := req.Config.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Trace(ctx, "read serverless export data source")
	export, err := d.provider.ServerlessClient.GetExport(ctx, data.ClusterId.ValueString(), data.ExportId.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Read Error", fmt.Sprintf("Unable to call GetExport, got error: %s", err))
		return
	}

	err = refreshServerlessExportDataSourceData(ctx, export, &data)
	if err != nil {
		resp.Diagnostics.AddError("Read Error", fmt.Sprintf("Unable to refresh serverless export data source data, got error: %s", err))
		return
	}

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func refreshServerlessExportDataSourceData(ctx context.Context, resp *exportV1beta1.Export, data *serverlessExportDataSourceData) error {
	data.DisplayName = types.StringValue(*resp.DisplayName)
	data.State = types.StringValue(string(*resp.State))
	data.CreateTime = types.StringValue(resp.CreateTime.String())
	data.CreatedBy = types.StringValue(*resp.CreatedBy)
	if resp.Reason.IsSet() {
		data.Reason = types.StringValue(*resp.Reason.Get())
	}
	if resp.UpdateTime.IsSet() {
		data.UpdateTime = types.StringValue(resp.UpdateTime.Get().String())
	}
	if resp.CompleteTime.IsSet() {
		data.CompleteTime = types.StringValue(resp.CompleteTime.Get().String())
	}
	if resp.SnapshotTime.IsSet() {
		data.SnapshotTime = types.StringValue(resp.SnapshotTime.Get().String())
	}
	if resp.ExpireTime.IsSet() {
		data.ExpireTime = types.StringValue(resp.ExpireTime.Get().String())
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
			patterns, diag := types.ListValueFrom(ctx, types.StringType, data.ExportOptions.Filter.Table.Patterns)
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
