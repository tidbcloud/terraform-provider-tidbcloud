package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/juju/errors"
	clusterV1beta1 "github.com/tidbcloud/tidbcloud-cli/pkg/tidbcloud/v1beta1/serverless/cluster"
)

const DefaultPageSize = 100

type serverlessClustersDataSourceData struct {
	ProjectId types.String            `tfsdk:"project_id"`
	Clusters  []serverlessClusterItem `tfsdk:"clusters"`
}

type serverlessClusterItem struct {
	ClusterId            types.String      `tfsdk:"cluster_id"`
	DisplayName          types.String      `tfsdk:"display_name"`
	Region               *region           `tfsdk:"region"`
	Endpoints            *endpoints        `tfsdk:"endpoints"`
	EncryptionConfig     *encryptionConfig `tfsdk:"encryption_config"`
	Version              types.String      `tfsdk:"version"`
	CreatedBy            types.String      `tfsdk:"created_by"`
	CreateTime           types.String      `tfsdk:"create_time"`
	UpdateTime           types.String      `tfsdk:"update_time"`
	UserPrefix           types.String      `tfsdk:"user_prefix"`
	State                types.String      `tfsdk:"state"`
	Labels               types.Map         `tfsdk:"labels"`
	Annotations          types.Map         `tfsdk:"annotations"`
}

var _ datasource.DataSource = &serverlessClustersDataSource{}

type serverlessClustersDataSource struct {
	provider *tidbcloudProvider
}

func NewServerlessClustersDataSource() datasource.DataSource {
	return &serverlessClustersDataSource{}
}

func (d *serverlessClustersDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_serverless_clusters"
}

func (d *serverlessClustersDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	var ok bool
	if d.provider, ok = req.ProviderData.(*tidbcloudProvider); !ok {
		resp.Diagnostics.AddError("Internal provider error",
			fmt.Sprintf("Error in Configure: expected %T but got %T", tidbcloudProvider{}, req.ProviderData))
	}
}

func (d *serverlessClustersDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "serverless clusters data source",
		Attributes: map[string]schema.Attribute{
			"project_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the project that the clusters belong to.",
				Optional:            true,
			},
			"clusters": schema.ListNestedAttribute{
				MarkdownDescription: "The clusters.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"cluster_id": schema.StringAttribute{
							MarkdownDescription: "The ID of the cluster.",
							Computed:            true,
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
				},
			},
		},
	}
}

func (d *serverlessClustersDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data serverlessClustersDataSourceData
	diags := req.Config.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Trace(ctx, "read serverless clusters data source")
	clusters, err := d.retrieveClusters(ctx, data.ProjectId.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Read Error", fmt.Sprintf("Unable to call ListClusters, got error: %s", err))
		return
	}
	var items []serverlessClusterItem
	for _, cluster := range clusters {
		var c serverlessClusterItem
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
		var pe private
		if e.Private.Aws != nil {
			awsAvailabilityZone, diag := types.ListValueFrom(ctx, types.StringType, e.Private.Aws.AvailabilityZone)
			if diag.HasError() {
				diags.AddError("Read Error", "unable to convert aws availability zone")
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

		c.Endpoints = &endpoints{
			Public: &public{
				Host:     types.StringValue(*e.Public.Host),
				Port:     types.Int32Value(*e.Public.Port),
				Disabled: types.BoolValue(*e.Public.Disabled),
			},
			Private: &pe,
		}

		en := cluster.EncryptionConfig
		c.EncryptionConfig = &encryptionConfig{
			EnhancedEncryptionEnabled: types.BoolValue(*en.EnhancedEncryptionEnabled),
		}

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

func (d *serverlessClustersDataSource) retrieveClusters(ctx context.Context, projectId string) ([]clusterV1beta1.TidbCloudOpenApiserverlessv1beta1Cluster, error) {
	var items []clusterV1beta1.TidbCloudOpenApiserverlessv1beta1Cluster
	pageSizeInt32 := int32(DefaultPageSize)
	var pageToken *string
	var filter *string
	if projectId != "" {
		projectFilter := fmt.Sprintf("projectId=%s", projectId)
		filter = &projectFilter
	}

	clusters, err := d.provider.ServerlessClient.ListClusters(ctx, filter, &pageSizeInt32, nil, nil, nil)
	if err != nil {
		return nil, errors.Trace(err)
	}
	items = append(items, clusters.Clusters...)
	for {
		pageToken = clusters.NextPageToken
		if IsNilOrEmpty(pageToken) {
			break
		}
		clusters, err = d.provider.ServerlessClient.ListClusters(ctx, filter, &pageSizeInt32, pageToken, nil, nil)
		if err != nil {
			return nil, errors.Trace(err)
		}
		items = append(items, clusters.Clusters...)
	}
	return items, nil
}
