package provider

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/tidbcloud/tidbcloud-cli/pkg/tidbcloud/v1beta1/iam"
)

const (
	MYSQLNATIVEPASSWORD = "mysql_native_password"
)

type sqlUserResourceData struct {
	ClusterId   types.String `tfsdk:"cluster_id"`
	AuthMethod  types.String `tfsdk:"auth_method"`
	UserName    types.String `tfsdk:"user_name"`
	BuiltinRole types.String `tfsdk:"builtin_role"`
	CustomRoles types.List   `tfsdk:"custom_roles"`
	Password    types.String `tfsdk:"password"`
}

type sqlUserResource struct {
	provider *tidbcloudProvider
}

func NewSQLUserResource() resource.Resource {
	return &sqlUserResource{}
}

func (r *sqlUserResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_sql_user"
}

func (r *sqlUserResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *sqlUserResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "sql user resource",
		Attributes: map[string]schema.Attribute{
			"cluster_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the cluster.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"auth_method": schema.StringAttribute{
				MarkdownDescription: "The authentication method of the user. Only mysql_native_password is supported.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"user_name": schema.StringAttribute{
				MarkdownDescription: "The name of the user. The user name must start with user_prefix for serverless cluster",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"builtin_role": schema.StringAttribute{
				MarkdownDescription: "The built-in role of the sql user, available values [role_admin, role_readonly, role_readwrite]. The built-in role [role_readonly, role_readwrite] must start with user_prefix for serverless cluster",
				Required:            true,
			},
			"custom_roles": schema.ListAttribute{
				MarkdownDescription: "The custom roles of the user.",
				Optional:            true,
				ElementType:         types.StringType,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.UseStateForUnknown(),
				},
			},
			"password": schema.StringAttribute{
				MarkdownDescription: "The password of the user.",
				Required:            true,
				Sensitive:           true,
			},
		},
	}
}

func (r sqlUserResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if !r.provider.configured {
		resp.Diagnostics.AddError(
			"Provider not configured",
			"The provider hasn't been configured before apply, likely because it depends on an unknown value from another resource. This leads to weird stuff happening, so we'd prefer if you didn't do that. Thanks!",
		)
		return
	}

	// get data from config
	var data sqlUserResourceData
	diags := req.Config.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Trace(ctx, "create sql_user_resource")
	body, err := buildCreateSQLUserBody(ctx, data)
	if err != nil {
		resp.Diagnostics.AddError("Create Error", fmt.Sprintf("Unable to build CreateSQLUser body, got error: %s", err))
		return
	}
	_, err = r.provider.IAMClient.CreateSQLUser(ctx, data.ClusterId.ValueString(), &body)
	if err != nil {
		resp.Diagnostics.AddError("Create Error", fmt.Sprintf("Unable to call CreateSQLUser, got error: %s", err))
		return
	}

	// save into the Terraform state.
	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (r sqlUserResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// get data from state
	var data sqlUserResourceData
	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Debug(ctx, fmt.Sprintf("read sql_user_resource clusterid: %s", data.ClusterId.ValueString()))

	// call read api
	tflog.Trace(ctx, "read sql_user_resource")
	sqlUser, err := r.provider.IAMClient.GetSQLUser(ctx, data.ClusterId.ValueString(), data.UserName.ValueString())
	if err != nil {
		tflog.Error(ctx, fmt.Sprintf("Unable to call CetSQLUser, error: %s", err))
		return
	}
	data.BuiltinRole = types.StringValue(*sqlUser.BuiltinRole)
	data.CustomRoles, diags = types.ListValueFrom(ctx, types.StringType, sqlUser.CustomRoles)
	if diags.HasError() {
		resp.Diagnostics.AddError("Read Error", "Unable to convert custom roles, got error")
		return
	}
	// save into the Terraform state.
	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (r sqlUserResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var clusterId string
	var userName string

	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("cluster_id"), &clusterId)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("user_name"), &userName)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Trace(ctx, "delete sql_user_resource")
	_, err := r.provider.IAMClient.DeleteSQLUser(ctx, clusterId, userName)
	if err != nil {
		resp.Diagnostics.AddError("Delete Error", fmt.Sprintf("Unable to call DeleteCluster, got error: %s", err))
		return
	}
}

func (r sqlUserResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// get plan
	var plan sqlUserResourceData
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	// get state
	var state sqlUserResourceData
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	builtinRole := plan.BuiltinRole.ValueString()
	password := plan.Password.ValueString()
	var customRoles []string
	diag := plan.CustomRoles.ElementsAs(ctx, &customRoles, false)
	if diag.HasError() {
		resp.Diagnostics.AddError("Update Error", "Unable to convert custom roles")
	}
	body := &iam.ApiUpdateSqlUserReq{
		BuiltinRole: &builtinRole,
		CustomRoles: customRoles,
		Password:    &password,
	}

	// call update api
	tflog.Trace(ctx, "update sql_user_resource")
	_, err := r.provider.IAMClient.UpdateSQLUser(ctx, state.ClusterId.ValueString(), state.UserName.ValueString(), body)
	if err != nil {
		resp.Diagnostics.AddError("Update Error", fmt.Sprintf("Unable to call UpdateSQLUser, got error: %s", err))
		return
	}

	state.BuiltinRole = plan.BuiltinRole
	state.CustomRoles = plan.CustomRoles
	state.Password = plan.Password

	// save into the Terraform state.
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

func (r sqlUserResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	idParts := strings.Split(req.ID, ",")

	if len(idParts) != 2 || idParts[0] == "" || idParts[1] == "" {
		resp.Diagnostics.AddError(
			"Unexpected Import Identifier",
			fmt.Sprintf("Expected import identifier with format: cluster_id, user_name. Got: %q", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("cluster_id"), idParts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("user_name"), idParts[1])...)
}

func buildCreateSQLUserBody(ctx context.Context, data sqlUserResourceData) (iam.ApiCreateSqlUserReq, error) {
	userName := data.UserName.ValueString()
	var authMethod string
	if data.AuthMethod.IsUnknown() || data.AuthMethod.IsNull() {
		authMethod = MYSQLNATIVEPASSWORD
	} else {
		authMethod = data.AuthMethod.ValueString()
	}
	builtinRole := data.BuiltinRole.ValueString()
	password := data.Password.ValueString()
	var customRoles []string
	diag := data.CustomRoles.ElementsAs(ctx, &customRoles, false)
	if diag.HasError() {
		return iam.ApiCreateSqlUserReq{}, errors.New("unable to convert custom roles")
	}
	autoPrefix := false
	body := iam.ApiCreateSqlUserReq{
		UserName:    &userName,
		AuthMethod:  &authMethod,
		BuiltinRole: &builtinRole,
		CustomRoles: customRoles,
		Password:    &password,
		AutoPrefix:  &autoPrefix,
	}

	return body, nil
}
