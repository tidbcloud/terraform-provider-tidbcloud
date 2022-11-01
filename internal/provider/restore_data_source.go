package provider

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"math/rand"
)

type restoresDataSourceData struct {
	Id        types.Int64 `tfsdk:"id"`
	ProjectId string      `tfsdk:"project_id"`
	Page      types.Int64 `tfsdk:"page"`
	PageSize  types.Int64 `tfsdk:"page_size"`
	Items     []restore   `tfsdk:"items"`
	Total     types.Int64 `tfsdk:"total"`
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
var _ provider.DataSourceType = restoresDataSourceType{}
var _ datasource.DataSource = restoresDataSource{}

type restoresDataSourceType struct{}

func (t restoresDataSourceType) GetSchema(ctx context.Context) (tfsdk.Schema, diag.Diagnostics) {
	return tfsdk.Schema{
		MarkdownDescription: "restores data source",
		Attributes: map[string]tfsdk.Attribute{
			"id": {
				MarkdownDescription: "data source ID.",
				Computed:            true,
				Type:                types.Int64Type,
			},
			"project_id": {
				MarkdownDescription: "The ID of the project. You can get the project ID from [tidbcloud_project datasource](../project).",
				Required:            true,
				Type:                types.StringType,
			},
			"page": {
				MarkdownDescription: "Default:1 The number of pages.",
				Optional:            true,
				Computed:            true,
				Type:                types.Int64Type,
			},
			"page_size": {
				MarkdownDescription: "Default:10 The size of a pages.",
				Optional:            true,
				Computed:            true,
				Type:                types.Int64Type,
			},
			"items": {
				MarkdownDescription: "Default:10 The size of a pages.",
				Computed:            true,
				Attributes: tfsdk.ListNestedAttributes(map[string]tfsdk.Attribute{
					"id": {
						MarkdownDescription: "The ID of the restore task.",
						Computed:            true,
						Type:                types.StringType,
					},
					"create_timestamp": {
						MarkdownDescription: "The creation time of the backup in UTC.The time format follows the ISO8601 standard, which is YYYY-MM-DD (year-month-day) + T +HH:MM:SS (hour-minutes-seconds) + Z. For example, 2020-01-01T00:00:00Z.",
						Computed:            true,
						Type:                types.StringType,
					},
					"backup_id": {
						MarkdownDescription: "The ID of the backup.",
						Computed:            true,
						Type:                types.StringType,
					},
					"cluster_id": {
						MarkdownDescription: "The cluster ID of the backup.",
						Computed:            true,
						Type:                types.StringType,
					},
					"status": {
						MarkdownDescription: "Enum: \"PENDING\" \"RUNNING\" \"FAILED\" \"SUCCESS\", The status of the restore task.",
						Computed:            true,
						Type:                types.StringType,
					},
					"error_message": {
						MarkdownDescription: "The error message of restore if failed.",
						Computed:            true,
						Type:                types.StringType,
					},
					"cluster": {
						MarkdownDescription: "The information of the restored cluster. The restored cluster is the new cluster your backup data is restored to.",
						Computed:            true,
						PlanModifiers: tfsdk.AttributePlanModifiers{
							resource.UseStateForUnknown(),
						},
						Attributes: tfsdk.SingleNestedAttributes(map[string]tfsdk.Attribute{
							"id": {
								MarkdownDescription: "The ID of the restored cluster. The restored cluster is the new cluster your backup data is restored to.",
								Computed:            true,
								Type:                types.StringType,
								PlanModifiers: tfsdk.AttributePlanModifiers{
									resource.UseStateForUnknown(),
								},
							},
							"name": {
								MarkdownDescription: "The name of the restored cluster. The restored cluster is the new cluster your backup data is restored to.",
								Computed:            true,
								Type:                types.StringType,
								PlanModifiers: tfsdk.AttributePlanModifiers{
									resource.UseStateForUnknown(),
								},
							},
							"status": {
								MarkdownDescription: "The status of the restored cluster. Possible values are \"AVAILABLE\", \"CREATING\", \"MODIFYING\", \"PAUSED\", \"RESUMING\", and \"CLEARED\".",
								Computed:            true,
								Type:                types.StringType,
								PlanModifiers: tfsdk.AttributePlanModifiers{
									resource.UseStateForUnknown(),
								},
							},
						}),
					},
				}),
			},
			"total": {
				MarkdownDescription: "The total number of restore tasks in the project.",
				Computed:            true,
				Type:                types.Int64Type,
			},
		},
	}, nil
}

func (t restoresDataSourceType) NewDataSource(ctx context.Context, in provider.Provider) (datasource.DataSource, diag.Diagnostics) {
	provider, diags := convertProviderType(in)

	return restoresDataSource{
		provider: provider,
	}, diags
}

type restoresDataSource struct {
	provider tidbcloudProvider
}

func (d restoresDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data restoresDataSourceData
	diags := req.Config.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// set default value
	if data.Page.IsNull() || data.Page.IsUnknown() {
		data.Page = types.Int64{Value: 1}
	}
	if data.PageSize.IsNull() || data.PageSize.IsUnknown() {
		data.PageSize = types.Int64{Value: 10}
	}

	tflog.Trace(ctx, "read restores data source")
	restores, err := d.provider.client.GetRestoreTasks(data.ProjectId, data.Page.Value, data.PageSize.Value)
	if err != nil {
		resp.Diagnostics.AddError("Read Error", fmt.Sprintf("Unable to call GetRestoreTasks, got error: %s", err))
		return
	}

	data.Total = types.Int64{Value: restores.Total}
	var items []restore
	for _, key := range restores.Items {
		items = append(items, restore{
			Id:              key.Id,
			CreateTimestamp: key.CreateTimestamp,
			BackupId:        key.BackupId,
			ClusterId:       key.ClusterId,
			ErrorMessage:    key.ErrorMessage,
			Status:          key.Status,
			Cluster: cluster{
				Id:     key.Cluster.Id,
				Name:   key.Cluster.Name,
				Status: key.Cluster.Status,
			},
		})
	}
	data.Items = items
	data.Id = types.Int64{Value: rand.Int63()}

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}
