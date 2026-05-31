package provider

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/tidbcloud/terraform-provider-tidbcloud/tidbcloud"
)

type memberRoleModel struct {
	RbacRole types.String `tfsdk:"rbac_role"`
	ScopeId  types.String `tfsdk:"scope_id"`
}

type memberResourceData struct {
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

type memberResource struct {
	provider *tidbcloudProvider
}

func NewMemberResource() resource.Resource {
	return &memberResource{}
}

func (r *memberResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_member"
}

func (r *memberResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	var ok bool
	if r.provider, ok = req.ProviderData.(*tidbcloudProvider); !ok {
		resp.Diagnostics.AddError("Internal provider error",
			fmt.Sprintf("Error in Configure: expected %T but got %T", tidbcloudProvider{}, req.ProviderData))
	}
}

// roleNestedAttribute is a Set (not List): role assignments are semantically
// unordered, so a Set avoids spurious diffs when the API returns roles in a
// different order than configured.
func roleNestedAttribute(mdDescription string) schema.SetNestedAttribute {
	return schema.SetNestedAttribute{
		MarkdownDescription: mdDescription,
		Optional:            true,
		NestedObject: schema.NestedAttributeObject{
			Attributes: map[string]schema.Attribute{
				"rbac_role": schema.StringAttribute{
					MarkdownDescription: "The RBAC role to assign.",
					Required:            true,
				},
				"scope_id": schema.StringAttribute{
					MarkdownDescription: "The ID of the project or instance to which the role applies.",
					Required:            true,
				},
			},
		},
	}
}

func (r *memberResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "member resource manages an organization member in TiDB Cloud. Creating the resource invites the member; the member stays in `Pending` status until the invitation is accepted.",
		Attributes: map[string]schema.Attribute{
			"email": schema.StringAttribute{
				MarkdownDescription: "The email address of the member. Changing this forces a new member to be invited.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"org_role": schema.StringAttribute{
				MarkdownDescription: "The organization-level role assigned to the member. Available values: `org:owner`, `org:member`, `org:billing_admin`, `org:billing_viewer`, `org:audit_admin`.",
				Required:            true,
			},
			"project_roles":  roleNestedAttribute("The project-level roles assigned to the member. `rbac_role` available values: `project:owner`, `project:dev`, `project:readonly`, `project:ctl_plane_viewer`. The existing project roles are fully replaced on update."),
			"instance_roles": roleNestedAttribute("The instance-level roles assigned to the member. `rbac_role` available values: `cluster:admin`, `cluster:viewer`. The existing instance roles are fully replaced on update."),
			"user_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the member.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"status": schema.StringAttribute{
				MarkdownDescription: "The current status of the member. `Active` indicates the member has accepted the invitation; `Pending` indicates the invitation has not yet been accepted.",
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

func (r memberResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if !r.provider.configured {
		resp.Diagnostics.AddError(
			"Provider not configured",
			"The provider hasn't been configured before apply, likely because it depends on an unknown value from another resource. This leads to weird stuff happening, so we'd prefer if you didn't do that. Thanks!",
		)
		return
	}

	var data memberResourceData
	diags := req.Config.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Trace(ctx, "create member_resource")
	body := &tidbcloud.OpenApiInviteUsersReq{
		Emails:        []string{data.Email.ValueString()},
		OrgRole:       &tidbcloud.OpenApiRbacRole{RbacRole: data.OrgRole.ValueString()},
		ProjectRoles:  buildRoles(data.ProjectRoles),
		InstanceRoles: buildRoles(data.InstanceRoles),
	}
	inviteRsp, err := r.provider.IAMClient.InviteMembers(ctx, body)
	if err != nil {
		resp.Diagnostics.AddError("Create Error", fmt.Sprintf("Unable to call InviteMembers, got error: %s", err))
		return
	}
	if inviteRsp != nil && inviteRsp.Success != nil && !*inviteRsp.Success {
		msg := ""
		if inviteRsp.Message != nil {
			msg = *inviteRsp.Message
		}
		resp.Diagnostics.AddError("Create Error", fmt.Sprintf("InviteMembers reported failure for %q: %s", data.Email.ValueString(), msg))
		return
	}
	invitedUserID := userIDFromInviteResult(inviteRsp, data.Email.ValueString())

	// The invitation was accepted by the API; from here on we must save state
	// (with at least the user_id) even on read failures, otherwise the member
	// would be left unmanaged and re-invited on the next apply.
	member, err := r.readMemberByEmailWithRetry(ctx, data.Email.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Create Error", fmt.Sprintf("Unable to read invited member, got error: %s", err))
		return
	}
	if member == nil {
		if invitedUserID == "" {
			resp.Diagnostics.AddError("Create Error", fmt.Sprintf("The invited member %q was not found after invitation and no user_id was returned", data.Email.ValueString()))
			return
		}
		// Save partial state so the member is tracked; a later refresh will
		// populate the remaining computed fields once the invite is visible.
		data.UserId = types.StringValue(invitedUserID)
		resp.Diagnostics.AddWarning(
			"Member not yet visible",
			fmt.Sprintf("Member %q was invited (user_id %s) but is not yet visible via the API. Run apply again to reconcile its computed attributes.", data.Email.ValueString(), invitedUserID),
		)
		diags = resp.State.Set(ctx, &data)
		resp.Diagnostics.Append(diags...)
		return
	}
	// Keep the user-supplied email and role lists from config; only refresh the
	// computed fields from the API to avoid spurious "inconsistent result" errors.
	applyComputedMemberFields(member, &data)

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (r memberResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data memberResourceData
	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Trace(ctx, "read member_resource")
	member, err := r.readMemberByEmail(ctx, data.Email.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Read Error", fmt.Sprintf("Unable to call ListMembers, got error: %s", err))
		return
	}
	if member == nil {
		// The member no longer exists; remove it from state.
		resp.State.RemoveResource(ctx)
		return
	}
	mapMemberToData(member, &data)

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (r memberResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan memberResourceData
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	var state memberResourceData
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := &tidbcloud.OpenApiUpdateUserReq{
		OrgRole:       &tidbcloud.OpenApiRbacRole{RbacRole: plan.OrgRole.ValueString()},
		ProjectRoles:  buildRoles(plan.ProjectRoles),
		InstanceRoles: buildRoles(plan.InstanceRoles),
	}

	tflog.Trace(ctx, "update member_resource")
	err := r.provider.IAMClient.UpdateMember(ctx, state.UserId.ValueString(), body)
	if err != nil {
		resp.Diagnostics.AddError("Update Error", fmt.Sprintf("Unable to call UpdateMember, got error: %s", err))
		return
	}

	member, err := r.readMemberByEmail(ctx, state.Email.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Update Error", fmt.Sprintf("Unable to read member after update, got error: %s", err))
		return
	}
	if member == nil {
		resp.Diagnostics.AddError("Update Error", fmt.Sprintf("The member %q was not found after update", state.Email.ValueString()))
		return
	}
	applyComputedMemberFields(member, &plan)

	diags = resp.State.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
}

func (r memberResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var userId string
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("user_id"), &userId)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Trace(ctx, "delete member_resource")
	err := r.provider.IAMClient.DeleteMember(ctx, userId)
	// Treat an already-removed member as a successful delete so destroy is
	// idempotent if the member was removed out of band.
	if err != nil && !tidbcloud.IsNotFoundError(err) {
		resp.Diagnostics.AddError("Delete Error", fmt.Sprintf("Unable to call DeleteMember, got error: %s", err))
		return
	}
}

func (r memberResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	email := strings.TrimSpace(req.ID)
	if email == "" {
		resp.Diagnostics.AddError(
			"Unexpected Import Identifier",
			fmt.Sprintf("Expected import identifier to be the member email. Got: %q", req.ID),
		)
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("email"), email)...)
}

// readMemberByEmail lists members filtered by email and returns the exact match,
// or nil if no member with that email exists.
func (r memberResource) readMemberByEmail(ctx context.Context, email string) (*tidbcloud.OpenApiUser, error) {
	return findMemberByEmail(ctx, r.provider.IAMClient, email)
}

const (
	memberCreateReadAttempts = 6
	memberCreateReadInterval = 2 * time.Second
)

// readMemberByEmailWithRetry tolerates eventual consistency: after an invite the
// member may not be immediately visible via List, so it polls a bounded number
// of times before giving up (returning a nil member, not an error).
func (r memberResource) readMemberByEmailWithRetry(ctx context.Context, email string) (*tidbcloud.OpenApiUser, error) {
	for attempt := 0; attempt < memberCreateReadAttempts; attempt++ {
		member, err := r.readMemberByEmail(ctx, email)
		if err != nil {
			return nil, err
		}
		if member != nil {
			return member, nil
		}
		if attempt == memberCreateReadAttempts-1 {
			break
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(memberCreateReadInterval):
		}
	}
	return nil, nil
}

// findMemberByEmail is shared by the resource and the single member data source.
func findMemberByEmail(ctx context.Context, client tidbcloud.TiDBCloudIAMClient, email string) (*tidbcloud.OpenApiUser, error) {
	pageSize := int32(DefaultPageSize)
	var pageToken *string
	for {
		rsp, err := client.ListMembers(ctx, &tidbcloud.ListMembersParams{
			PageSize:  &pageSize,
			PageToken: pageToken,
			Email:     &email,
		})
		if err != nil {
			return nil, err
		}
		for i := range rsp.Users {
			u := rsp.Users[i]
			if u.Email != nil && strings.EqualFold(*u.Email, email) {
				return &u, nil
			}
		}
		pageToken = rsp.NextPageToken
		if IsNilOrEmpty(pageToken) {
			break
		}
	}
	return nil, nil
}

// buildRoles converts the Terraform role models to API roles. It always returns
// a non-nil slice so that, on update, an empty role list serializes as `[]` and
// clears the member's roles (the invite request omits empty role lists via
// omitempty, so this is safe for create too).
func buildRoles(roles []memberRoleModel) []tidbcloud.OpenApiRbacRole {
	out := make([]tidbcloud.OpenApiRbacRole, 0, len(roles))
	for _, role := range roles {
		scopeId := role.ScopeId.ValueString()
		out = append(out, tidbcloud.OpenApiRbacRole{
			RbacRole: role.RbacRole.ValueString(),
			ScopeId:  &scopeId,
		})
	}
	return out
}

// userIDFromInviteResult extracts the invited member's user_id from the bulk
// invite response, matching on email (case-insensitively) and falling back to
// the sole result when present.
func userIDFromInviteResult(rsp *tidbcloud.OpenApiInviteUsersRsp, email string) string {
	if rsp == nil {
		return ""
	}
	for _, u := range rsp.Users {
		if u.Email != nil && strings.EqualFold(*u.Email, email) && u.UserId != nil {
			return *u.UserId
		}
	}
	if len(rsp.Users) == 1 && rsp.Users[0].UserId != nil {
		return *rsp.Users[0].UserId
	}
	return ""
}

func rolesToModel(roles []tidbcloud.OpenApiRbacRole) []memberRoleModel {
	if len(roles) == 0 {
		return nil
	}
	out := make([]memberRoleModel, 0, len(roles))
	for _, role := range roles {
		out = append(out, memberRoleModel{
			RbacRole: types.StringValue(role.RbacRole),
			ScopeId:  types.StringPointerValue(role.ScopeId),
		})
	}
	return out
}

// applyComputedMemberFields refreshes only the API-computed fields, leaving the
// user-managed email, org role, and role lists untouched. Used by Create and
// Update where those values come from config/plan.
func applyComputedMemberFields(member *tidbcloud.OpenApiUser, data *memberResourceData) {
	data.UserId = types.StringPointerValue(member.UserId)
	data.Status = types.StringPointerValue(member.Status)
	data.FirstName = types.StringPointerValue(member.FirstName)
	data.LastName = types.StringPointerValue(member.LastName)
	data.InviteTime = types.StringPointerValue(member.InviteTime)
	data.LastLoginTime = types.StringPointerValue(member.LastLoginTime)
}

// mapMemberToData copies the full API member onto the Terraform model, including
// the org role and role lists. Used by Read to detect drift. The configured
// email is intentionally NOT overwritten: email is a RequiresReplace key, and
// some APIs canonicalize the address (e.g. lowercasing), which would otherwise
// trigger perpetual replacement. The member is matched to state by email
// already, so the stored value is preserved.
func mapMemberToData(member *tidbcloud.OpenApiUser, data *memberResourceData) {
	applyComputedMemberFields(member, data)
	if member.OrgRole != nil {
		data.OrgRole = types.StringValue(member.OrgRole.RbacRole)
	}
	data.ProjectRoles = rolesToModel(member.ProjectRoles)
	data.InstanceRoles = rolesToModel(member.InstanceRoles)
}
