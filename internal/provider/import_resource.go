package provider

import (
	"context"
	"errors"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	importService "github.com/tidbcloud/terraform-provider-tidbcloud/pkg/import/client/import_service"
	importModel "github.com/tidbcloud/terraform-provider-tidbcloud/pkg/import/models"
	"os"
	"strconv"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &ImportResource{}

func NewImportResource() resource.Resource {
	return &ImportResource{}
}

// ImportResource defines the resource implementation.
type ImportResource struct {
	provider *tidbcloudProvider
}

// ImportResourceModel describes the resource data model.
type ImportResourceModel struct {
	Id        types.String `tfsdk:"id"`
	ClusterId types.String `tfsdk:"cluster_id"`
	ProjectId types.String `tfsdk:"project_id"`
	// The type of data source. Required: true
	Type types.String `tfsdk:"type"`
	// The format of data to import. Required: true
	DataFormat types.String `tfsdk:"data_format"`
	// The CSV configuration.
	CsvFormat *ImportCustomCSVFormat `tfsdk:"csv_format"`

	/*
		used for importing from local file
	*/
	// The file name returned by generating upload url.
	FileName types.String `tfsdk:"file_name"`
	// The target db and table to import data.
	TargetTable *ImportTargetTable `tfsdk:"target_table"`
	NewFileName types.String       `tfsdk:"new_file_name"`

	/*
		used for importing from S3
	*/
	// The arn of AWS IAM role.
	AwsRoleArn types.String `tfsdk:"aws_role_arn"`
	// The full s3 path that contains data to import.
	SourceURL types.String `tfsdk:"source_url"`

	// computed fields
	CreatedAt                  types.String `tfsdk:"created_at"`
	Status                     types.String `tfsdk:"status"`
	TotalSize                  types.String `tfsdk:"total_size"`
	TotalFiles                 types.Int64  `tfsdk:"total_files"`
	CompletedTables            types.Int64  `tfsdk:"completed_tables"`
	PendingTables              types.Int64  `tfsdk:"pending_tables"`
	CompletedPercent           types.Int64  `tfsdk:"completed_percent"`
	Message                    types.String `tfsdk:"message"`
	ElapsedTimeSeconds         types.Int64  `tfsdk:"elapsed_time_seconds"`
	ProcessedSourceDataSize    types.String `tfsdk:"processed_source_data_size"`
	TotalTablesCount           types.Int64  `tfsdk:"total_tables_count"`
	PostImportCompletedPercent types.Int64  `tfsdk:"post_import_completed_percent"`
	AllCompletedTables         types.List   `tfsdk:"all_completed_tables"`
}

type ImportCustomCSVFormat struct {
	// backslash escape
	BackslashEscape types.Bool `tfsdk:"backslash_escape"`
	// delimiter
	Delimiter types.String `tfsdk:"delimiter"`
	// header
	Header types.Bool `tfsdk:"header"`
	// separator
	Separator types.String `tfsdk:"separator"`
	// trim last separator
	TrimLastSeparator types.Bool `tfsdk:"trim_last_separator"`
}

type ImportTargetTable struct {
	// database
	Database types.String `tfsdk:"database"`
	// table
	Table types.String `tfsdk:"table"`
}

type ImportAllCompletedTable struct {
	TableName types.String `tfsdk:"table_name"`
	Result    types.String `tfsdk:"result"`
	Message   types.String `tfsdk:"message"`
}

func (r *ImportResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_import"
}

func (r *ImportResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ImportResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "import resource",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The ID of the import.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"project_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the project. You can get the project ID from [tidbcloud_projects datasource](../data-sources/projects.md).",
				Required:            true,
			},
			"cluster_id": schema.StringAttribute{
				MarkdownDescription: "The ID of your cluster.",
				Required:            true,
			},
			"type": schema.StringAttribute{
				MarkdownDescription: "The type of data source. Enum: \"S3\" \"LOCAL\".",
				Required:            true,
			},
			"data_format": schema.StringAttribute{
				MarkdownDescription: "The format of data to import.Enum: \"SqlFile\" \"AuroraSnapshot\" \"CSV\" \"Parquet\".",
				Required:            true,
			},
			"csv_format": schema.SingleNestedAttribute{
				MarkdownDescription: "The CSV configuration.",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"backslash_escape": schema.BoolAttribute{
						MarkdownDescription: "In CSV file whether to parse backslash inside fields as escape characters (default true).",
						Optional:            true,
					},
					"delimiter": schema.StringAttribute{
						MarkdownDescription: "The delimiter used for quoting of CSV file (default \"\\\"\").",
						Optional:            true,
					},
					"header": schema.BoolAttribute{
						MarkdownDescription: "In CSV file whether regard the first row as header (default true).",
						Optional:            true,
					},
					"separator": schema.StringAttribute{
						MarkdownDescription: "The field separator of CSV file (default \",\").",
						Optional:            true,
					},
					"trim_last_separator": schema.BoolAttribute{
						MarkdownDescription: "In CSV file whether to treat Separator as the line terminator and trim all trailing separators (default false).",
						Optional:            true,
					},
				},
			},
			"file_name": schema.StringAttribute{
				MarkdownDescription: "The local file path, used for importing from LOCAL",
				Optional:            true,
			},
			"target_table": schema.SingleNestedAttribute{
				MarkdownDescription: "The target db and table to import data, used for importing from LOCAL",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"database": schema.StringAttribute{
						MarkdownDescription: "The database of your cluster.",
						Optional:            true,
					},
					"table": schema.StringAttribute{
						MarkdownDescription: "The table of your cluster.",
						Optional:            true,
					},
				},
			},
			"new_file_name": schema.StringAttribute{
				MarkdownDescription: "The file name returned by generating upload url, used for importing from local file.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"aws_role_arn": schema.StringAttribute{
				MarkdownDescription: "The arn of AWS IAM role, used for importing from S3",
				Optional:            true,
			},
			"source_url": schema.StringAttribute{
				MarkdownDescription: "The full s3 path that contains data to import, used for importing from S3",
				Optional:            true,
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "Import task create time",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"status": schema.StringAttribute{
				MarkdownDescription: "Import task status",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"total_size": schema.StringAttribute{
				MarkdownDescription: "Import task total size",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"total_files": schema.Int64Attribute{
				MarkdownDescription: "Import task total files",
				Computed:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"completed_tables": schema.Int64Attribute{
				MarkdownDescription: "Import task completed tables",
				Computed:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"pending_tables": schema.Int64Attribute{
				MarkdownDescription: "Import task pending tables",
				Computed:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"completed_percent": schema.Int64Attribute{
				MarkdownDescription: "Import task completed percent",
				Computed:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"message": schema.StringAttribute{
				MarkdownDescription: "Import task message",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"elapsed_time_seconds": schema.Int64Attribute{
				MarkdownDescription: "Import task elapsed time seconds",
				Computed:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"processed_source_data_size": schema.StringAttribute{
				MarkdownDescription: "Import task processed source data size",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"total_tables_count": schema.Int64Attribute{
				MarkdownDescription: "Import task total tables count",
				Computed:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"post_import_completed_percent": schema.Int64Attribute{
				MarkdownDescription: "Import task post import completed percent",
				Computed:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"all_completed_tables": schema.ListAttribute{
				MarkdownDescription: "Import task all completed tables",
				Computed:            true,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.UseStateForUnknown(),
				},
				ElementType: types.ObjectType{
					AttrTypes: map[string]attr.Type{
						"table_name": types.StringType,
						"result":     types.StringType,
						"message":    types.StringType,
					},
				},
			},
		},
	}
}

func (r *ImportResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if r.provider == nil || !r.provider.configured {
		resp.Diagnostics.AddError(
			"Provider not configured",
			"The provider hasn't been configured before apply, likely because it depends on an unknown value from another resource. This leads to weird stuff happening, so we'd prefer if you didn't do that. Thanks!",
		)
		return
	}

	// Read Terraform plan data into the model
	var data *ImportResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Trace(ctx, "created import resource")
	// build body
	body, buildBodyErr := buildCreateImportBody(data)
	if buildBodyErr != nil {
		resp.Diagnostics.AddError("Create Error", fmt.Sprintf("Unable to buildCreateImportBody, got error: %s", buildBodyErr))
		return
	}

	// import with LOCAL type need to upload the file first
	if data.Type.ValueString() == "LOCAL" {
		newFileName, uploadError := r.uploadFile(data.FileName.ValueString(), data)
		if uploadError != nil {
			resp.Diagnostics.AddError("Create Error", fmt.Sprintf("Unable to uploadFile, got error: %s", uploadError))
			return
		}
		body.FileName = newFileName
		data.NewFileName = types.StringValue(newFileName)
	} else {
		// all values must be known after apply,
		data.NewFileName = types.StringNull()
	}
	// call CreateImport
	createImportParams := importService.NewCreateImportParams().WithProjectID(data.ProjectId.ValueString()).WithClusterID(data.ClusterId.ValueString()).WithBody(*body)
	createImportResp, err := r.provider.client.CreateImport(createImportParams)
	if err != nil {
		resp.Diagnostics.AddError("Create Error", fmt.Sprintf("Unable to call CreateImport, got error: %s", err))
		return
	}
	data.Id = types.StringValue(*createImportResp.Payload.ID)

	// Refresh for any unknown value.
	tflog.Trace(ctx, "read import resource")
	getImportParams := importService.NewGetImportParams().WithProjectID(data.ProjectId.ValueString()).WithClusterID(data.ClusterId.ValueString()).WithID(data.Id.ValueString())
	getImportResp, err := r.provider.client.GetImport(getImportParams)
	if err != nil {
		resp.Diagnostics.AddError("Create Error", fmt.Sprintf("Unable to call GetImport, got error: %s", err))
		return
	}

	resp.Diagnostics.Append(refreshImportResource(ctx, data, getImportResp.Payload)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Trace(ctx, "save into the Terraform state.")
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func buildCreateImportBody(data *ImportResourceModel) (*importService.CreateImportBody, error) {
	// check dataFormat
	dataFormat := importModel.OpenapiDataFormat(data.DataFormat.ValueString())
	if dataFormat != importModel.OpenapiDataFormatSQLFile && dataFormat != importModel.OpenapiDataFormatParquet && dataFormat != importModel.OpenapiDataFormatAuroraSnapshot && dataFormat != importModel.OpenapiDataFormatCSV {
		return nil, errors.New("invalid import dataFormat, only support SqlFile, AuroraSnapshot CSV and Parquet now")
	}
	// build body
	body := &importService.CreateImportBody{
		DataFormat: &dataFormat,
	}
	// build body by type
	importType := importModel.CreateImportReqImportType(data.Type.ValueString())
	body.Type = &importType
	if importType == importModel.CreateImportReqImportTypeS3 {
		if dataFormat == importModel.OpenapiDataFormatCSV {
			body.CsvFormat = buildCSVFormat(data)
		}
		if data.AwsRoleArn.ValueString() == "" {
			return nil, errors.New("AwsRoleArn can not be empty in S3 type")
		}
		body.AwsRoleArn = data.AwsRoleArn.ValueString()
		if data.SourceURL.ValueString() == "" {
			return nil, errors.New("SourceURL can not be empty in S3 type")
		}
		body.SourceURL = data.SourceURL.ValueString()
	} else if importType == importModel.CreateImportReqImportTypeLOCAL {
		if dataFormat != importModel.OpenapiDataFormatCSV {
			return nil, errors.New("LOCAL type only support CSV data format now")
		}
		body.CsvFormat = buildCSVFormat(data)
		if data.FileName.ValueString() == "" {
			return nil, errors.New("FileName can not be empty in Local type")
		}
		if data.TargetTable == nil {
			return nil, errors.New("TargetTable can not be empty in Local type")
		}
		if data.TargetTable.Database.IsNull() || data.TargetTable.Database.IsUnknown() {
			return nil, errors.New("TargetTable's Database can not be empty in Local type")
		}
		if data.TargetTable.Table.IsNull() || data.TargetTable.Table.IsUnknown() {
			return nil, errors.New("TargetTable's Database can not be empty in Local type")
		}
		body.TargetTable = &importModel.OpenapiTable{
			Schema: data.TargetTable.Database.ValueString(),
			Table:  data.TargetTable.Table.ValueString(),
		}
	} else {
		return nil, errors.New("invalid import type, Only support LOCAL and S3 now")
	}
	return body, nil
}

// UploadFile will upload the Local file and return the new url
func (r *ImportResource) uploadFile(fileName string, data *ImportResourceModel) (string, error) {
	localFile, err := os.Open(fileName)
	if err != nil {
		return "", err
	}
	defer localFile.Close()

	stat, err := localFile.Stat()
	if err != nil {
		return "", err
	}
	size := strconv.FormatInt(stat.Size(), 10)
	name := stat.Name()
	urlRes, err := r.provider.client.GenerateUploadURL(importService.NewGenerateUploadURLParams().WithProjectID(data.ProjectId.ValueString()).WithClusterID(data.ClusterId.ValueString()).WithBody(importService.GenerateUploadURLBody{
		ContentLength: &size,
		FileName:      &name,
	}))
	if err != nil {
		return "", err
	}
	url := urlRes.Payload.UploadURL

	err = r.provider.client.PreSignedUrlUpload(url, localFile, stat.Size())
	if err != nil {
		return "", err
	}
	return *urlRes.Payload.NewFileName, nil
}

func buildCSVFormat(data *ImportResourceModel) *importModel.OpenapiCustomCSVFormat {
	var importCustomCSVFormat = &importModel.OpenapiCustomCSVFormat{
		BackslashEscape:   true,
		Delimiter:         "\"",
		Header:            true,
		Separator:         ",",
		TrimLastSeparator: false,
	}
	if data.CsvFormat != nil {
		if !data.CsvFormat.BackslashEscape.IsNull() && !data.CsvFormat.BackslashEscape.IsUnknown() {
			importCustomCSVFormat.BackslashEscape = data.CsvFormat.BackslashEscape.ValueBool()
		}
		if !data.CsvFormat.Delimiter.IsNull() && !data.CsvFormat.Delimiter.IsUnknown() {
			importCustomCSVFormat.Delimiter = data.CsvFormat.Delimiter.ValueString()
		}
		if !data.CsvFormat.Header.IsNull() && !data.CsvFormat.Header.IsUnknown() {
			importCustomCSVFormat.Header = data.CsvFormat.Header.ValueBool()
		}
		if !data.CsvFormat.Separator.IsNull() && !data.CsvFormat.Separator.IsUnknown() {
			importCustomCSVFormat.Separator = data.CsvFormat.Separator.ValueString()
		}
		if !data.CsvFormat.TrimLastSeparator.IsNull() && !data.CsvFormat.TrimLastSeparator.IsUnknown() {
			importCustomCSVFormat.TrimLastSeparator = data.CsvFormat.TrimLastSeparator.ValueBool()
		}
	}
	return importCustomCSVFormat
}

func (r *ImportResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data *ImportResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Trace(ctx, "read import resource")
	getImportParams := importService.NewGetImportParams().WithProjectID(data.ProjectId.ValueString()).WithClusterID(data.ClusterId.ValueString()).WithID(data.Id.ValueString())
	getImportResp, err := r.provider.client.GetImport(getImportParams)
	if err != nil {
		resp.Diagnostics.AddError("Read Error", fmt.Sprintf("Unable to call GetClusterById, got error: %s", err))
		return
	}

	resp.Diagnostics.Append(refreshImportResource(ctx, data, getImportResp.Payload)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func refreshImportResource(ctx context.Context, data *ImportResourceModel, payload *importModel.OpenapiGetImportResp) diag.Diagnostics {

	data.CreatedAt = types.StringValue(payload.CreatedAt.String())
	data.Status = types.StringValue(string(*payload.Status))
	data.TotalSize = types.StringValue(*payload.TotalSize)
	data.TotalFiles = types.Int64Value(*payload.TotalFiles)
	data.CompletedTables = types.Int64Value(*payload.CompletedTables)
	data.PendingTables = types.Int64Value(*payload.PendingTables)
	data.CompletedPercent = types.Int64Value(*payload.CompletedPercent)
	data.ElapsedTimeSeconds = types.Int64Value(*payload.ElapsedTimeSeconds)
	data.TotalTablesCount = types.Int64Value(payload.TotalTablesCount)
	data.PostImportCompletedPercent = types.Int64Value(payload.PostImportCompletedPercent)
	data.Message = types.StringValue(*payload.Message)
	data.ProcessedSourceDataSize = types.StringValue(payload.ProcessedSourceDataSize)

	var allCompletedTables []ImportAllCompletedTable
	for _, ct := range payload.AllCompletedTables {
		allCompletedTables = append(allCompletedTables, ImportAllCompletedTable{
			TableName: types.StringValue(*ct.TableName),
			Result:    types.StringValue(string(*ct.Result)),
			Message:   types.StringValue(ct.Message),
		})
	}

	objectMap := map[string]attr.Type{
		"table_name": types.StringType,
		"result":     types.StringType,
		"message":    types.StringType,
	}
	listValue, diags := types.ListValueFrom(ctx, types.ObjectType{AttrTypes: objectMap}, allCompletedTables)
	data.AllCompletedTables = listValue
	return diags
}

func (r *ImportResource) Update(_ context.Context, _ resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Unsupported", fmt.Sprintf("import resource can't be updated"))
}

func (r *ImportResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data *ImportResourceModel
	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Trace(ctx, "delete import resource")
	cancelImportParams := importService.NewCancelImportParams().WithProjectID(data.ProjectId.ValueString()).WithClusterID(data.ClusterId.ValueString()).WithID(data.Id.ValueString())
	_, err := r.provider.client.CancelImport(cancelImportParams)
	if err != nil {
		resp.Diagnostics.AddError("Delete Error", fmt.Sprintf("Unable to call CancelImport, got error: %s", err))
		return
	}
}
