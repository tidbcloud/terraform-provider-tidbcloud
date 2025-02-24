package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type serverlessSQLUser struct {
	ClusterId   types.String `tfsdk:"cluster_id"`
	AuthMethod  types.String `tfsdk:"auth_method"`
	UserName    types.String `tfsdk:"user_name"`
	BuiltinRole types.String `tfsdk:"builtin_role"`
	CustomRoles types.List   `tfsdk:"custom_roles"`
}

var _ datasource.DataSource = &serverlessSQLUserDataSource{}

type serverlessSQLUserDataSource struct {
	provider *tidbcloudProvider
}

func NewServerlessSQLUserDataSource() datasource.DataSource {
	return &serverlessSQLUserDataSource{}
}

func (d *serverlessSQLUserDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_serverless_sql_user"
}

func (d *serverlessSQLUserDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	var ok bool
	if d.provider, ok = req.ProviderData.(*tidbcloudProvider); !ok {
		resp.Diagnostics.AddError("Internal provider error",
			fmt.Sprintf("Error in Configure: expected %T but got %T", tidbcloudProvider{}, req.ProviderData))
	}
}

func (d *serverlessSQLUserDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "serverless sql user data source",
		Attributes: map[string]schema.Attribute{
			"cluster_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the cluster.",
				Required:            true,
			},
			"auth_method": schema.StringAttribute{
				MarkdownDescription: "The authentication method of the user.",
				Computed:            true,
			},
			"user_name": schema.StringAttribute{
				MarkdownDescription: "The name of the user.",
				Required:            true,
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
	}
}

func (d *serverlessSQLUserDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data serverlessSQLUser
	diags := req.Config.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Trace(ctx, "read serverless sql user data source")
	sqlUser, err := d.provider.IAMClient.GetSQLUser(ctx, data.ClusterId.ValueString(), data.UserName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Read Error", fmt.Sprintf("Unable to call GetSQLUser, got error: %s", err))
		return
	}

	data.AuthMethod = types.StringValue(*sqlUser.AuthMethod)
	data.BuiltinRole = types.StringValue(*sqlUser.BuiltinRole)
	customRoles, diags := types.ListValueFrom(ctx, types.StringType, sqlUser.CustomRoles)
	if diags.HasError() {
		return
	}
	data.CustomRoles = customRoles

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}
