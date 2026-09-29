package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type dedicatedChangefeedDataSourceData struct {
	ChangefeedId        types.String                  `tfsdk:"changefeed_id"`
	ClusterId           types.String                  `tfsdk:"cluster_id"`
	Name                types.String                  `tfsdk:"name"`
	ReplicationCapacity types.String                  `tfsdk:"replication_capacity"`
	DownstreamType      types.String                  `tfsdk:"downstream_type"`
	NetworkInfo         *changefeedNetworkInfoModel   `tfsdk:"network_info"`
	StartPosition       *changefeedStartPositionModel `tfsdk:"start_position"`
	TableConfig         *changefeedTableConfigModel   `tfsdk:"table_config"`
	Kafka               *changefeedKafkaModel         `tfsdk:"kafka"`
	Mysql               *changefeedMysqlModel         `tfsdk:"mysql"`
	State               types.String                  `tfsdk:"state"`
	CreateTime          types.String                  `tfsdk:"create_time"`
	CheckpointTso       types.String                  `tfsdk:"checkpoint_tso"`
	CheckpointTs        types.String                  `tfsdk:"checkpoint_ts"`
	Error               types.String                  `tfsdk:"error"`
}

var _ datasource.DataSource = &dedicatedChangefeedDataSource{}

type dedicatedChangefeedDataSource struct {
	provider *tidbcloudProvider
}

func NewDedicatedChangefeedDataSource() datasource.DataSource {
	return &dedicatedChangefeedDataSource{}
}

func (d *dedicatedChangefeedDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dedicated_changefeed"
}

func (d *dedicatedChangefeedDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	var ok bool
	if d.provider, ok = req.ProviderData.(*tidbcloudProvider); !ok {
		resp.Diagnostics.AddError("Internal provider error",
			fmt.Sprintf("Error in Configure: expected %T but got %T", tidbcloudProvider{}, req.ProviderData))
	}
}

func (d *dedicatedChangefeedDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "dedicated changefeed data source",
		Attributes: map[string]schema.Attribute{
			"changefeed_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the changefeed.",
				Required:            true,
			},
			"cluster_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the cluster.",
				Computed:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The user-defined name of the changefeed.",
				Computed:            true,
			},
			"replication_capacity": schema.StringAttribute{
				MarkdownDescription: "The replication capacity (RCU) of the changefeed.",
				Computed:            true,
			},
			"downstream_type": schema.StringAttribute{
				MarkdownDescription: "The downstream type of the changefeed.",
				Computed:            true,
			},
			"network_info": schema.SingleNestedAttribute{
				MarkdownDescription: "The network configuration for the downstream connection.",
				Computed:            true,
				Attributes: map[string]schema.Attribute{
					"network_type": schema.StringAttribute{
						MarkdownDescription: "The network type.",
						Computed:            true,
					},
					"sink_endpoint_id": schema.StringAttribute{
						MarkdownDescription: "The private endpoint ID configured in the TiDB Cloud console.",
						Computed:            true,
					},
					"ports": schema.ListAttribute{
						MarkdownDescription: "The downstream target ports for the `NETWORK_TYPE_PRIVATE_LINK` network type.",
						Computed:            true,
						ElementType:         types.Int64Type,
					},
				},
			},
			"start_position": schema.SingleNestedAttribute{
				MarkdownDescription: "The start position for the changefeed.",
				Computed:            true,
				Attributes: map[string]schema.Attribute{
					"mode": schema.StringAttribute{
						MarkdownDescription: "The start position mode.",
						Computed:            true,
					},
					"start_tso": schema.StringAttribute{
						MarkdownDescription: "The start TSO.",
						Computed:            true,
					},
					"start_timestamp": schema.StringAttribute{
						MarkdownDescription: "The start UTC timestamp.",
						Computed:            true,
					},
				},
			},
			"table_config": schema.SingleNestedAttribute{
				MarkdownDescription: "The table filtering and event filter configuration.",
				Computed:            true,
				Attributes: map[string]schema.Attribute{
					"filter_rules": schema.ListAttribute{
						MarkdownDescription: "The table filter rules.",
						Computed:            true,
						ElementType:         types.StringType,
					},
					"mode": schema.StringAttribute{
						MarkdownDescription: "The mode for handling unsupported tables.",
						Computed:            true,
					},
					"case_sensitive": schema.BoolAttribute{
						MarkdownDescription: "Whether the filter rules are case-sensitive.",
						Computed:            true,
					},
					"event_filters": schema.ListNestedAttribute{
						MarkdownDescription: "The event filter rules for fine-grained control.",
						Computed:            true,
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"table_matchers": schema.ListAttribute{
									MarkdownDescription: "The table name patterns to match.",
									Computed:            true,
									ElementType:         types.StringType,
								},
								"ignored_events": schema.ListAttribute{
									MarkdownDescription: "The event types to ignore.",
									Computed:            true,
									ElementType:         types.StringType,
								},
								"ignored_sql_statements": schema.ListAttribute{
									MarkdownDescription: "The SQL statement patterns to ignore.",
									Computed:            true,
									ElementType:         types.StringType,
								},
								"ignored_insert_value_expression": schema.StringAttribute{
									MarkdownDescription: "The SQL expression to filter INSERT events by column value.",
									Computed:            true,
								},
								"ignored_update_old_value_expression": schema.StringAttribute{
									MarkdownDescription: "The SQL expression to filter UPDATE events by the old column value.",
									Computed:            true,
								},
								"ignored_update_new_value_expression": schema.StringAttribute{
									MarkdownDescription: "The SQL expression to filter UPDATE events by the new column value.",
									Computed:            true,
								},
								"ignored_delete_value_expression": schema.StringAttribute{
									MarkdownDescription: "The SQL expression to filter DELETE events by column value.",
									Computed:            true,
								},
							},
						},
					},
				},
			},
			"kafka": schema.SingleNestedAttribute{
				MarkdownDescription: "The Kafka downstream configuration. The SASL password is input-only and never returned by the API.",
				Computed:            true,
				Attributes: map[string]schema.Attribute{
					"broker": schema.SingleNestedAttribute{
						MarkdownDescription: "The Kafka broker connection configuration.",
						Computed:            true,
						Attributes: map[string]schema.Attribute{
							"version": schema.StringAttribute{
								MarkdownDescription: "The Kafka broker version.",
								Computed:            true,
							},
							"broker_endpoints": schema.StringAttribute{
								MarkdownDescription: "The comma-separated list of Kafka broker addresses.",
								Computed:            true,
							},
							"use_tls": schema.BoolAttribute{
								MarkdownDescription: "Whether to use TLS for the Kafka connection.",
								Computed:            true,
							},
							"insecure_skip_verify": schema.BoolAttribute{
								MarkdownDescription: "Whether to skip TLS certificate verification.",
								Computed:            true,
							},
							"compression": schema.StringAttribute{
								MarkdownDescription: "The Kafka message compression type.",
								Computed:            true,
							},
						},
					},
					"authentication": schema.SingleNestedAttribute{
						MarkdownDescription: "The Kafka authentication configuration.",
						Computed:            true,
						Attributes: map[string]schema.Attribute{
							"auth_type": schema.StringAttribute{
								MarkdownDescription: "The Kafka authentication type.",
								Computed:            true,
							},
							"username": schema.StringAttribute{
								MarkdownDescription: "The SASL username for Kafka authentication.",
								Computed:            true,
							},
							"password": schema.StringAttribute{
								MarkdownDescription: "The SASL password for Kafka authentication. Always null: the API never returns it.",
								Computed:            true,
								Sensitive:           true,
							},
						},
					},
					"data_format": schema.SingleNestedAttribute{
						MarkdownDescription: "The data format and serialization configuration.",
						Computed:            true,
						Attributes: map[string]schema.Attribute{
							"protocol": schema.StringAttribute{
								MarkdownDescription: "The Kafka output protocol.",
								Computed:            true,
							},
							"enable_tidb_extension": schema.BoolAttribute{
								MarkdownDescription: "Whether to enable TiDB extension fields.",
								Computed:            true,
							},
							"output_raw_change_event": schema.BoolAttribute{
								MarkdownDescription: "Whether to output raw change events.",
								Computed:            true,
							},
							"debezium_config": schema.SingleNestedAttribute{
								MarkdownDescription: "The Debezium-specific configuration.",
								Computed:            true,
								Attributes: map[string]schema.Attribute{
									"output_old_value": schema.BoolAttribute{
										MarkdownDescription: "Whether to output the old value before the change.",
										Computed:            true,
									},
									"disable_schema": schema.BoolAttribute{
										MarkdownDescription: "Whether to disable the schema in Debezium messages.",
										Computed:            true,
									},
								},
							},
						},
					},
					"topic_partition_config": schema.SingleNestedAttribute{
						MarkdownDescription: "The topic and partition configuration.",
						Computed:            true,
						Attributes: map[string]schema.Attribute{
							"dispatch_type": schema.StringAttribute{
								MarkdownDescription: "The message dispatch type to topics.",
								Computed:            true,
							},
							"default_topic": schema.StringAttribute{
								MarkdownDescription: "The default topic for dispatching all messages.",
								Computed:            true,
							},
							"topic_prefix": schema.StringAttribute{
								MarkdownDescription: "The topic name prefix.",
								Computed:            true,
							},
							"separator": schema.StringAttribute{
								MarkdownDescription: "The separator between the prefix and the table or database name.",
								Computed:            true,
							},
							"topic_suffix": schema.StringAttribute{
								MarkdownDescription: "The topic name suffix.",
								Computed:            true,
							},
							"replication_factor": schema.Int64Attribute{
								MarkdownDescription: "The replication factor for auto-created topics.",
								Computed:            true,
							},
							"partition_num": schema.Int64Attribute{
								MarkdownDescription: "The number of partitions for auto-created topics.",
								Computed:            true,
							},
							"partition_dispatchers": schema.ListNestedAttribute{
								MarkdownDescription: "The custom partition dispatcher configurations.",
								Computed:            true,
								NestedObject: schema.NestedAttributeObject{
									Attributes: map[string]schema.Attribute{
										"partition_type": schema.StringAttribute{
											MarkdownDescription: "The partition dispatch strategy.",
											Computed:            true,
										},
										"matcher": schema.ListAttribute{
											MarkdownDescription: "The table name patterns to match for the dispatcher.",
											Computed:            true,
											ElementType:         types.StringType,
										},
										"index_name": schema.StringAttribute{
											MarkdownDescription: "The index name for the `INDEX_VALUE` partition dispatcher.",
											Computed:            true,
										},
										"columns": schema.ListAttribute{
											MarkdownDescription: "The column names for the `COLUMNS` partition dispatcher.",
											Computed:            true,
											ElementType:         types.StringType,
										},
									},
								},
							},
						},
					},
					"column_selectors": schema.ListNestedAttribute{
						MarkdownDescription: "The column selectors for filtering specific columns.",
						Computed:            true,
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"matcher": schema.ListAttribute{
									MarkdownDescription: "The table name patterns to match for column selection.",
									Computed:            true,
									ElementType:         types.StringType,
								},
								"columns": schema.ListAttribute{
									MarkdownDescription: "The column names to include for the matched tables.",
									Computed:            true,
									ElementType:         types.StringType,
								},
							},
						},
					},
				},
			},
			"mysql": schema.SingleNestedAttribute{
				MarkdownDescription: "The MySQL downstream configuration. The password is input-only and never returned by the API.",
				Computed:            true,
				Attributes: map[string]schema.Attribute{
					"connection": schema.SingleNestedAttribute{
						MarkdownDescription: "The MySQL connection configuration.",
						Computed:            true,
						Attributes: map[string]schema.Attribute{
							"endpoint": schema.StringAttribute{
								MarkdownDescription: "The MySQL endpoint address in the `host:port` format.",
								Computed:            true,
							},
							"username": schema.StringAttribute{
								MarkdownDescription: "The MySQL username for authentication.",
								Computed:            true,
							},
							"password": schema.StringAttribute{
								MarkdownDescription: "The MySQL password for authentication. Always null: the API never returns it.",
								Computed:            true,
								Sensitive:           true,
							},
						},
					},
				},
			},
			"state": schema.StringAttribute{
				MarkdownDescription: "The current state of the changefeed.",
				Computed:            true,
			},
			"create_time": schema.StringAttribute{
				MarkdownDescription: "The creation time of the changefeed.",
				Computed:            true,
			},
			"checkpoint_tso": schema.StringAttribute{
				MarkdownDescription: "The checkpoint TSO of the changefeed.",
				Computed:            true,
			},
			"checkpoint_ts": schema.StringAttribute{
				MarkdownDescription: "The checkpoint timestamp of the changefeed.",
				Computed:            true,
			},
			"error": schema.StringAttribute{
				MarkdownDescription: "The error message when the changefeed is in the `FAILED` or `ERROR` state.",
				Computed:            true,
			},
		},
	}
}

func (d *dedicatedChangefeedDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data dedicatedChangefeedDataSourceData
	diags := req.Config.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Trace(ctx, "read dedicated changefeed data source")
	changefeed, err := d.provider.DedicatedClient.GetChangefeed(ctx, data.ChangefeedId.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Read Error", fmt.Sprintf("Unable to call GetChangefeed, got error: %s", err))
		return
	}

	data.ChangefeedId = types.StringPointerValue(changefeed.Id)
	data.ClusterId = types.StringValue(changefeed.ClusterId)
	data.Name = types.StringValue(changefeed.Name)
	data.ReplicationCapacity = types.StringValue(changefeed.ReplicationCapacity)
	data.DownstreamType = types.StringValue(changefeed.DownstreamType)
	data.NetworkInfo = networkInfoAPIToModel(ctx, changefeed.NetworkInfo)
	data.StartPosition = startPositionAPIToModel(changefeed.StartPosition)
	data.TableConfig = tableConfigAPIToModel(ctx, changefeed.TableConfig)
	data.Kafka = kafkaAPIToModel(ctx, changefeed.Kafka)
	data.Mysql = mysqlAPIToModel(changefeed.Mysql)
	data.State = types.StringPointerValue(changefeed.State)
	data.CreateTime = types.StringPointerValue(changefeed.CreateTime)
	data.CheckpointTso = types.StringPointerValue(changefeed.CheckpointTso)
	data.CheckpointTs = types.StringPointerValue(changefeed.CheckpointTs)
	data.Error = types.StringPointerValue(changefeed.Error)

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}
