package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/juju/errors"
	exportV1beta1 "github.com/tidbcloud/tidbcloud-cli/pkg/tidbcloud/v1beta1/serverless/export"
)

type serverlessExportsDataSourceData struct {
	ClusterId types.String           `tfsdk:"cluster_id"`
	Exports   []serverlessExportItem `tfsdk:"exports"`
}

type serverlessExportItem struct {
	ExportId      types.String   `tfsdk:"export_id"`
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

var _ datasource.DataSource = &serverlessExportsDataSource{}

type serverlessExportsDataSource struct {
	provider *tidbcloudProvider
}

func NewServerlessExportsDataSource() datasource.DataSource {
	return &serverlessExportsDataSource{}
}

func (d *serverlessExportsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_serverless_exports"
}

func (d *serverlessExportsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	var ok bool
	if d.provider, ok = req.ProviderData.(*tidbcloudProvider); !ok {
		resp.Diagnostics.AddError("Internal provider error",
			fmt.Sprintf("Error in Configure: expected %T but got %T", tidbcloudProvider{}, req.ProviderData))
	}
}

func (d *serverlessExportsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "serverless exports data source",
		Attributes: map[string]schema.Attribute{
			"cluster_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the cluster.",
				Required:            true,
			},
			"exports": schema.ListNestedAttribute{
				MarkdownDescription: "The exports of the cluster.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"export_id": schema.StringAttribute{
							MarkdownDescription: "The unique ID of the export.",
							Computed:            true,
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
							MarkdownDescription: "The target type of the export.",
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
											MarkdownDescription: "The URI of the S3 folder.",
											Computed:            true,
										},
										"auth_type": schema.StringAttribute{
											MarkdownDescription: "The auth method of the export S3.",
											Computed:            true,
										},
										"access_key": schema.SingleNestedAttribute{
											MarkdownDescription: "The access key of the S3.",
											Computed:            true,
											Attributes: map[string]schema.Attribute{
												"id": schema.StringAttribute{
													MarkdownDescription: "The access key ID of the S3.",
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
				},
			},
		},
	}
}

func (d *serverlessExportsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data serverlessExportsDataSourceData
	diags := req.Config.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Trace(ctx, "read serverless exports data source")
	exports, err := d.retrieveExports(ctx, data.ClusterId.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Read Error", fmt.Sprintf("Unable to call ListExports, got error: %s", err))
		return
	}
	var items []serverlessExportItem
	for _, export := range exports {
		var e serverlessExportItem
		e.DisplayName = types.StringValue(*export.DisplayName)
		e.State = types.StringValue(string(*export.State))
		e.CreateTime = types.StringValue(export.CreateTime.String())
		e.CreatedBy = types.StringValue(*export.CreatedBy)
		if export.Reason.IsSet() {
			e.Reason = types.StringValue(*export.Reason.Get())
		}
		if export.UpdateTime.IsSet() {
			e.UpdateTime = types.StringValue(export.UpdateTime.Get().String())
		}
		if export.CompleteTime.IsSet() {
			e.CompleteTime = types.StringValue(export.CompleteTime.Get().String())
		}
		if export.SnapshotTime.IsSet() {
			e.SnapshotTime = types.StringValue(export.SnapshotTime.Get().String())
		}
		if export.ExpireTime.IsSet() {
			e.ExpireTime = types.StringValue(export.ExpireTime.Get().String())
		}

		exportOptionsFileType := *export.ExportOptions.FileType
		eo := exportOptions{
			FileType: types.StringValue(string(exportOptionsFileType)),
		}
		if export.ExportOptions.Filter != nil {
			if export.ExportOptions.Filter.Sql != nil {
				eo.Filter = &exportFilter{
					Sql: types.StringValue(*export.ExportOptions.Filter.Sql),
				}
			} else {
				patterns, diag := types.ListValueFrom(ctx, types.StringType, e.ExportOptions.Filter.Table.Patterns)
				if diag.HasError() {
					resp.Diagnostics.AddError("Read Error", fmt.Sprintf("Unable to convert patterns to list, got error: %s", diag))
					return
				}
				eo.Filter = &exportFilter{
					Table: &tableFilter{
						Patterns: patterns,
						Where:    types.StringValue(*export.ExportOptions.Filter.Table.Where),
					},
				}
			}
		}
		switch exportOptionsFileType {
		case exportV1beta1.EXPORTFILETYPEENUM_SQL:
			eo.Compression = types.StringValue(string(*export.ExportOptions.Compression))
		case exportV1beta1.EXPORTFILETYPEENUM_CSV:
			eo.Compression = types.StringValue(string(*export.ExportOptions.Compression))
			if export.ExportOptions.CsvFormat != nil {
				eo.CsvFormat = &csvFormat{
					Separator:  types.StringValue(*export.ExportOptions.CsvFormat.Separator),
					Delimiter:  types.StringValue(*export.ExportOptions.CsvFormat.Delimiter.Get()),
					NullValue:  types.StringValue(*export.ExportOptions.CsvFormat.NullValue.Get()),
					SkipHeader: types.BoolValue(*export.ExportOptions.CsvFormat.SkipHeader),
				}
			}
		case exportV1beta1.EXPORTFILETYPEENUM_PARQUET:
			eo.ParquetFormat = &parquetFormat{
				Compression: types.StringValue(string(*export.ExportOptions.ParquetFormat.Compression)),
			}
		}
		e.ExportOptions = &eo

		exportTargetType := *export.Target.Type
		et := exportTarget{
			Type: types.StringValue(string(exportTargetType)),
		}
		switch exportTargetType {
		case exportV1beta1.EXPORTTARGETTYPEENUM_LOCAL:
		case exportV1beta1.EXPORTTARGETTYPEENUM_S3:
			et.S3 = &s3Target{
				Uri:      types.StringValue(*export.Target.S3.Uri),
				AuthType: types.StringValue(string(export.Target.S3.AuthType)),
				AccessKey: &accessKey{
					Id: types.StringValue(export.Target.S3.AccessKey.Id),
				},
			}
		case exportV1beta1.EXPORTTARGETTYPEENUM_GCS:
			et.Gcs = &gcsTarget{
				Uri:      types.StringValue(export.Target.Gcs.Uri),
				AuthType: types.StringValue(string(export.Target.Gcs.AuthType)),
			}
		case exportV1beta1.EXPORTTARGETTYPEENUM_AZURE_BLOB:
			et.AzureBlob = &azureBlobTarget{
				Uri:      types.StringValue(export.Target.AzureBlob.Uri),
				AuthType: types.StringValue(string(export.Target.AzureBlob.AuthType)),
			}
		}
		e.Target = &et
		items = append(items, e)
	}

	data.Exports = items
	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (d serverlessExportsDataSource) retrieveExports(ctx context.Context, clusterId string) ([]exportV1beta1.Export, error) {
	var items []exportV1beta1.Export
	pageSizeInt32 := int32(DefaultPageSize)
	var pageToken *string
	for {
		exports, err := d.provider.ServerlessClient.ListExports(ctx, clusterId, &pageSizeInt32, pageToken, nil)
		if err != nil {
			return nil, errors.Trace(err)
		}
		items = append(items, exports.Exports...)

		pageToken = exports.NextPageToken
		if IsNilOrEmpty(pageToken) {
			break
		}
	}
	return items, nil
}
