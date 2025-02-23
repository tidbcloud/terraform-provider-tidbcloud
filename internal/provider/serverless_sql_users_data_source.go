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
	sqlUsers []serverlessSQLUser `tfsdk:"sql_users"`
}

type serverlessSQLUser struct {
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
	clusters, err := d.RetrieveSQLUsers(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Read Error", fmt.Sprintf("Unable to call ListClusters, got error: %s", err))
		return
	}
	var items []serverlessClusterItem
	for _, cluster := range clusters {
		var c serverlessClusterItem
		labels, diag := types.MapValueFrom(ctx, types.StringType, *cluster.Labels)
		if diag.HasError() {
			return
		}
		annotations, diag := types.MapValueFrom(ctx, types.StringType, *cluster.Annotations)
		if diag.HasError() {
			return
		}
		c.ClusterId = types.StringValue(*cluster.ClusterId)
		c.DisplayName = types.StringValue(cluster.DisplayName)

		r := cluster.Region
		c.Region = &region{
			Name:          types.StringValue(*r.Name),
			RegionId:      types.StringValue(*r.RegionId),
			CloudProvider: types.StringValue(string(*r.CloudProvider)),
			DisplayName:   types.StringValue(*r.DisplayName),
		}

		e := cluster.Endpoints
		var pe privateEndpoint
		if e.Private.Aws != nil {
			awsAvailabilityZone, diags := types.ListValueFrom(ctx, types.StringType, e.Private.Aws.AvailabilityZone)
			if diags.HasError() {
				return
			}
			pe = privateEndpoint{
				Host: types.StringValue(*e.Private.Host),
				Port: types.Int64Value(int64(*e.Private.Port)),
				AWSEndpoint: &awsEndpoint{
					ServiceName:      types.StringValue(*e.Private.Aws.ServiceName),
					AvailabilityZone: awsAvailabilityZone,
				},
			}
		}

		if e.Private.Gcp != nil {
			pe = privateEndpoint{
				Host: types.StringValue(*e.Private.Host),
				Port: types.Int64Value(int64(*e.Private.Port)),
				GCPEndpoint: &gcpEndpoint{
					ServiceAttachmentName: types.StringValue(*e.Private.Gcp.ServiceAttachmentName),
				},
			}
		}

		c.Endpoints = &endpoints{
			PublicEndpoint: publicEndpoint{
				Host:     types.StringValue(*e.Public.Host),
				Port:     types.Int64Value(int64(*e.Public.Port)),
				Disabled: types.BoolValue(*e.Public.Disabled),
			},
			PrivateEndpoint: pe,
		}

		en := cluster.EncryptionConfig
		c.EncryptionConfig = &encryptionConfig{
			EnhancedEncryptionEnabled: types.BoolValue(*en.EnhancedEncryptionEnabled),
		}

		c.HighAvailabilityType = types.StringValue(string(*cluster.HighAvailabilityType))
		c.Version = types.StringValue(*cluster.Version)
		c.CreatedBy = types.StringValue(*cluster.CreatedBy)
		c.CreateTime = types.StringValue(cluster.CreateTime.String())
		c.UpdateTime = types.StringValue(cluster.UpdateTime.String())
		c.UserPrefix = types.StringValue(*cluster.UserPrefix)
		c.State = types.StringValue(string(*cluster.State))

		c.Labels = labels
		c.Annotations = annotations
		items = append(items, c)
	}
	data.Clusters = items

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (d *serverlessSQLUsersDataSource) RetrieveSQLUsers(ctx context.Context, clusterId string) ([]iam.ApiSqlUser, error) {
	var items []iam.ApiSqlUser

	pageSizeInt32 := int32(DefaultPageSize)
	var pageToken *string
	// loop to get all SQL users
	for {
		sqlUsers, err := d.ListSQLUsers(ctx, &pageSizeInt32, pageToken)
		if err != nil {
			return nil, errors.Trace(err)
		}
		items = append(items, sqlUsers.SqlUsers...)

		pageToken = sqlUsers.NextPageToken
		if util.IsNilOrEmpty(pageToken) {
			break
		}
	}
	return items, nil
}
