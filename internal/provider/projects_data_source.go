package provider

import (
	"context"
	"fmt"
	projectApi "github.com/c4pt0r/go-tidbcloud-sdk-v1/client/project"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"math/rand"
	"strconv"
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

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &projectsDataSource{}

type projectsDataSource struct {
	provider *tidbcloudProvider
}

func NewProjectsDataSource() datasource.DataSource {
	return &projectsDataSource{}
}

func (d *projectsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_projects"
}

func (d *projectsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	var ok bool
	if d.provider, ok = req.ProviderData.(*tidbcloudProvider); !ok {
		resp.Diagnostics.AddError("Internal provider error",
			fmt.Sprintf("Error in Configure: expected %T but got %T", tidbcloudProvider{}, req.ProviderData))
	}
}

func (d *projectsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "projects data source",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "data source ID.",
				Computed:            true,
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
				MarkdownDescription: "The items of accessible projects.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							MarkdownDescription: "The ID of the project.",
							Computed:            true,
						},
						"org_id": schema.StringAttribute{
							MarkdownDescription: "The ID of the TiDB Cloud organization to which the project belongs.",
							Computed:            true,
						},
						"name": schema.StringAttribute{
							MarkdownDescription: "The name of the project.",
							Computed:            true,
						},
						"cluster_count": schema.Int64Attribute{
							MarkdownDescription: "The number of TiDB Cloud clusters deployed in the project.",
							Computed:            true,
						},
						"user_count": schema.Int64Attribute{
							MarkdownDescription: "The number of users in the project.",
							Computed:            true,
						},
						"create_timestamp": schema.StringAttribute{
							MarkdownDescription: "The creation time of the cluster in Unix timestamp seconds (epoch time).",
							Computed:            true,
						},
					},
				},
			},
			"total": schema.Int64Attribute{
				MarkdownDescription: "The total number of accessible projects.",
				Computed:            true,
			},
		},
	}
}

func (d *projectsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	if d.provider == nil || !d.provider.configured {
		resp.Diagnostics.AddError(
			"Provider not configured",
			"The provider hasn't been configured before apply, likely because it depends on an unknown value from another resource. This leads to weird stuff happening, so we'd prefer if you didn't do that. Thanks!",
		)
		return
	}

	var data projectsDataSourceData
	diags := req.Config.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// set default value
	var page int64 = 1
	var pageSize int64 = 10
	if !data.Page.IsNull() && !data.Page.IsUnknown() {
		page = data.Page.ValueInt64()
	}
	if !data.PageSize.IsNull() && !data.PageSize.IsUnknown() {
		pageSize = data.PageSize.ValueInt64()
	}

	tflog.Trace(ctx, "read projects data source")
	listProjectsOK, err := d.provider.client.ListProjects(projectApi.NewListProjectsParams().WithPage(&page).WithPageSize(&pageSize))
	if err != nil {
		resp.Diagnostics.AddError("Read Error", fmt.Sprintf("Unable to call read project, got error: %s", err))
		return
	}

	data.Total = types.Int64Value(*listProjectsOK.Payload.Total)
	var items []project
	for _, key := range listProjectsOK.Payload.Items {
		items = append(items, project{
			Id:              key.ID,
			OrgId:           key.OrgID,
			Name:            key.Name,
			ClusterCount:    key.ClusterCount,
			UserCount:       key.UserCount,
			CreateTimestamp: key.CreateTimestamp,
		})
	}
	data.Projects = items
	data.Id = types.StringValue(strconv.FormatInt(rand.Int63(), 10))

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}
