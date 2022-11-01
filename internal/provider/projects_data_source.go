package provider

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type projectsDataSourceData struct {
	Id       types.String `tfsdk:"id"`
	Page     types.Int64  `tfsdk:"page"`
	PageSize types.Int64  `tfsdk:"page_size"`
	Projects []project    `tfsdk:"items"`
	Total    types.Int64  `tfsdk:"total"`
}

type project struct {
	Id              string `tfsdk:"id"`
	OrgId           string `tfsdk:"org_id"`
	Name            string `tfsdk:"name"`
	ClusterCount    int64  `tfsdk:"cluster_count"`
	UserCount       int64  `tfsdk:"user_count"`
	CreateTimestamp string `tfsdk:"create_timestamp"`
}

// Ensure provider defined types fully satisfy framework interfaces
var _ provider.DataSourceType = projectsDataSourceType{}
var _ datasource.DataSource = projectsDataSource{}

type projectsDataSourceType struct{}

func (t projectsDataSourceType) GetSchema(ctx context.Context) (tfsdk.Schema, diag.Diagnostics) {
	return tfsdk.Schema{
		MarkdownDescription: "projects data source",
		Attributes: map[string]tfsdk.Attribute{
			"id": {
				MarkdownDescription: "ignore it, it is just for test.",
				Computed:            true,
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
				MarkdownDescription: "The items of accessible projects.",
				Computed:            true,
				Attributes: tfsdk.ListNestedAttributes(map[string]tfsdk.Attribute{
					"id": {
						MarkdownDescription: "The ID of the project.",
						Computed:            true,
						Type:                types.StringType,
					},
					"org_id": {
						MarkdownDescription: "The ID of the TiDB Cloud organization to which the project belongs.",
						Computed:            true,
						Type:                types.StringType,
					},
					"name": {
						MarkdownDescription: "The name of the project.",
						Computed:            true,
						Type:                types.StringType,
					},
					"cluster_count": {
						MarkdownDescription: "The number of TiDB Cloud clusters deployed in the project.",
						Computed:            true,
						Type:                types.Int64Type,
					},
					"user_count": {
						MarkdownDescription: "The number of users in the project.",
						Computed:            true,
						Type:                types.Int64Type,
					},
					"create_timestamp": {
						MarkdownDescription: "The creation time of the cluster in Unix timestamp seconds (epoch time).",
						Computed:            true,
						Type:                types.StringType,
					},
				}),
			},
			"total": {
				MarkdownDescription: "The total number of accessible projects.",
				Computed:            true,
				Type:                types.Int64Type,
			},
		},
	}, nil
}

func (t projectsDataSourceType) NewDataSource(ctx context.Context, in provider.Provider) (datasource.DataSource, diag.Diagnostics) {
	provider, diags := convertProviderType(in)

	return projectsDataSource{
		provider: provider,
	}, diags
}

type projectsDataSource struct {
	provider tidbcloudProvider
}

func (d projectsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data projectsDataSourceData
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

	tflog.Trace(ctx, "read project data source")
	projects, err := d.provider.client.GetAllProjects(data.Page.Value, data.PageSize.Value)
	if err != nil {
		resp.Diagnostics.AddError("Read Error", fmt.Sprintf("Unable to call read project, got error: %s", err))
		return
	}

	data.Total = types.Int64{Value: projects.Total}
	var items []project
	for _, key := range projects.Items {
		items = append(items, project{
			Id:              key.Id,
			OrgId:           key.OrgId,
			Name:            key.Name,
			ClusterCount:    key.ClusterCount,
			UserCount:       key.UserCount,
			CreateTimestamp: key.CreateTimestamp,
		})
	}
	data.Projects = items
	data.Id = types.String{Value: "just for test"}

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}
