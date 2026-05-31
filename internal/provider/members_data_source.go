package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/juju/errors"
	"github.com/tidbcloud/terraform-provider-tidbcloud/tidbcloud"
)

type membersDataSourceData struct {
	Members []memberItem `tfsdk:"members"`
}

type memberItem struct {
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

var _ datasource.DataSource = &membersDataSource{}

type membersDataSource struct {
	provider *tidbcloudProvider
}

func NewMembersDataSource() datasource.DataSource {
	return &membersDataSource{}
}

func (d *membersDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_members"
}

func (d *membersDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	var ok bool
	if d.provider, ok = req.ProviderData.(*tidbcloudProvider); !ok {
		resp.Diagnostics.AddError("Internal provider error",
			fmt.Sprintf("Error in Configure: expected %T but got %T", tidbcloudProvider{}, req.ProviderData))
	}
}

func (d *membersDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "members data source lists all members of the organization.",
		Attributes: map[string]schema.Attribute{
			"members": schema.ListNestedAttribute{
				MarkdownDescription: "The members of the organization.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"email": schema.StringAttribute{
							MarkdownDescription: "The email address of the member.",
							Computed:            true,
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
				},
			},
		},
	}
}

func (d *membersDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data membersDataSourceData
	diags := req.Config.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Trace(ctx, "read members data source")
	members, err := d.retrieveMembers(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Read Error", fmt.Sprintf("Unable to call ListMembers, got error: %s", err))
		return
	}

	var items []memberItem
	for i := range members {
		member := members[i]
		item := memberItem{
			Email:         types.StringPointerValue(member.Email),
			UserId:        types.StringPointerValue(member.UserId),
			Status:        types.StringPointerValue(member.Status),
			FirstName:     types.StringPointerValue(member.FirstName),
			LastName:      types.StringPointerValue(member.LastName),
			InviteTime:    types.StringPointerValue(member.InviteTime),
			LastLoginTime: types.StringPointerValue(member.LastLoginTime),
			ProjectRoles:  rolesToModel(member.ProjectRoles),
			InstanceRoles: rolesToModel(member.InstanceRoles),
		}
		if member.OrgRole != nil {
			item.OrgRole = types.StringValue(member.OrgRole.RbacRole)
		}
		items = append(items, item)
	}
	data.Members = items

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (d *membersDataSource) retrieveMembers(ctx context.Context) ([]tidbcloud.OpenApiUser, error) {
	var items []tidbcloud.OpenApiUser

	pageSize := int32(DefaultPageSize)
	var pageToken *string
	for {
		rsp, err := d.provider.IAMClient.ListMembers(ctx, &tidbcloud.ListMembersParams{
			PageSize:  &pageSize,
			PageToken: pageToken,
		})
		if err != nil {
			return nil, errors.Trace(err)
		}
		items = append(items, rsp.Users...)

		pageToken = rsp.NextPageToken
		if IsNilOrEmpty(pageToken) {
			break
		}
	}
	return items, nil
}
