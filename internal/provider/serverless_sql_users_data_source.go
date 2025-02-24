package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/juju/errors"
	"github.com/tidbcloud/tidbcloud-cli/pkg/tidbcloud/v1beta1/iam"
)

type serverlessSQLUsersDataSourceData struct {
	ClusterId types.String            `tfsdk:"cluster_id"`
	SQLUsers  []serverlessSQLUserItem `tfsdk:"sql_users"`
}

type serverlessSQLUserItem struct {
	AuthMethod  types.String `tfsdk:"auth_method"`
	UserName    types.String `tfsdk:"user_name"`
	BuiltinRole types.String `tfsdk:"builtin_role"`
	CustomRoles types.List   `tfsdk:"custom_roles"`
}

var _ datasource.DataSource = &serverlessSQLUsersDataSource{}

type serverlessSQLUsersDataSource struct {
	provider *tidbcloudProvider
}

func NewServerlessSQLUsersDataSource() datasource.DataSource {
	return &serverlessSQLUsersDataSource{}
}

func (d *serverlessSQLUsersDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_serverless_sql_users"
}

func (d *serverlessSQLUsersDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	var ok bool
	if d.provider, ok = req.ProviderData.(*tidbcloudProvider); !ok {
		resp.Diagnostics.AddError("Internal provider error",
			fmt.Sprintf("Error in Configure: expected %T but got %T", tidbcloudProvider{}, req.ProviderData))
	}
}

func (d *serverlessSQLUsersDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "serverless sql users data source",
		Attributes: map[string]schema.Attribute{
			"cluster_id": schema.StringAttribute{
				MarkdownDescription: "The id of the cluster where the users are.",
				Required:            true,
			},
			"sql_users": schema.ListNestedAttribute{
				MarkdownDescription: "The regions.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"auth_method": schema.StringAttribute{
							MarkdownDescription: "The authentication method of the user.",
							Computed:            true,
						},
						"user_name": schema.StringAttribute{
							MarkdownDescription: "The name of the user.",
							Computed:            true,
						},
						"builtin_role": schema.StringAttribute{
							MarkdownDescription: "The built-in role of the user.",
							Computed:            true,
						},
						"custom_roles": schema.ListAttribute{
							MarkdownDescription: "The custom roles of the user.",
							Computed:            true,
							ElementType:         types.StringType,
						},
					},
				},
			},
		},
	}
}

func (d *serverlessSQLUsersDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data serverlessSQLUsersDataSourceData
	diags := req.Config.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Trace(ctx, "read serverless sql users data source")
	users, err := d.RetrieveSQLUsers(ctx, data.ClusterId.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Read Error", fmt.Sprintf("Unable to call ListClusters, got error: %s", err))
		return
	}
	var items []serverlessSQLUserItem
	for _, user := range users {
		var u serverlessSQLUserItem
		customRoles, diags := types.ListValueFrom(ctx, types.StringType, user.CustomRoles)
		if diags.HasError() {
			return
		}
		u.CustomRoles = customRoles
		u.AuthMethod = types.StringValue(*user.AuthMethod)
		u.UserName = types.StringValue(*user.UserName)
		u.BuiltinRole = types.StringValue(*user.BuiltinRole)
		items = append(items, u)
	}
	data.SQLUsers = items

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (d *serverlessSQLUsersDataSource) RetrieveSQLUsers(ctx context.Context, clusterId string) ([]iam.ApiSqlUser, error) {
	var items []iam.ApiSqlUser

	pageSizeInt32 := int32(DefaultPageSize)
	var pageToken *string
	// loop to get all SQL users
	for {
		sqlUsers, err := d.provider.IAMClient.ListSQLUsers(ctx, clusterId, &pageSizeInt32, pageToken)
		if err != nil {
			return nil, errors.Trace(err)
		}
		items = append(items, sqlUsers.SqlUsers...)

		pageToken = sqlUsers.NextPageToken
		if IsNilOrEmpty(pageToken) {
			break
		}
	}
	return items, nil
}
