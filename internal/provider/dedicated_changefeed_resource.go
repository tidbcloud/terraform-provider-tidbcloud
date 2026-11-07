package provider

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"github.com/tidbcloud/terraform-provider-tidbcloud/tidbcloud"
)

const (
	changefeedTimeout      = 30 * time.Minute
	changefeedPollInterval = 10 * time.Second
)

var (
	_ resource.Resource               = &dedicatedChangefeedResource{}
	_ resource.ResourceWithModifyPlan = &dedicatedChangefeedResource{}
)

type changefeedNetworkInfoModel struct {
	NetworkType    types.String `tfsdk:"network_type"`
	SinkEndpointId types.String `tfsdk:"sink_endpoint_id"`
	Ports          types.List   `tfsdk:"ports"`
}

type changefeedStartPositionModel struct {
	Mode           types.String `tfsdk:"mode"`
	StartTso       types.String `tfsdk:"start_tso"`
	StartTimestamp types.String `tfsdk:"start_timestamp"`
}

type changefeedEventFilterModel struct {
	TableMatchers                   types.List   `tfsdk:"table_matchers"`
	IgnoredEvents                   types.List   `tfsdk:"ignored_events"`
	IgnoredSqlStatements            types.List   `tfsdk:"ignored_sql_statements"`
	IgnoredInsertValueExpression    types.String `tfsdk:"ignored_insert_value_expression"`
	IgnoredUpdateOldValueExpression types.String `tfsdk:"ignored_update_old_value_expression"`
	IgnoredUpdateNewValueExpression types.String `tfsdk:"ignored_update_new_value_expression"`
	IgnoredDeleteValueExpression    types.String `tfsdk:"ignored_delete_value_expression"`
}

type changefeedTableConfigModel struct {
	FilterRules   types.List                   `tfsdk:"filter_rules"`
	Mode          types.String                 `tfsdk:"mode"`
	CaseSensitive types.Bool                   `tfsdk:"case_sensitive"`
	EventFilters  []changefeedEventFilterModel `tfsdk:"event_filters"`
}

type changefeedKafkaBrokerModel struct {
	Version            types.String `tfsdk:"version"`
	BrokerEndpoints    types.String `tfsdk:"broker_endpoints"`
	UseTls             types.Bool   `tfsdk:"use_tls"`
	InsecureSkipVerify types.Bool   `tfsdk:"insecure_skip_verify"`
	Compression        types.String `tfsdk:"compression"`
}

type changefeedKafkaAuthenticationModel struct {
	AuthType types.String `tfsdk:"auth_type"`
	Username types.String `tfsdk:"username"`
	Password types.String `tfsdk:"password"`
}

type changefeedKafkaDebeziumConfigModel struct {
	OutputOldValue types.Bool `tfsdk:"output_old_value"`
	DisableSchema  types.Bool `tfsdk:"disable_schema"`
}

type changefeedKafkaDataFormatModel struct {
	Protocol             types.String                        `tfsdk:"protocol"`
	EnableTidbExtension  types.Bool                          `tfsdk:"enable_tidb_extension"`
	OutputRawChangeEvent types.Bool                          `tfsdk:"output_raw_change_event"`
	DebeziumConfig       *changefeedKafkaDebeziumConfigModel `tfsdk:"debezium_config"`
}

type changefeedKafkaPartitionDispatcherModel struct {
	PartitionType types.String `tfsdk:"partition_type"`
	Matcher       types.List   `tfsdk:"matcher"`
	IndexName     types.String `tfsdk:"index_name"`
	Columns       types.List   `tfsdk:"columns"`
}

type changefeedKafkaTopicPartitionConfigModel struct {
	DispatchType         types.String                              `tfsdk:"dispatch_type"`
	DefaultTopic         types.String                              `tfsdk:"default_topic"`
	TopicPrefix          types.String                              `tfsdk:"topic_prefix"`
	Separator            types.String                              `tfsdk:"separator"`
	TopicSuffix          types.String                              `tfsdk:"topic_suffix"`
	ReplicationFactor    types.Int64                               `tfsdk:"replication_factor"`
	PartitionNum         types.Int64                               `tfsdk:"partition_num"`
	PartitionDispatchers []changefeedKafkaPartitionDispatcherModel `tfsdk:"partition_dispatchers"`
}

type changefeedKafkaColumnSelectorModel struct {
	Matcher types.List `tfsdk:"matcher"`
	Columns types.List `tfsdk:"columns"`
}

type changefeedKafkaModel struct {
	Broker               *changefeedKafkaBrokerModel               `tfsdk:"broker"`
	Authentication       *changefeedKafkaAuthenticationModel       `tfsdk:"authentication"`
	DataFormat           *changefeedKafkaDataFormatModel           `tfsdk:"data_format"`
	TopicPartitionConfig *changefeedKafkaTopicPartitionConfigModel `tfsdk:"topic_partition_config"`
	ColumnSelectors      []changefeedKafkaColumnSelectorModel      `tfsdk:"column_selectors"`
}

type changefeedMysqlConnectionModel struct {
	Endpoint types.String `tfsdk:"endpoint"`
	Username types.String `tfsdk:"username"`
	Password types.String `tfsdk:"password"`
}

type changefeedMysqlModel struct {
	Connection *changefeedMysqlConnectionModel `tfsdk:"connection"`
}

type dedicatedChangefeedResourceData struct {
	ChangefeedId        types.String                  `tfsdk:"changefeed_id"`
	ClusterId           types.String                  `tfsdk:"cluster_id"`
	Name                types.String                  `tfsdk:"name"`
	ReplicationCapacity types.String                  `tfsdk:"replication_capacity"`
	DownstreamType      types.String                  `tfsdk:"downstream_type"`
	Paused              types.Bool                    `tfsdk:"paused"`
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

type dedicatedChangefeedResource struct {
	provider *tidbcloudProvider
}

func NewDedicatedChangefeedResource() resource.Resource {
	return &dedicatedChangefeedResource{}
}

func (r *dedicatedChangefeedResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dedicated_changefeed"
}

func (r *dedicatedChangefeedResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	var ok bool
	r.provider, ok = req.ProviderData.(*tidbcloudProvider)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *tidbcloudProvider, got: %T", req.ProviderData),
		)
	}
}

func (r *dedicatedChangefeedResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "dedicated changefeed resource manages a changefeed (CDC replication) of a TiDB Cloud Dedicated cluster. Only the `KAFKA` and `MYSQL` downstream types are supported for now. Editing the downstream configuration of a RUNNING changefeed automatically pauses it for the duration of the edit and resumes it afterwards (the API requires the PAUSED state for edits); replication catches up after the resume. Every update is planned against the live state of the changefeed and sequences the pause, resume, scale and edit calls it needs, so a pause-state change can be combined with a `replication_capacity` change or a downstream edit in one apply. A changefeed in the FAILED state rejects all modifications.",
		Attributes: map[string]schema.Attribute{
			"changefeed_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the changefeed.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"cluster_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the cluster.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The user-defined name of the changefeed.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"replication_capacity": schema.StringAttribute{
				MarkdownDescription: "The replication capacity (RCU) of the changefeed, for example `4rcu`. Changing it scales the changefeed via an independent ScaleChangefeed call that requires the RUNNING state. The scale is sequenced with a pause-state change in the same apply (resume, then scale; or scale, then pause). The only combination that is rejected, at plan time, is changing it on a changefeed that is paused and stays paused.",
				Required:            true,
			},
			"downstream_type": schema.StringAttribute{
				MarkdownDescription: "The downstream type of the changefeed. Available values: `KAFKA`, `MYSQL`.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"paused": schema.BoolAttribute{
				MarkdownDescription: "Whether the changefeed is paused. Refreshed from the live changefeed state on every read (`PAUSED`, `PAUSING` and `EDITING` read as `true`), so a pause or resume done outside Terraform shows up as a diff. Leave it unset to not manage the pause state at all: Terraform then never pauses or resumes the changefeed (other than around a downstream-config edit, which it restores) and only reports the live value — useful when operators pause the changefeed by hand, e.g. for downstream maintenance.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"network_info": schema.SingleNestedAttribute{
				MarkdownDescription: "The network configuration for the downstream connection.",
				Required:            true,
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.RequiresReplace(),
				},
				Attributes: map[string]schema.Attribute{
					"network_type": schema.StringAttribute{
						MarkdownDescription: "The network type. Available values: `NETWORK_TYPE_PUBLIC`, `NETWORK_TYPE_VPC_PEERING`, `NETWORK_TYPE_PRIVATE_LINK`.",
						Required:            true,
					},
					"sink_endpoint_id": schema.StringAttribute{
						MarkdownDescription: "The private endpoint ID configured in the TiDB Cloud console. Required for the `NETWORK_TYPE_PRIVATE_LINK` network type.",
						Optional:            true,
					},
					"ports": schema.ListAttribute{
						MarkdownDescription: "The downstream target ports for the `NETWORK_TYPE_PRIVATE_LINK` network type.",
						Optional:            true,
						ElementType:         types.Int64Type,
					},
				},
			},
			"start_position": schema.SingleNestedAttribute{
				MarkdownDescription: "The start position for the changefeed.",
				Required:            true,
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.RequiresReplace(),
				},
				Attributes: map[string]schema.Attribute{
					"mode": schema.StringAttribute{
						MarkdownDescription: "The start position mode. Available values: `FROM_NOW`, `FROM_TSO`, `FROM_UTC`.",
						Required:            true,
					},
					"start_tso": schema.StringAttribute{
						MarkdownDescription: "The start TSO. Required when the mode is `FROM_TSO`.",
						Optional:            true,
					},
					"start_timestamp": schema.StringAttribute{
						MarkdownDescription: "The start UTC timestamp in RFC 3339 format. Required when the mode is `FROM_UTC`.",
						Optional:            true,
					},
				},
			},
			"table_config": schema.SingleNestedAttribute{
				MarkdownDescription: "The table filtering and event filter configuration.",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"filter_rules": schema.ListAttribute{
						MarkdownDescription: "The table filter rules. Uses the TiCDC table filter syntax.",
						Optional:            true,
						ElementType:         types.StringType,
					},
					"mode": schema.StringAttribute{
						MarkdownDescription: "The mode for handling unsupported tables. Available values: `IGNORE_NOT_SUPPORT_TABLE`, `FORCE_SYNC`.",
						Optional:            true,
					},
					"case_sensitive": schema.BoolAttribute{
						MarkdownDescription: "Whether the filter rules are case-sensitive.",
						Optional:            true,
					},
					"event_filters": schema.ListNestedAttribute{
						MarkdownDescription: "The event filter rules for fine-grained control.",
						Optional:            true,
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"table_matchers": schema.ListAttribute{
									MarkdownDescription: "The table name patterns to match. Uses the TiCDC table filter syntax.",
									Optional:            true,
									ElementType:         types.StringType,
								},
								"ignored_events": schema.ListAttribute{
									MarkdownDescription: "The event types to ignore.",
									Optional:            true,
									ElementType:         types.StringType,
								},
								"ignored_sql_statements": schema.ListAttribute{
									MarkdownDescription: "The SQL statement patterns to ignore (DDL only). Supports regular expressions.",
									Optional:            true,
									ElementType:         types.StringType,
								},
								"ignored_insert_value_expression": schema.StringAttribute{
									MarkdownDescription: "The SQL expression to filter INSERT events by column value.",
									Optional:            true,
								},
								"ignored_update_old_value_expression": schema.StringAttribute{
									MarkdownDescription: "The SQL expression to filter UPDATE events by the old column value.",
									Optional:            true,
								},
								"ignored_update_new_value_expression": schema.StringAttribute{
									MarkdownDescription: "The SQL expression to filter UPDATE events by the new column value.",
									Optional:            true,
								},
								"ignored_delete_value_expression": schema.StringAttribute{
									MarkdownDescription: "The SQL expression to filter DELETE events by column value.",
									Optional:            true,
								},
							},
						},
					},
				},
			},
			"kafka": schema.SingleNestedAttribute{
				MarkdownDescription: "The Kafka downstream configuration. Required when `downstream_type` is `KAFKA`.",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"broker": schema.SingleNestedAttribute{
						MarkdownDescription: "The Kafka broker connection configuration.",
						Required:            true,
						Attributes: map[string]schema.Attribute{
							"version": schema.StringAttribute{
								MarkdownDescription: "The Kafka broker version. Available values: `KAFKA_VERSION_0XX`, `KAFKA_VERSION_1XX`, `KAFKA_VERSION_2XX`, `KAFKA_VERSION_3XX`.",
								Required:            true,
							},
							"broker_endpoints": schema.StringAttribute{
								MarkdownDescription: "The comma-separated list of Kafka broker addresses. Required for the `NETWORK_TYPE_PUBLIC` and `NETWORK_TYPE_VPC_PEERING` network types.",
								Optional:            true,
							},
							"use_tls": schema.BoolAttribute{
								MarkdownDescription: "Whether to use TLS for the Kafka connection.",
								Optional:            true,
							},
							"insecure_skip_verify": schema.BoolAttribute{
								MarkdownDescription: "Whether to skip TLS certificate verification.",
								Optional:            true,
							},
							"compression": schema.StringAttribute{
								MarkdownDescription: "The Kafka message compression type. Available values: `NONE`, `GZIP`, `SNAPPY`, `LZ4`, `ZSTD`.",
								Optional:            true,
							},
						},
					},
					"authentication": schema.SingleNestedAttribute{
						MarkdownDescription: "The Kafka authentication configuration.",
						Required:            true,
						Attributes: map[string]schema.Attribute{
							"auth_type": schema.StringAttribute{
								MarkdownDescription: "The Kafka authentication type. Available values: `DISABLE`, `SASL_PLAIN`, `SASL_SCRAM_SHA_256`, `SASL_SCRAM_SHA_512`.",
								Required:            true,
							},
							"username": schema.StringAttribute{
								MarkdownDescription: "The SASL username for Kafka authentication.",
								Optional:            true,
							},
							"password": schema.StringAttribute{
								MarkdownDescription: "The SASL password for Kafka authentication. This field is input-only and never returned by the API.",
								Optional:            true,
								Sensitive:           true,
							},
						},
					},
					"data_format": schema.SingleNestedAttribute{
						MarkdownDescription: "The data format and serialization configuration.",
						Required:            true,
						Attributes: map[string]schema.Attribute{
							"protocol": schema.StringAttribute{
								MarkdownDescription: "The Kafka output protocol. Available values: `PROTOCOL_CANAL_JSON`, `PROTOCOL_OPEN_PROTOCOL`, `PROTOCOL_DEBEZIUM`. (`PROTOCOL_AVRO` is not supported by this resource yet.)",
								Required:            true,
							},
							"enable_tidb_extension": schema.BoolAttribute{
								MarkdownDescription: "Whether to enable TiDB extension fields.",
								Optional:            true,
							},
							"output_raw_change_event": schema.BoolAttribute{
								MarkdownDescription: "Whether to output raw change events.",
								Optional:            true,
							},
							"debezium_config": schema.SingleNestedAttribute{
								MarkdownDescription: "The Debezium-specific configuration.",
								Optional:            true,
								Attributes: map[string]schema.Attribute{
									"output_old_value": schema.BoolAttribute{
										MarkdownDescription: "Whether to output the old value before the change.",
										Optional:            true,
									},
									"disable_schema": schema.BoolAttribute{
										MarkdownDescription: "Whether to disable the schema in Debezium messages.",
										Optional:            true,
									},
								},
							},
						},
					},
					"topic_partition_config": schema.SingleNestedAttribute{
						MarkdownDescription: "The topic and partition configuration.",
						Required:            true,
						Attributes: map[string]schema.Attribute{
							"dispatch_type": schema.StringAttribute{
								MarkdownDescription: "The message dispatch type to topics. Available values: `DISPATCH_TYPE_ONE_TOPIC`, `DISPATCH_TYPE_BY_TABLE`, `DISPATCH_TYPE_BY_DATABASE`.",
								Required:            true,
							},
							"default_topic": schema.StringAttribute{
								MarkdownDescription: "The default topic for dispatching all messages when `dispatch_type` is `DISPATCH_TYPE_ONE_TOPIC`.",
								Optional:            true,
							},
							"topic_prefix": schema.StringAttribute{
								MarkdownDescription: "The topic name prefix for dispatch types that create per-table or per-database topics.",
								Optional:            true,
							},
							"separator": schema.StringAttribute{
								MarkdownDescription: "The separator between the prefix and the table or database name.",
								Optional:            true,
							},
							"topic_suffix": schema.StringAttribute{
								MarkdownDescription: "The topic name suffix for dispatch types that create per-table or per-database topics.",
								Optional:            true,
							},
							"replication_factor": schema.Int64Attribute{
								MarkdownDescription: "The replication factor for auto-created topics.",
								Required:            true,
							},
							"partition_num": schema.Int64Attribute{
								MarkdownDescription: "The number of partitions for auto-created topics.",
								Required:            true,
							},
							"partition_dispatchers": schema.ListNestedAttribute{
								MarkdownDescription: "The custom partition dispatcher configurations.",
								Optional:            true,
								NestedObject: schema.NestedAttributeObject{
									Attributes: map[string]schema.Attribute{
										"partition_type": schema.StringAttribute{
											MarkdownDescription: "The partition dispatch strategy.",
											Required:            true,
										},
										"matcher": schema.ListAttribute{
											MarkdownDescription: "The table name patterns to match for the dispatcher.",
											Optional:            true,
											ElementType:         types.StringType,
										},
										"index_name": schema.StringAttribute{
											MarkdownDescription: "The index name for the `INDEX_VALUE` partition dispatcher.",
											Optional:            true,
										},
										"columns": schema.ListAttribute{
											MarkdownDescription: "The column names for the `COLUMNS` partition dispatcher.",
											Optional:            true,
											ElementType:         types.StringType,
										},
									},
								},
							},
						},
					},
					"column_selectors": schema.ListNestedAttribute{
						MarkdownDescription: "The column selectors for filtering specific columns.",
						Optional:            true,
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"matcher": schema.ListAttribute{
									MarkdownDescription: "The table name patterns to match for column selection.",
									Optional:            true,
									ElementType:         types.StringType,
								},
								"columns": schema.ListAttribute{
									MarkdownDescription: "The column names to include for the matched tables.",
									Optional:            true,
									ElementType:         types.StringType,
								},
							},
						},
					},
				},
			},
			"mysql": schema.SingleNestedAttribute{
				MarkdownDescription: "The MySQL downstream configuration. Required when `downstream_type` is `MYSQL`.",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"connection": schema.SingleNestedAttribute{
						MarkdownDescription: "The MySQL connection configuration.",
						Required:            true,
						Attributes: map[string]schema.Attribute{
							"endpoint": schema.StringAttribute{
								MarkdownDescription: "The MySQL endpoint address in the `host:port` format. Required for the `NETWORK_TYPE_PUBLIC` and `NETWORK_TYPE_VPC_PEERING` network types.",
								Optional:            true,
							},
							"username": schema.StringAttribute{
								MarkdownDescription: "The MySQL username for authentication.",
								Required:            true,
							},
							"password": schema.StringAttribute{
								MarkdownDescription: "The MySQL password for authentication. This field is input-only and never returned by the API.",
								Optional:            true,
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

func (r dedicatedChangefeedResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if !r.provider.configured {
		resp.Diagnostics.AddError(
			"Provider not configured",
			"The provider hasn't been configured before apply, likely because it depends on an unknown value from another resource. This leads to weird stuff happening, so we'd prefer if you didn't do that. Thanks!",
		)
		return
	}

	var data dedicatedChangefeedResourceData
	diags := req.Plan.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := validateChangefeedDownstream(&data); err != nil {
		resp.Diagnostics.AddError("Invalid Configuration", err.Error())
		return
	}

	tflog.Trace(ctx, "create dedicated_changefeed_resource")
	body := &tidbcloud.CreateChangefeedRequest{Changefeed: buildChangefeedBody(ctx, &data, &resp.Diagnostics)}
	if resp.Diagnostics.HasError() {
		return
	}
	changefeed, err := r.provider.DedicatedClient.CreateChangefeed(ctx, body)
	if err != nil {
		resp.Diagnostics.AddError("Create Error", fmt.Sprintf("Unable to call CreateChangefeed, got error: %s", err))
		return
	}
	if changefeed.Id == nil {
		resp.Diagnostics.AddError("Create Error", "CreateChangefeed did not return a changefeed ID")
		return
	}
	changefeedId := *changefeed.Id

	// The changefeed now exists remotely, so it must be tracked from this
	// point on: persist the identity and planned configuration before waiting
	// for readiness, and keep the latest valid snapshot on every later error.
	// A failed create then leaves a destroyable (tainted) changefeed instead
	// of an unmanaged one that the next apply would create again.
	refreshDedicatedChangefeedComputedFields(changefeed, &data)
	// paused unset in config is unknown in the plan and must be known before
	// the first save: a new changefeed starts running and is only paused
	// below when the configuration asks for it.
	if data.Paused.IsUnknown() || data.Paused.IsNull() {
		data.Paused = types.BoolValue(false)
	}
	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	changefeed, err = WaitDedicatedChangefeedState(ctx, changefeedTimeout, changefeedPollInterval, changefeedId, r.provider.DedicatedClient,
		[]string{tidbcloud.ChangefeedStateCreating},
		[]string{tidbcloud.ChangefeedStateRunning, tidbcloud.ChangefeedStateWarning},
	)
	if err != nil {
		// State keeps the created identity saved above.
		resp.Diagnostics.AddError("Create Error", fmt.Sprintf("Changefeed %s is not ready, get error: %s", changefeedId, err))
		return
	}
	refreshDedicatedChangefeedComputedFields(changefeed, &data)
	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if data.Paused.ValueBool() {
		if err := r.provider.DedicatedClient.PauseChangefeed(ctx, changefeedId); err != nil {
			// State keeps the ready (running) snapshot saved above.
			resp.Diagnostics.AddError("Create Error", fmt.Sprintf("Unable to call PauseChangefeed, got error: %s", err))
			return
		}
		changefeed, err = WaitDedicatedChangefeedState(ctx, changefeedTimeout, changefeedPollInterval, changefeedId, r.provider.DedicatedClient,
			[]string{tidbcloud.ChangefeedStatePausing, tidbcloud.ChangefeedStateRunning, tidbcloud.ChangefeedStateWarning},
			[]string{tidbcloud.ChangefeedStatePaused},
		)
		if err != nil {
			resp.Diagnostics.AddError("Create Error", fmt.Sprintf("Changefeed %s is not paused, get error: %s", changefeedId, err))
			return
		}
	}

	refreshDedicatedChangefeedComputedFields(changefeed, &data)

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (r dedicatedChangefeedResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data dedicatedChangefeedResourceData
	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Trace(ctx, "read dedicated_changefeed_resource")
	changefeed, err := r.provider.DedicatedClient.GetChangefeed(ctx, data.ChangefeedId.ValueString())
	if err != nil {
		if tidbcloud.IsChangefeedNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Read Error", fmt.Sprintf("Unable to call GetChangefeed, got error: %s", err))
		return
	}
	if changefeed.State != nil && (*changefeed.State == tidbcloud.ChangefeedStateDeleted || *changefeed.State == tidbcloud.ChangefeedStateDeleting) {
		resp.State.RemoveResource(ctx)
		return
	}

	refreshDedicatedChangefeedResourceData(ctx, changefeed, &data)
	// Not an error: that would block every plan of the configuration,
	// including the -replace that recovers the changefeed. Update rejects
	// changes to it with a clear error instead.
	if changefeed.State != nil && (*changefeed.State == tidbcloud.ChangefeedStateFailed || *changefeed.State == tidbcloud.ChangefeedStateError) {
		msg := ""
		if changefeed.Error != nil {
			msg = ": " + *changefeed.Error
		}
		resp.Diagnostics.AddWarning(
			"Changefeed not healthy",
			fmt.Sprintf("Changefeed %s is in the %s state%s. It cannot be modified until the failure is resolved out of band or it is recreated (terraform apply -replace).", data.ChangefeedId.ValueString(), *changefeed.State, msg),
		)
	}

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

// scaleRequiresRunningError rejects the one replication_capacity (RCU) change
// that cannot be sequenced: ScaleChangefeed requires the RUNNING state, so a
// changefeed that is paused and stays paused cannot be scaled. A paused feed
// that is being resumed is resumed first and then scaled, and a running feed
// that is being paused is scaled first and then paused. It returns a non-nil
// error describing the violation, or nil when the change is allowed.
func scaleRequiresRunningError(capacityChanging, currentlyPaused, willBePaused bool) error {
	if capacityChanging && currentlyPaused && willBePaused {
		return fmt.Errorf("replication_capacity (RCU) can only be changed while the changefeed is RUNNING: ScaleChangefeed requires the RUNNING state, and this changefeed is paused and stays paused. Set paused = false (it is resumed before the scale), or change replication_capacity in an apply where the changefeed is running")
	}
	return nil
}

// ModifyPlan rejects the replication_capacity (RCU) change that cannot be
// sequenced at PLAN time, so the user sees the constraint before apply
// (ScaleChangefeed would otherwise fail at apply with a 400 "current status
// paused, not allowed to Scale"). The current pause state is the refreshed
// state, which Read derives from the live changefeed.
func (r *dedicatedChangefeedResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	// Only meaningful on update: skip create (no prior capacity to change) and
	// destroy (no plan).
	if req.State.Raw.IsNull() || req.Plan.Raw.IsNull() {
		return
	}
	var plan, state dedicatedChangefeedResourceData
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if plan.Paused.IsUnknown() {
		return
	}
	capacityChanging := !plan.ReplicationCapacity.Equal(state.ReplicationCapacity)
	if err := scaleRequiresRunningError(capacityChanging, state.Paused.ValueBool(), plan.Paused.ValueBool()); err != nil {
		resp.Diagnostics.AddAttributeError(path.Root("replication_capacity"), "Invalid Update", err.Error())
	}
}

func (r dedicatedChangefeedResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan dedicatedChangefeedResourceData
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	var state dedicatedChangefeedResourceData
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	// paused left unset in config: Terraform does not manage the pause state.
	var configPaused types.Bool
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("paused"), &configPaused)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := validateChangefeedDownstream(&plan); err != nil {
		resp.Diagnostics.AddError("Invalid Configuration", err.Error())
		return
	}

	changefeedId := state.ChangefeedId.ValueString()
	isCapacityChanging := !plan.ReplicationCapacity.Equal(state.ReplicationCapacity)
	isDownstreamConfigChanging := !reflect.DeepEqual(plan.TableConfig, state.TableConfig) ||
		!reflect.DeepEqual(plan.Kafka, state.Kafka) ||
		!reflect.DeepEqual(plan.Mysql, state.Mysql)

	tflog.Trace(ctx, "update dedicated_changefeed_resource")
	// Everything below is decided from the LIVE changefeed, not the terraform
	// state: the feed may have been paused, resumed or failed out of band since
	// the last refresh.
	current, err := r.provider.DedicatedClient.GetChangefeed(ctx, changefeedId)
	if err != nil {
		resp.Diagnostics.AddError("Update Error", fmt.Sprintf("Unable to read changefeed %s before update, got error: %s", changefeedId, err))
		return
	}
	// A FAILED changefeed accepts no mutations (pause/resume/scale/edit all
	// reject); surface that as a clear error instead of an opaque API failure.
	if current.State != nil && *current.State == tidbcloud.ChangefeedStateFailed {
		resp.Diagnostics.AddError(
			"Invalid Update",
			fmt.Sprintf("Changefeed %s is in FAILED state and cannot be modified. Resolve the failure out of band (or recreate the changefeed, e.g. terraform apply -replace) before changing its configuration.", changefeedId),
		)
		return
	}
	// Let an operation that is still in flight (for example one started out
	// of band) settle, so the sequencing below starts from a steady state.
	if current.State != nil && isTransitionalChangefeedState(*current.State) {
		current, err = WaitDedicatedChangefeedState(ctx, changefeedTimeout, changefeedPollInterval, changefeedId, r.provider.DedicatedClient,
			changefeedTransitionalStates, changefeedSteadyStates)
		if err != nil {
			resp.Diagnostics.AddError("Update Error", fmt.Sprintf("Changefeed %s did not settle before the update, get error: %s", changefeedId, err))
			return
		}
	}

	livePaused, _ := pausedFromChangefeedState(current.State)
	desiredPaused := livePaused
	if !configPaused.IsNull() && !configPaused.IsUnknown() {
		desiredPaused = configPaused.ValueBool()
	}
	// ModifyPlan checks the same against the refreshed state; this guards the
	// feed having been paused since.
	if err := scaleRequiresRunningError(isCapacityChanging, livePaused, desiredPaused); err != nil {
		resp.Diagnostics.AddError("Invalid Update", err.Error())
		return
	}

	// Order the calls so that each one finds the state its API requires
	// (ScaleChangefeed: RUNNING; EditChangefeedDownstreamConfig: PAUSED) with
	// as few pause/resume cycles as possible:
	//   live paused:  edit -> resume -> scale
	//   live running: scale -> pause -> edit -> resume
	if livePaused {
		// A failed edit leaves the feed paused, as it entered the update.
		if isDownstreamConfigChanging && !r.editChangefeedDownstreamConfig(ctx, changefeedId, &plan, &resp.Diagnostics) {
			return
		}
		if !desiredPaused && !r.resumeChangefeed(ctx, changefeedId, &resp.Diagnostics) {
			return
		}
		if isCapacityChanging && !r.scaleChangefeed(ctx, changefeedId, plan.ReplicationCapacity.ValueString(), &resp.Diagnostics) {
			return
		}
	} else {
		if isCapacityChanging && !r.scaleChangefeed(ctx, changefeedId, plan.ReplicationCapacity.ValueString(), &resp.Diagnostics) {
			return
		}
		if isDownstreamConfigChanging || desiredPaused {
			if !r.pauseChangefeed(ctx, changefeedId, &resp.Diagnostics) {
				return
			}
		}
		if isDownstreamConfigChanging {
			// Best-effort recovery: never leave a feed paused that this update
			// paused only for the edit.
			resumed := desiredPaused
			defer func() {
				if resumed {
					return
				}
				if err := r.provider.DedicatedClient.ResumeChangefeed(ctx, changefeedId); err != nil {
					resp.Diagnostics.AddWarning(
						"Changefeed left paused",
						fmt.Sprintf("The downstream-config edit failed and the automatic resume of changefeed %s also failed: %s. Resume it manually or re-run terraform apply.", changefeedId, err),
					)
				}
			}()
			if !r.editChangefeedDownstreamConfig(ctx, changefeedId, &plan, &resp.Diagnostics) {
				return
			}
			if !desiredPaused {
				resumed = true
				if !r.resumeChangefeed(ctx, changefeedId, &resp.Diagnostics) {
					return
				}
			}
		}
	}

	changefeed, err := r.provider.DedicatedClient.GetChangefeed(ctx, changefeedId)
	if err != nil {
		resp.Diagnostics.AddError("Update Error", fmt.Sprintf("Unable to read changefeed after update, got error: %s", err))
		return
	}
	refreshDedicatedChangefeedComputedFields(changefeed, &plan)
	if plan.Paused.IsUnknown() || plan.Paused.IsNull() {
		plan.Paused = types.BoolValue(desiredPaused)
	}

	diags = resp.State.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
}

// pauseChangefeed pauses a running changefeed and waits for PAUSED.
func (r dedicatedChangefeedResource) pauseChangefeed(ctx context.Context, changefeedId string, diags *diag.Diagnostics) bool {
	if err := r.provider.DedicatedClient.PauseChangefeed(ctx, changefeedId); err != nil {
		diags.AddError("Pause Error", fmt.Sprintf("Unable to call PauseChangefeed, got error: %s", err))
		return false
	}
	if _, err := WaitDedicatedChangefeedState(ctx, changefeedTimeout, changefeedPollInterval, changefeedId, r.provider.DedicatedClient,
		[]string{tidbcloud.ChangefeedStatePausing, tidbcloud.ChangefeedStateRunning, tidbcloud.ChangefeedStateWarning},
		[]string{tidbcloud.ChangefeedStatePaused},
	); err != nil {
		diags.AddError("Pause Error", fmt.Sprintf("Changefeed %s is not paused, get error: %s", changefeedId, err))
		return false
	}
	return true
}

// resumeChangefeed resumes a paused changefeed and waits for RUNNING.
func (r dedicatedChangefeedResource) resumeChangefeed(ctx context.Context, changefeedId string, diags *diag.Diagnostics) bool {
	if err := r.provider.DedicatedClient.ResumeChangefeed(ctx, changefeedId); err != nil {
		diags.AddError("Resume Error", fmt.Sprintf("Unable to call ResumeChangefeed, got error: %s", err))
		return false
	}
	if _, err := WaitDedicatedChangefeedState(ctx, changefeedTimeout, changefeedPollInterval, changefeedId, r.provider.DedicatedClient,
		[]string{tidbcloud.ChangefeedStatePaused},
		changefeedRunningStates,
	); err != nil {
		diags.AddError("Resume Error", fmt.Sprintf("Changefeed %s is not resumed, get error: %s", changefeedId, err))
		return false
	}
	return true
}

// scaleChangefeed scales a running changefeed and waits for it to be running
// again.
func (r dedicatedChangefeedResource) scaleChangefeed(ctx context.Context, changefeedId, replicationCapacity string, diags *diag.Diagnostics) bool {
	if _, err := r.provider.DedicatedClient.ScaleChangefeed(ctx, changefeedId, replicationCapacity); err != nil {
		diags.AddError("Update Error", fmt.Sprintf("Unable to call ScaleChangefeed, got error: %s", err))
		return false
	}
	if _, err := WaitDedicatedChangefeedState(ctx, changefeedTimeout, changefeedPollInterval, changefeedId, r.provider.DedicatedClient,
		[]string{tidbcloud.ChangefeedStateScaling},
		changefeedRunningStates,
	); err != nil {
		diags.AddError("Update Error", fmt.Sprintf("Changefeed %s failed to scale, get error: %s", changefeedId, err))
		return false
	}
	return true
}

// editChangefeedDownstreamConfig applies the planned downstream config to a
// paused changefeed and waits for it to settle back in PAUSED.
func (r dedicatedChangefeedResource) editChangefeedDownstreamConfig(ctx context.Context, changefeedId string, plan *dedicatedChangefeedResourceData, diags *diag.Diagnostics) bool {
	body := &tidbcloud.EditChangefeedDownstreamConfigRequest{
		DownstreamType: plan.DownstreamType.ValueString(),
		TableConfig:    tableConfigModelToAPI(ctx, plan.TableConfig, diags),
		Kafka:          kafkaModelToAPI(ctx, plan.Kafka, diags),
		Mysql:          mysqlModelToAPI(plan.Mysql),
	}
	if diags.HasError() {
		return false
	}
	if _, err := r.provider.DedicatedClient.EditChangefeedDownstreamConfig(ctx, changefeedId, body); err != nil {
		diags.AddError("Update Error", fmt.Sprintf("Unable to call EditChangefeedDownstreamConfig, got error: %s", err))
		return false
	}
	if _, err := WaitDedicatedChangefeedState(ctx, changefeedTimeout, changefeedPollInterval, changefeedId, r.provider.DedicatedClient,
		[]string{tidbcloud.ChangefeedStateEditing},
		[]string{tidbcloud.ChangefeedStatePaused},
	); err != nil {
		diags.AddError("Update Error", fmt.Sprintf("Changefeed %s failed to apply the downstream configuration, get error: %s", changefeedId, err))
		return false
	}
	return true
}

func (r dedicatedChangefeedResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var changefeedId string
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("changefeed_id"), &changefeedId)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Trace(ctx, "delete dedicated_changefeed_resource")
	err := r.provider.DedicatedClient.DeleteChangefeed(ctx, changefeedId)
	// Treat an already-removed changefeed as a successful delete so destroy is
	// idempotent if the changefeed was removed out of band.
	if err != nil {
		if tidbcloud.IsChangefeedNotFoundError(err) {
			return
		}
		resp.Diagnostics.AddError("Delete Error", fmt.Sprintf("Unable to call DeleteChangefeed, got error: %s", err))
		return
	}

	if err := waitDedicatedChangefeedDeleted(ctx, changefeedTimeout, changefeedPollInterval, changefeedId, r.provider.DedicatedClient); err != nil {
		resp.Diagnostics.AddError("Delete Error", fmt.Sprintf("Changefeed %s is not deleted, get error: %s", changefeedId, err))
		return
	}
}

func (r dedicatedChangefeedResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("changefeed_id"), req, resp)
}

// validateChangefeedDownstream ensures exactly the downstream configuration
// matching downstream_type is present. The check cannot be expressed with plan
// modifiers because it spans multiple attributes.
func validateChangefeedDownstream(data *dedicatedChangefeedResourceData) error {
	switch data.DownstreamType.ValueString() {
	case tidbcloud.ChangefeedDownstreamTypeKafka:
		if data.Kafka == nil {
			return fmt.Errorf("`kafka` must be configured when `downstream_type` is %q", tidbcloud.ChangefeedDownstreamTypeKafka)
		}
		if data.Mysql != nil {
			return fmt.Errorf("`mysql` must not be configured when `downstream_type` is %q", tidbcloud.ChangefeedDownstreamTypeKafka)
		}
	case tidbcloud.ChangefeedDownstreamTypeMySQL:
		if data.Mysql == nil {
			return fmt.Errorf("`mysql` must be configured when `downstream_type` is %q", tidbcloud.ChangefeedDownstreamTypeMySQL)
		}
		if data.Kafka != nil {
			return fmt.Errorf("`kafka` must not be configured when `downstream_type` is %q", tidbcloud.ChangefeedDownstreamTypeMySQL)
		}
	default:
		return fmt.Errorf("unsupported `downstream_type` %q; available values: %q, %q",
			data.DownstreamType.ValueString(), tidbcloud.ChangefeedDownstreamTypeKafka, tidbcloud.ChangefeedDownstreamTypeMySQL)
	}
	return nil
}

// refreshDedicatedChangefeedComputedFields refreshes only the API-computed
// fields, leaving the user-managed configuration untouched. Used by Create and
// Update where those values come from config/plan.
func refreshDedicatedChangefeedComputedFields(changefeed *tidbcloud.Changefeed, data *dedicatedChangefeedResourceData) {
	data.ChangefeedId = types.StringPointerValue(changefeed.Id)
	data.State = types.StringPointerValue(changefeed.State)
	data.CreateTime = types.StringPointerValue(changefeed.CreateTime)
	data.CheckpointTso = types.StringPointerValue(changefeed.CheckpointTso)
	data.CheckpointTs = types.StringPointerValue(changefeed.CheckpointTs)
	data.Error = types.StringPointerValue(changefeed.Error)
}

// refreshDedicatedChangefeedResourceData copies the API changefeed onto the
// Terraform model. On import the nested blocks are populated wholesale from
// the API. On every other read the downstream blocks (table_config, kafka,
// mysql) are reconciled field-wise via the merge helpers in
// dedicated_changefeed_convert.go: readable fields the configuration manages
// are refreshed from the API so out-of-band edits surface as drift, while
// input-only credentials are retained from prior state and attributes the
// configuration leaves unset stay null (server-side defaults must not become
// perpetual diffs). network_info and start_position are immutable
// (RequiresReplace) and stay import-only.
func refreshDedicatedChangefeedResourceData(ctx context.Context, changefeed *tidbcloud.Changefeed, data *dedicatedChangefeedResourceData) {
	isImport := data.Name.IsNull()

	refreshDedicatedChangefeedComputedFields(changefeed, data)
	data.ClusterId = types.StringValue(changefeed.ClusterId)
	data.Name = types.StringValue(changefeed.Name)
	data.ReplicationCapacity = types.StringValue(changefeed.ReplicationCapacity)
	data.DownstreamType = types.StringValue(changefeed.DownstreamType)

	// paused is read from the live state on every refresh, so a pause or
	// resume done out of band shows up as a diff (or, with paused unset in
	// config, is simply reported). A FAILED/ERROR feed keeps the prior value.
	if paused, ok := pausedFromChangefeedState(changefeed.State); ok {
		data.Paused = types.BoolValue(paused)
	} else if data.Paused.IsNull() || data.Paused.IsUnknown() {
		data.Paused = types.BoolValue(false)
	}

	if isImport {
		data.NetworkInfo = networkInfoAPIToModel(ctx, changefeed.NetworkInfo)
		data.StartPosition = startPositionAPIToModel(changefeed.StartPosition)
		data.TableConfig = tableConfigAPIToModel(ctx, changefeed.TableConfig)
		data.Kafka = kafkaAPIToModel(ctx, changefeed.Kafka)
		data.Mysql = mysqlAPIToModel(changefeed.Mysql)
	} else {
		data.TableConfig = mergeTableConfig(data.TableConfig, tableConfigAPIToModel(ctx, changefeed.TableConfig))
		data.Kafka = mergeKafka(data.Kafka, kafkaAPIToModel(ctx, changefeed.Kafka))
		data.Mysql = mergeMysql(data.Mysql, mysqlAPIToModel(changefeed.Mysql))
	}
}

var (
	// changefeedRunningStates are the states a changefeed is in once it is
	// replicating.
	changefeedRunningStates = []string{tidbcloud.ChangefeedStateRunning, tidbcloud.ChangefeedStateWarning}
	// changefeedSteadyStates are the states an Update can start sequencing from.
	changefeedSteadyStates = []string{tidbcloud.ChangefeedStateRunning, tidbcloud.ChangefeedStateWarning, tidbcloud.ChangefeedStatePaused}
	// changefeedTransitionalStates are in-flight operations to wait out.
	changefeedTransitionalStates = []string{tidbcloud.ChangefeedStateCreating, tidbcloud.ChangefeedStatePausing, tidbcloud.ChangefeedStateScaling, tidbcloud.ChangefeedStateEditing}
)

func isTransitionalChangefeedState(state string) bool {
	for _, s := range changefeedTransitionalStates {
		if s == state {
			return true
		}
	}
	return false
}

// pausedFromChangefeedState maps a live changefeed state to the `paused`
// attribute. ok is false for states with no meaningful answer (FAILED, ERROR,
// DELETING, DELETED, missing or unrecognised), where the caller keeps what it
// had. EDITING reads as paused: an edit is only accepted on a PAUSED feed and
// settles back in PAUSED.
func pausedFromChangefeedState(state *string) (paused bool, ok bool) {
	if state == nil {
		return false, false
	}
	switch *state {
	case tidbcloud.ChangefeedStatePaused, tidbcloud.ChangefeedStatePausing, tidbcloud.ChangefeedStateEditing:
		return true, true
	case tidbcloud.ChangefeedStateRunning, tidbcloud.ChangefeedStateWarning, tidbcloud.ChangefeedStateScaling, tidbcloud.ChangefeedStateCreating:
		return false, true
	default:
		return false, false
	}
}

func buildChangefeedBody(ctx context.Context, data *dedicatedChangefeedResourceData, diags *diag.Diagnostics) *tidbcloud.Changefeed {
	return &tidbcloud.Changefeed{
		ClusterId:           data.ClusterId.ValueString(),
		Name:                data.Name.ValueString(),
		ReplicationCapacity: data.ReplicationCapacity.ValueString(),
		DownstreamType:      data.DownstreamType.ValueString(),
		NetworkInfo:         networkInfoModelToAPI(ctx, data.NetworkInfo, diags),
		StartPosition:       startPositionModelToAPI(data.StartPosition),
		TableConfig:         tableConfigModelToAPI(ctx, data.TableConfig, diags),
		Kafka:               kafkaModelToAPI(ctx, data.Kafka, diags),
		Mysql:               mysqlModelToAPI(data.Mysql),
	}
}

func WaitDedicatedChangefeedState(ctx context.Context, timeout time.Duration, interval time.Duration, changefeedId string,
	client tidbcloud.TiDBCloudDedicatedClient, pending []string, target []string) (*tidbcloud.Changefeed, error) {
	stateConf := &retry.StateChangeConf{
		Pending:      pending,
		Target:       target,
		Timeout:      timeout,
		MinTimeout:   500 * time.Millisecond,
		PollInterval: interval,
		Refresh:      dedicatedChangefeedStateRefreshFunc(ctx, changefeedId, client),
	}

	outputRaw, err := stateConf.WaitForStateContext(ctx)

	if output, ok := outputRaw.(*tidbcloud.Changefeed); ok {
		return output, err
	}
	return nil, err
}

func dedicatedChangefeedStateRefreshFunc(ctx context.Context, changefeedId string,
	client tidbcloud.TiDBCloudDedicatedClient) retry.StateRefreshFunc {
	return func() (interface{}, string, error) {
		tflog.Trace(ctx, "Waiting for dedicated changefeed state")
		changefeed, err := client.GetChangefeed(ctx, changefeedId)
		if err != nil {
			return nil, "", err
		}
		if changefeed.State == nil {
			return nil, "", fmt.Errorf("changefeed %s has no state", changefeedId)
		}
		if *changefeed.State == tidbcloud.ChangefeedStateFailed || *changefeed.State == tidbcloud.ChangefeedStateError {
			msg := ""
			if changefeed.Error != nil {
				msg = *changefeed.Error
			}
			return changefeed, *changefeed.State, fmt.Errorf("changefeed is in %s state: %s", *changefeed.State, msg)
		}
		return changefeed, *changefeed.State, nil
	}
}

func waitDedicatedChangefeedDeleted(ctx context.Context, timeout time.Duration, interval time.Duration, changefeedId string,
	client tidbcloud.TiDBCloudDedicatedClient) error {
	deadline := time.Now().Add(timeout)
	for {
		changefeed, err := client.GetChangefeed(ctx, changefeedId)
		if err != nil {
			if tidbcloud.IsChangefeedNotFoundError(err) {
				return nil
			}
			return err
		}
		if changefeed.State != nil && *changefeed.State == tidbcloud.ChangefeedStateDeleted {
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("timeout waiting for changefeed %s to be deleted", changefeedId)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(interval):
		}
	}
}
