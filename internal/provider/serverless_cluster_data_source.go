package provider

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	clusterV1beta1 "github.com/tidbcloud/tidbcloud-cli/pkg/tidbcloud/v1beta1/serverless/cluster"
)

type serverlessCluster struct {
	ClusterId             types.String           `tfsdk:"cluster_id"`
	DisplayName           types.String           `tfsdk:"display_name"`
	Region                *region                `tfsdk:"region"`
	SpendingLimit         *spendingLimit         `tfsdk:"spending_limit"`
	AutoScaling           *autoScaling           `tfsdk:"auto_scaling"`
	AutomatedBackupPolicy *automatedBackupPolicy `tfsdk:"automated_backup_policy"`
	Endpoints             *endpoints             `tfsdk:"endpoints"`
	EncryptionConfig      *encryptionConfig      `tfsdk:"encryption_config"`
	Version               types.String           `tfsdk:"version"`
	CreatedBy             types.String           `tfsdk:"created_by"`
	CreateTime            types.String           `tfsdk:"create_time"`
	UpdateTime            types.String           `tfsdk:"update_time"`
	UserPrefix            types.String           `tfsdk:"user_prefix"`
	State                 types.String           `tfsdk:"state"`
	Labels                types.Map              `tfsdk:"labels"`
	Annotations           types.Map              `tfsdk:"annotations"`
}

var _ datasource.DataSource = &serverlessClusterDataSource{}

type serverlessClusterDataSource struct {
	provider *tidbcloudProvider
}

func NewServerlessClusterDataSource() datasource.DataSource {
	return &serverlessClusterDataSource{}
}

func (d *serverlessClusterDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_serverless_cluster"
}

func (d *serverlessClusterDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	var ok bool
	if d.provider, ok = req.ProviderData.(*tidbcloudProvider); !ok {
		resp.Diagnostics.AddError("Internal provider error",
			fmt.Sprintf("Error in Configure: expected %T but got %T", tidbcloudProvider{}, req.ProviderData))
	}
}

func (d *serverlessClusterDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "serverless cluster data source",
		Attributes: map[string]schema.Attribute{
			"cluster_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the cluster.",
				Required:            true,
			},
			"display_name": schema.StringAttribute{
				MarkdownDescription: "The display name of the cluster.",
				Computed:            true,
			},
			"region": schema.SingleNestedAttribute{
				MarkdownDescription: "The region of the cluster.",
				Computed:            true,
				Attributes: map[string]schema.Attribute{
					"name": schema.StringAttribute{
						MarkdownDescription: "The unique name of the region.",
						Computed:            true,
					},
					"region_id": schema.StringAttribute{
						MarkdownDescription: "The ID of the region.",
						Computed:            true,
					},
					"cloud_provider": schema.StringAttribute{
						MarkdownDescription: "The cloud provider of the region.",
						Computed:            true,
					},
					"display_name": schema.StringAttribute{
						MarkdownDescription: "The display name of the region.",
						Computed:            true,
					},
				},
			},
			"spending_limit": schema.SingleNestedAttribute{
				MarkdownDescription: "The spending limit of the cluster.",
				Computed:            true,
				Attributes: map[string]schema.Attribute{
					"monthly": schema.Int32Attribute{
						MarkdownDescription: "Maximum monthly spending limit in USD cents.",
						Computed:            true,
					},
				},
			},
			"auto_scaling": schema.SingleNestedAttribute{
				MarkdownDescription: "The auto scaling configuration of the cluster.",
				Computed:            true,
				Attributes: map[string]schema.Attribute{
					"min_rcu": schema.Int64Attribute{
						MarkdownDescription: "The minimum RCU (Request Capacity Unit) of the cluster.",
						Computed:            true,
					},
					"max_rcu": schema.Int64Attribute{
						MarkdownDescription: "The maximum RCU (Request Capacity Unit) of the cluster.",
						Computed:            true,
					},
				},
			},
			"automated_backup_policy": schema.SingleNestedAttribute{
				MarkdownDescription: "The automated backup policy of the cluster.",
				Computed:            true,
				Attributes: map[string]schema.Attribute{
					"start_time": schema.StringAttribute{
						MarkdownDescription: "The time of day when the automated backup will start.",
						Computed:            true,
					},
					"retention_days": schema.Int32Attribute{
						MarkdownDescription: "The number of days to retain automated backups.",
						Computed:            true,
					},
				},
			},
			"endpoints": schema.SingleNestedAttribute{
				MarkdownDescription: "The endpoints for connecting to the cluster.",
				Computed:            true,
				Attributes: map[string]schema.Attribute{
					"public": schema.SingleNestedAttribute{
						MarkdownDescription: "The public endpoint for connecting to the cluster.",
						Computed:            true,
						Attributes: map[string]schema.Attribute{
							"host": schema.StringAttribute{
								MarkdownDescription: "The host of the public endpoint.",
								Computed:            true,
							},
							"port": schema.Int32Attribute{
								MarkdownDescription: "The port of the public endpoint.",
								Computed:            true,
							},
							"disabled": schema.BoolAttribute{
								MarkdownDescription: "Whether the public endpoint is disabled.",
								Computed:            true,
							},
						},
					},
					"private": schema.SingleNestedAttribute{
						MarkdownDescription: "The private endpoint for connecting to the cluster.",
						Computed:            true,
						Attributes: map[string]schema.Attribute{
							"host": schema.StringAttribute{
								MarkdownDescription: "The host of the private endpoint.",
								Computed:            true,
							},
							"port": schema.Int32Attribute{
								MarkdownDescription: "The port of the private endpoint.",
								Computed:            true,
							},
							"aws": schema.SingleNestedAttribute{
								MarkdownDescription: "Message for AWS PrivateLink information.",
								Computed:            true,
								Attributes: map[string]schema.Attribute{
									"service_name": schema.StringAttribute{
										MarkdownDescription: "The AWS service name for private access.",
										Computed:            true,
									},
									"availability_zone": schema.ListAttribute{
										MarkdownDescription: "The availability zones that the service is available in.",
										Computed:            true,
										ElementType:         types.StringType,
									},
								},
							},
						},
					},
				},
			},
			"encryption_config": schema.SingleNestedAttribute{
				MarkdownDescription: "The encryption settings for the cluster.",
				Computed:            true,
				Attributes: map[string]schema.Attribute{
					"enhanced_encryption_enabled": schema.BoolAttribute{
						MarkdownDescription: "Whether enhanced encryption is enabled.",
						Computed:            true,
					},
				},
			},
			"version": schema.StringAttribute{
				MarkdownDescription: "The version of the cluster.",
				Computed:            true,
			},
			"created_by": schema.StringAttribute{
				MarkdownDescription: "The email of the creator of the cluster.",
				Computed:            true,
			},
			"create_time": schema.StringAttribute{
				MarkdownDescription: "The time the cluster was created.",
				Computed:            true,
			},
			"update_time": schema.StringAttribute{
				MarkdownDescription: "The time the cluster was last updated.",
				Computed:            true,
			},
			"user_prefix": schema.StringAttribute{
				MarkdownDescription: "The unique prefix in SQL user name.",
				Computed:            true,
			},
			"state": schema.StringAttribute{
				MarkdownDescription: "The state of the cluster.",
				Computed:            true,
			},
			"labels": schema.MapAttribute{
				MarkdownDescription: "The labels of the cluster.",
				Computed:            true,
				ElementType:         types.StringType,
			},
			"annotations": schema.MapAttribute{
				MarkdownDescription: "The annotations of the cluster.",
				Computed:            true,
				ElementType:         types.StringType,
			},
		},
	}
}

func (d *serverlessClusterDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data serverlessCluster
	diags := req.Config.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Trace(ctx, "read serverless cluster data source")
	cluster, err := d.provider.ServerlessClient.GetCluster(ctx, data.ClusterId.ValueString(), clusterV1beta1.CLUSTERSERVICEGETCLUSTERVIEWPARAMETER_FULL)
	if err != nil {
		resp.Diagnostics.AddError("Read Error", fmt.Sprintf("Unable to call GetCluster, got error: %s", err))
		return
	}

	labels, diag := types.MapValueFrom(ctx, types.StringType, *cluster.Labels)
	if diag.HasError() {
		diags.AddError("Read Error", "unable to convert labels")
		return
	}
	annotations, diag := types.MapValueFrom(ctx, types.StringType, *cluster.Annotations)
	if diag.HasError() {
		diags.AddError("Read Error", "unable to convert annotations")
		return
	}
	data.ClusterId = types.StringValue(*cluster.ClusterId)
	data.DisplayName = types.StringValue(cluster.DisplayName)

	r := cluster.Region
	data.Region = &region{
		Name:          types.StringValue(*r.Name),
		RegionId:      types.StringValue(*r.RegionId),
		CloudProvider: types.StringValue(string(*r.CloudProvider)),
		DisplayName:   types.StringValue(*r.DisplayName),
	}

	if cluster.SpendingLimit != nil {
		data.SpendingLimit = &spendingLimit{
			Monthly: types.Int32Value(*cluster.SpendingLimit.Monthly),
		}
	}

	if cluster.AutoScaling != nil {
		data.AutoScaling = &autoScaling{
			MinRCU: types.Int64Value(*cluster.AutoScaling.MinRcu),
			MaxRCU: types.Int64Value(*cluster.AutoScaling.MaxRcu),
		}
	}

	a := cluster.AutomatedBackupPolicy
	data.AutomatedBackupPolicy = &automatedBackupPolicy{
		StartTime:     types.StringValue(*a.StartTime),
		RetentionDays: types.Int32Value(*a.RetentionDays),
	}

	e := cluster.Endpoints
	var pe private
	if e.Private.Aws != nil {
		awsAvailabilityZone, diags := types.ListValueFrom(ctx, types.StringType, e.Private.Aws.AvailabilityZone)
		if diags.HasError() {
			return
		}
		pe = private{
			Host: types.StringValue(*e.Private.Host),
			Port: types.Int32Value(*e.Private.Port),
			AWS: &aws{
				ServiceName:      types.StringValue(*e.Private.Aws.ServiceName),
				AvailabilityZone: awsAvailabilityZone,
			},
		}
	}

	data.Endpoints = &endpoints{
		Public: &public{
			Host:     types.StringValue(*e.Public.Host),
			Port:     types.Int32Value(*e.Public.Port),
			Disabled: types.BoolValue(*e.Public.Disabled),
		},
		Private: &pe,
	}

	en := cluster.EncryptionConfig
	data.EncryptionConfig = &encryptionConfig{
		EnhancedEncryptionEnabled: types.BoolValue(*en.EnhancedEncryptionEnabled),
	}

	data.Version = types.StringValue(*cluster.Version)
	data.CreatedBy = types.StringValue(*cluster.CreatedBy)
	data.CreateTime = types.StringValue(cluster.CreateTime.Format(time.RFC3339))
	data.UpdateTime = types.StringValue(cluster.UpdateTime.Format(time.RFC3339))
	data.UserPrefix = types.StringValue(*cluster.UserPrefix)
	data.State = types.StringValue(string(*cluster.State))
	data.Labels = labels
	data.Annotations = annotations

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}
