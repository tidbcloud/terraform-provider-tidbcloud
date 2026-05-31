package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type memberDataSourceData struct {
	Email         types.String      `tfsdk:"email"`
	OrgRole       types.String      `tfsdk:"org_role"`
	ProjectRoles  []memberRoleModel `tfsdk:"project_roles"`
	InstanceRoles []memberRoleModel `tfsdk:"instance_roles"`
	UserId        types.String      `tfsdk:"user_id"`
	Status        types.String      `tfsdk:"status"`
	FirstName     types.String      `tfsdk:"first_name"`
	LastName      types.String      `tfsdk:"last_name"`
	InviteTime    types.String      `tfsdk:"invite_time"`
	LastLoginTime types.String      `tfsdk:"last_login_time"`
}

var _ datasource.DataSource = &memberDataSource{}

type memberDataSource struct {
	provider *tidbcloudProvider
}

func NewMemberDataSource() datasource.DataSource {
	return &memberDataSource{}
}

func (d *memberDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_member"
}

func (d *memberDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	var ok bool
	if d.provider, ok = req.ProviderData.(*tidbcloudProvider); !ok {
		resp.Diagnostics.AddError("Internal provider error",
			fmt.Sprintf("Error in Configure: expected %T but got %T", tidbcloudProvider{}, req.ProviderData))
	}
}

func memberRoleComputedAttribute(mdDescription string) schema.ListNestedAttribute {
	return schema.ListNestedAttribute{
		MarkdownDescription: mdDescription,
		Computed:            true,
		NestedObject: schema.NestedAttributeObject{
			Attributes: map[string]schema.Attribute{
				"rbac_role": schema.StringAttribute{
					MarkdownDescription: "The RBAC role assigned.",
					Computed:            true,
				},
				"scope_id": schema.StringAttribute{
					MarkdownDescription: "The ID of the project or instance to which the role applies.",
					Computed:            true,
				},
			},
		},
	}
}

func (d *memberDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "member data source",
		Attributes: map[string]schema.Attribute{
			"email": schema.StringAttribute{
				MarkdownDescription: "The email address of the member.",
				Required:            true,
			},
			"org_role": schema.StringAttribute{
				MarkdownDescription: "The organization-level role assigned to the member.",
				Computed:            true,
			},
			"project_roles":  memberRoleComputedAttribute("The project-level roles assigned to the member."),
			"instance_roles": memberRoleComputedAttribute("The instance-level roles assigned to the member."),
			"user_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the member.",
				Computed:            true,
			},
			"status": schema.StringAttribute{
				MarkdownDescription: "The current status of the member.",
				Computed:            true,
			},
			"first_name": schema.StringAttribute{
				MarkdownDescription: "The first name of the member.",
				Computed:            true,
			},
			"last_name": schema.StringAttribute{
				MarkdownDescription: "The last name of the member.",
				Computed:            true,
			},
			"invite_time": schema.StringAttribute{
				MarkdownDescription: "The timestamp when the member was invited, in ISO 8601 format.",
				Computed:            true,
			},
			"last_login_time": schema.StringAttribute{
				MarkdownDescription: "The timestamp when the member last logged in, in ISO 8601 format.",
				Computed:            true,
			},
		},
	}
}

func (d *memberDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data memberDataSourceData
	diags := req.Config.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Trace(ctx, "read member data source")
	member, err := findMemberByEmail(ctx, d.provider.IAMClient, data.Email.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Read Error", fmt.Sprintf("Unable to call ListMembers, got error: %s", err))
		return
	}
	if member == nil {
		resp.Diagnostics.AddError("Read Error", fmt.Sprintf("Member with email %q was not found", data.Email.ValueString()))
		return
	}

	data.UserId = types.StringPointerValue(member.UserId)
	data.Status = types.StringPointerValue(member.Status)
	data.FirstName = types.StringPointerValue(member.FirstName)
	data.LastName = types.StringPointerValue(member.LastName)
	data.InviteTime = types.StringPointerValue(member.InviteTime)
	data.LastLoginTime = types.StringPointerValue(member.LastLoginTime)
	if member.OrgRole != nil {
		data.OrgRole = types.StringValue(member.OrgRole.RbacRole)
	}
	data.ProjectRoles = rolesToModel(member.ProjectRoles)
	data.InstanceRoles = rolesToModel(member.InstanceRoles)

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}
