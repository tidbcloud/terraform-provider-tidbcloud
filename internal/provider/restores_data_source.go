package provider

import (
	"context"
	"fmt"
	"math/rand"
	"strconv"

	restoreApi "github.com/c4pt0r/go-tidbcloud-sdk-v1/client/restore"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type restoresDataSourceData struct {
	Id        types.String `tfsdk:"id"`
	ProjectId string       `tfsdk:"project_id"`
	Page      types.Int64  `tfsdk:"page"`
	PageSize  types.Int64  `tfsdk:"page_size"`
	Items     []restore    `tfsdk:"items"`
	Total     types.Int64  `tfsdk:"total"`
}

type restore struct {
	Id              string  `tfsdk:"id"`
	CreateTimestamp string  `tfsdk:"create_timestamp"`
	BackupId        string  `tfsdk:"backup_id"`
	ClusterId       string  `tfsdk:"cluster_id"`
	Status          string  `tfsdk:"status"`
	Cluster         cluster `tfsdk:"cluster"`
	ErrorMessage    string  `tfsdk:"error_message"`
}

// Ensure provider defined types fully satisfy framework interfaces
var _ datasource.DataSource = &restoresDataSource{}

type restoresDataSource struct {
	provider *TidbcloudProvider
}

func NewRestoresDataSource() datasource.DataSource {
	return &restoresDataSource{}
}

func (d *restoresDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_restores"
}

func (d *restoresDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	var ok bool
	if d.provider, ok = req.ProviderData.(*TidbcloudProvider); !ok {
		resp.Diagnostics.AddError("Internal provider error",
			fmt.Sprintf("Error in Configure: expected %T but got %T", TidbcloudProvider{}, req.ProviderData))
	}
}

func (d *restoresDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "restores data source",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "data source ID.",
				Computed:            true,
			},
			"project_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the project. You can get the project ID from [tidbcloud_projects datasource](../data-sources/projects.md).",
				Required:            true,
			},
			"page": schema.Int64Attribute{
				MarkdownDescription: "Default:1 The number of pages.",
				Optional:            true,
				Computed:            true,
			},
			"page_size": schema.Int64Attribute{
				MarkdownDescription: "Default:10 The size of a pages.",
				Optional:            true,
				Computed:            true,
			},
			"items": schema.ListNestedAttribute{
				MarkdownDescription: "Default:10 The size of a pages.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							MarkdownDescription: "The ID of the restore task.",
							Computed:            true,
						},
						"create_timestamp": schema.StringAttribute{
							MarkdownDescription: "The creation time of the backup in UTC.The time format follows the ISO8601 standard, which is YYYY-MM-DD (year-month-day) + T +HH:MM:SS (hour-minutes-seconds) + Z. For example, 2020-01-01T00:00:00Z.",
							Computed:            true,
						},
						"backup_id": schema.StringAttribute{
							MarkdownDescription: "The ID of the backup.",
							Computed:            true,
						},
						"cluster_id": schema.StringAttribute{
							MarkdownDescription: "The cluster ID of the backup.",
							Computed:            true,
						},
						"status": schema.StringAttribute{
							MarkdownDescription: "Enum: \"PENDING\" \"RUNNING\" \"FAILED\" \"SUCCESS\", The status of the restore task.",
							Computed:            true,
						},
						"error_message": schema.StringAttribute{
							MarkdownDescription: "The error message of restore if failed.",
							Computed:            true,
						},
						"cluster": schema.SingleNestedAttribute{
							MarkdownDescription: "The information of the restored cluster. The restored cluster is the new cluster your backup data is restored to.",
							Computed:            true,
							Attributes: map[string]schema.Attribute{
								"id": schema.StringAttribute{
									MarkdownDescription: "The ID of the restored cluster. The restored cluster is the new cluster your backup data is restored to.",
									Computed:            true,
								},
								"name": schema.StringAttribute{
									MarkdownDescription: "The name of the restored cluster. The restored cluster is the new cluster your backup data is restored to.",
									Computed:            true,
								},
								"status": schema.StringAttribute{
									MarkdownDescription: "The status of the restored cluster. Possible values are \"AVAILABLE\", \"CREATING\", \"MODIFYING\", \"PAUSED\", \"RESUMING\", and \"CLEARED\".",
									Computed:            true,
								},
							},
						},
					},
				},
			},
			"total": schema.Int64Attribute{
				MarkdownDescription: "The total number of restore tasks in the project.",
				Computed:            true,
			},
		},
	}
}

func (d *restoresDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data restoresDataSourceData
	diags := req.Config.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// set default value
	var page int32 = 1
	var pageSize int32 = 10
	if !data.Page.IsNull() && !data.Page.IsUnknown() {
		page = int32(data.Page.ValueInt64())
	}
	if !data.PageSize.IsNull() && !data.PageSize.IsUnknown() {
		pageSize = int32(data.PageSize.ValueInt64())
	}

	tflog.Trace(ctx, "read restores data source")
	listRestoreTasksOK, err := d.provider.client.ListRestoreTasks(restoreApi.NewListRestoreTasksParams().WithProjectID(data.ProjectId).WithPage(&page).WithPageSize(&pageSize))
	if err != nil {
		resp.Diagnostics.AddError("Read Error", fmt.Sprintf("Unable to call GetRestoreTasks, got error: %s", err))
		return
	}

	data.Total = types.Int64Value(listRestoreTasksOK.Payload.Total)
	var items []restore
	for _, key := range listRestoreTasksOK.Payload.Items {
		items = append(items, restore{
			Id:              key.ID,
			CreateTimestamp: key.CreateTimestamp.String(),
			BackupId:        key.BackupID,
			ClusterId:       key.ClusterID,
			ErrorMessage:    key.ErrorMessage,
			Status:          key.Status,
			Cluster: cluster{
				Id:     key.Cluster.ID,
				Name:   key.Cluster.Name,
				Status: key.Cluster.Status,
			},
		})
	}
	data.Items = items
	data.Id = types.StringValue(strconv.FormatInt(rand.Int63(), 10))

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}
