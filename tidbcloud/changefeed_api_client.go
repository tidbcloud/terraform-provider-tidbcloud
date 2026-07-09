package tidbcloud

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
)

// ChangefeedAPIError wraps a non-2xx response from the hand-written changefeed
// REST client, preserving the HTTP status code so callers can react to it (for
// example, treating 404 on delete as success).
type ChangefeedAPIError struct {
	StatusCode int
	Err        error
}

func (e *ChangefeedAPIError) Error() string {
	if e.Err != nil {
		return e.Err.Error()
	}
	return fmt.Sprintf("unexpected status: %d", e.StatusCode)
}

func (e *ChangefeedAPIError) Unwrap() error { return e.Err }

// IsChangefeedNotFoundError reports whether err is a ChangefeedAPIError with a
// 404 status.
func IsChangefeedNotFoundError(err error) bool {
	var apiErr *ChangefeedAPIError
	if errors.As(err, &apiErr) {
		return apiErr.StatusCode == http.StatusNotFound
	}
	return false
}

// The dedicated Go SDK (github.com/tidbcloud/tidbcloud-cli/.../v1beta1/dedicated)
// does not expose the changefeed API, so the request/response models below are
// hand-written to match the dedicated v1beta1 OpenAPI spec
// (https://docs-download.pingcap.com/api/tidbcloud-oas-v1beta1-dedicated.swagger.json).
// Only the Kafka and MySQL downstream configurations are modeled for now; the
// S3, GCS, and Azure Blob downstreams can be added later without breaking
// changes.

// ChangefeedState enumerates the lifecycle states of a changefeed.
const (
	ChangefeedStateRunning  = "RUNNING"
	ChangefeedStateFailed   = "FAILED"
	ChangefeedStateError    = "ERROR"
	ChangefeedStateCreating = "CREATING"
	ChangefeedStatePausing  = "PAUSING"
	ChangefeedStatePaused   = "PAUSED"
	ChangefeedStateDeleting = "DELETING"
	ChangefeedStateDeleted  = "DELETED"
	ChangefeedStateEditing  = "EDITING"
	ChangefeedStateWarning  = "WARNING"
	ChangefeedStateScaling  = "SCALING"
)

// Changefeed downstream types.
const (
	ChangefeedDownstreamTypeKafka = "KAFKA"
	ChangefeedDownstreamTypeMySQL = "MYSQL"
)

// ChangefeedNetworkInfo is the network configuration for the downstream
// connection.
type ChangefeedNetworkInfo struct {
	// NetworkType is one of NETWORK_TYPE_PUBLIC, NETWORK_TYPE_VPC_PEERING,
	// NETWORK_TYPE_PRIVATE_LINK.
	NetworkType string `json:"networkType"`
	// SinkEndpointId is required for the PRIVATE_LINK network type.
	SinkEndpointId *string `json:"sinkEndpointId,omitempty"`
	// Ports holds the downstream target ports for the PRIVATE_LINK network type.
	Ports []int64 `json:"ports,omitempty"`
}

// ChangefeedEventFilter is an event filter rule for fine-grained control.
type ChangefeedEventFilter struct {
	TableMatchers                   []string `json:"tableMatchers,omitempty"`
	IgnoredEvents                   []string `json:"ignoredEvents,omitempty"`
	IgnoredSqlStatements            []string `json:"ignoredSqlStatements,omitempty"`
	IgnoredInsertValueExpression    *string  `json:"ignoredInsertValueExpression,omitempty"`
	IgnoredUpdateOldValueExpression *string  `json:"ignoredUpdateOldValueExpression,omitempty"`
	IgnoredUpdateNewValueExpression *string  `json:"ignoredUpdateNewValueExpression,omitempty"`
	IgnoredDeleteValueExpression    *string  `json:"ignoredDeleteValueExpression,omitempty"`
}

// ChangefeedTableConfig is the table filtering and event filter configuration.
type ChangefeedTableConfig struct {
	// FilterRules uses the TiCDC table filter syntax.
	FilterRules []string `json:"filterRules,omitempty"`
	// Mode is one of IGNORE_NOT_SUPPORT_TABLE, FORCE_SYNC.
	Mode          *string                 `json:"mode,omitempty"`
	CaseSensitive *bool                   `json:"caseSensitive,omitempty"`
	EventFilters  []ChangefeedEventFilter `json:"eventFilters,omitempty"`
}

// ChangefeedStartPosition is the start position for the changefeed.
type ChangefeedStartPosition struct {
	// Mode is one of FROM_NOW, FROM_TSO, FROM_UTC.
	Mode string `json:"mode"`
	// StartTso is required when the mode is FROM_TSO.
	StartTso *string `json:"startTso,omitempty"`
	// StartTimestamp is required when the mode is FROM_UTC (RFC 3339).
	StartTimestamp *string `json:"startTimestamp,omitempty"`
}

// ChangefeedKafkaBroker is the Kafka broker connection configuration.
type ChangefeedKafkaBroker struct {
	// Version is one of KAFKA_VERSION_0XX, KAFKA_VERSION_1XX,
	// KAFKA_VERSION_2XX, KAFKA_VERSION_3XX.
	Version string `json:"version"`
	// BrokerEndpoints is a comma-separated list of broker addresses. Required
	// for the PUBLIC and VPC_PEERING network types.
	BrokerEndpoints    *string `json:"brokerEndpoints,omitempty"`
	UseTls             *bool   `json:"useTls,omitempty"`
	InsecureSkipVerify *bool   `json:"insecureSkipVerify,omitempty"`
	// Compression is one of NONE, GZIP, SNAPPY, LZ4, ZSTD.
	Compression *string `json:"compression,omitempty"`
}

// ChangefeedKafkaAuthentication is the Kafka authentication configuration.
type ChangefeedKafkaAuthentication struct {
	// AuthType is one of DISABLE, SASL_PLAIN, SASL_SCRAM_SHA_256,
	// SASL_SCRAM_SHA_512.
	AuthType string  `json:"authType"`
	Username *string `json:"username,omitempty"`
	// Password is input-only and not returned in responses.
	Password *string `json:"password,omitempty"`
}

// ChangefeedKafkaDebeziumConfig is the Debezium-specific data format
// configuration.
type ChangefeedKafkaDebeziumConfig struct {
	OutputOldValue *bool `json:"outputOldValue,omitempty"`
	DisableSchema  *bool `json:"disableSchema,omitempty"`
}

// ChangefeedKafkaDataFormat is the Kafka message format and serialization
// configuration. The Avro-specific configuration (schema registry) is not
// modeled yet.
type ChangefeedKafkaDataFormat struct {
	// Protocol is one of PROTOCOL_CANAL_JSON, PROTOCOL_OPEN_PROTOCOL,
	// PROTOCOL_AVRO, PROTOCOL_DEBEZIUM.
	Protocol             string                         `json:"protocol"`
	DebeziumConfig       *ChangefeedKafkaDebeziumConfig `json:"debeziumConfig,omitempty"`
	EnableTidbExtension  *bool                          `json:"enableTidbExtension,omitempty"`
	OutputRawChangeEvent *bool                          `json:"outputRawChangeEvent,omitempty"`
}

// ChangefeedKafkaPartitionDispatcher is a custom partition dispatcher
// configuration.
type ChangefeedKafkaPartitionDispatcher struct {
	PartitionType string   `json:"partitionType"`
	Matcher       []string `json:"matcher,omitempty"`
	IndexName     *string  `json:"indexName,omitempty"`
	Columns       []string `json:"columns,omitempty"`
}

// ChangefeedKafkaTopicPartitionConfig is the Kafka topic and partition
// configuration.
type ChangefeedKafkaTopicPartitionConfig struct {
	// DispatchType is one of DISPATCH_TYPE_ONE_TOPIC, DISPATCH_TYPE_BY_TABLE,
	// DISPATCH_TYPE_BY_DATABASE.
	DispatchType         string                               `json:"dispatchType"`
	DefaultTopic         *string                              `json:"defaultTopic,omitempty"`
	TopicPrefix          *string                              `json:"topicPrefix,omitempty"`
	Separator            *string                              `json:"separator,omitempty"`
	TopicSuffix          *string                              `json:"topicSuffix,omitempty"`
	ReplicationFactor    int64                                `json:"replicationFactor"`
	PartitionNum         int64                                `json:"partitionNum"`
	PartitionDispatchers []ChangefeedKafkaPartitionDispatcher `json:"partitionDispatchers,omitempty"`
}

// ChangefeedKafkaColumnSelector is the column filtering configuration for
// specific tables.
type ChangefeedKafkaColumnSelector struct {
	Matcher []string `json:"matcher,omitempty"`
	Columns []string `json:"columns,omitempty"`
}

// ChangefeedKafkaConfig is the Kafka downstream configuration.
type ChangefeedKafkaConfig struct {
	Broker               *ChangefeedKafkaBroker               `json:"broker,omitempty"`
	Authentication       *ChangefeedKafkaAuthentication       `json:"authentication,omitempty"`
	DataFormat           *ChangefeedKafkaDataFormat           `json:"dataFormat,omitempty"`
	TopicPartitionConfig *ChangefeedKafkaTopicPartitionConfig `json:"topicPartitionConfig,omitempty"`
	ColumnSelectors      []ChangefeedKafkaColumnSelector      `json:"columnSelectors,omitempty"`
}

// ChangefeedMySQLConnection is the MySQL connection configuration.
type ChangefeedMySQLConnection struct {
	// Endpoint is the host:port address. Required for the PUBLIC and
	// VPC_PEERING network types; for PRIVATE_LINK use
	// networkInfo.sinkEndpointId instead.
	Endpoint *string `json:"endpoint,omitempty"`
	Username string  `json:"username"`
	// Password is input-only and not returned in responses.
	Password *string `json:"password,omitempty"`
}

// ChangefeedMySQLConfig is the MySQL downstream configuration.
type ChangefeedMySQLConfig struct {
	Connection *ChangefeedMySQLConnection `json:"connection,omitempty"`
}

// Changefeed is a changefeed of a TiDB Cloud Dedicated cluster.
type Changefeed struct {
	// Id is output-only.
	Id        *string `json:"id,omitempty"`
	ClusterId string  `json:"clusterId"`
	Name      string  `json:"name"`
	// NetworkInfo is required on create.
	NetworkInfo *ChangefeedNetworkInfo `json:"networkInfo,omitempty"`
	// State is output-only.
	State *string `json:"state,omitempty"`
	// CreateTime is output-only.
	CreateTime *string `json:"createTime,omitempty"`
	// CheckpointTso is output-only.
	CheckpointTso *string `json:"checkpointTso,omitempty"`
	// CheckpointTs is output-only.
	CheckpointTs *string `json:"checkpointTs,omitempty"`
	// ReplicationCapacity is an RCU option name such as "4rcu"; call
	// ListChangefeedRCUs for the available options.
	ReplicationCapacity string `json:"replicationCapacity"`
	// DownstreamType is one of KAFKA, MYSQL, S3, GCS, AZURE_BLOB.
	DownstreamType string                   `json:"downstreamType"`
	TableConfig    *ChangefeedTableConfig   `json:"tableConfig,omitempty"`
	StartPosition  *ChangefeedStartPosition `json:"startPosition,omitempty"`
	Kafka          *ChangefeedKafkaConfig   `json:"kafka,omitempty"`
	Mysql          *ChangefeedMySQLConfig   `json:"mysql,omitempty"`
	// Error is output-only; set when the changefeed is in the FAILED or ERROR
	// state.
	Error *string `json:"error,omitempty"`
}

// CreateChangefeedRequest is the body of POST /changefeeds.
type CreateChangefeedRequest struct {
	Changefeed *Changefeed `json:"changefeed"`
	DryRun     *bool       `json:"dryRun,omitempty"`
}

// ListChangefeedsParams holds the filters/pagination for GET /changefeeds.
type ListChangefeedsParams struct {
	ClusterId      string
	DownstreamType *string
	PageSize       *int32
	PageToken      *string
}

// ListChangefeedsResponse is the response of GET /changefeeds.
type ListChangefeedsResponse struct {
	Changefeeds   []Changefeed `json:"changefeeds,omitempty"`
	TotalSize     *int32       `json:"totalSize,omitempty"`
	NextPageToken *string      `json:"nextPageToken,omitempty"`
}

// EditChangefeedDownstreamConfigRequest is the body of
// POST /changefeeds/{changefeedId}:editDownstreamConfig.
type EditChangefeedDownstreamConfigRequest struct {
	DryRun         *bool                  `json:"dryRun,omitempty"`
	TableConfig    *ChangefeedTableConfig `json:"tableConfig,omitempty"`
	DownstreamType string                 `json:"downstreamType"`
	Kafka          *ChangefeedKafkaConfig `json:"kafka,omitempty"`
	Mysql          *ChangefeedMySQLConfig `json:"mysql,omitempty"`
}

type scaleChangefeedRequest struct {
	ReplicationCapacity string `json:"replicationCapacity"`
}

func (d *DedicatedClientDelegate) CreateChangefeed(ctx context.Context, body *CreateChangefeedRequest) (*Changefeed, error) {
	var out Changefeed
	if err := d.doChangefeedRequest(ctx, http.MethodPost, "/changefeeds", nil, body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (d *DedicatedClientDelegate) GetChangefeed(ctx context.Context, changefeedId string) (*Changefeed, error) {
	var out Changefeed
	if err := d.doChangefeedRequest(ctx, http.MethodGet, "/changefeeds/"+url.PathEscape(changefeedId), nil, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (d *DedicatedClientDelegate) ListChangefeeds(ctx context.Context, params *ListChangefeedsParams) (*ListChangefeedsResponse, error) {
	q := url.Values{}
	if params != nil {
		if params.ClusterId != "" {
			q.Set("clusterId", params.ClusterId)
		}
		if params.DownstreamType != nil && *params.DownstreamType != "" {
			q.Set("downstreamType", *params.DownstreamType)
		}
		if params.PageSize != nil {
			q.Set("pageSize", strconv.FormatInt(int64(*params.PageSize), 10))
		}
		if params.PageToken != nil && *params.PageToken != "" {
			q.Set("pageToken", *params.PageToken)
		}
	}
	var out ListChangefeedsResponse
	if err := d.doChangefeedRequest(ctx, http.MethodGet, "/changefeeds", q, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (d *DedicatedClientDelegate) DeleteChangefeed(ctx context.Context, changefeedId string) error {
	return d.doChangefeedRequest(ctx, http.MethodDelete, "/changefeeds/"+url.PathEscape(changefeedId), nil, nil, nil)
}

func (d *DedicatedClientDelegate) PauseChangefeed(ctx context.Context, changefeedId string) error {
	return d.doChangefeedRequest(ctx, http.MethodPost, "/changefeeds/"+url.PathEscape(changefeedId)+":pause", nil, struct{}{}, nil)
}

func (d *DedicatedClientDelegate) ResumeChangefeed(ctx context.Context, changefeedId string) error {
	return d.doChangefeedRequest(ctx, http.MethodPost, "/changefeeds/"+url.PathEscape(changefeedId)+":resume", nil, struct{}{}, nil)
}

func (d *DedicatedClientDelegate) ScaleChangefeed(ctx context.Context, changefeedId string, replicationCapacity string) (*Changefeed, error) {
	var out Changefeed
	body := scaleChangefeedRequest{ReplicationCapacity: replicationCapacity}
	if err := d.doChangefeedRequest(ctx, http.MethodPost, "/changefeeds/"+url.PathEscape(changefeedId)+":scale", nil, body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (d *DedicatedClientDelegate) EditChangefeedDownstreamConfig(ctx context.Context, changefeedId string, body *EditChangefeedDownstreamConfigRequest) (*Changefeed, error) {
	var out Changefeed
	if err := d.doChangefeedRequest(ctx, http.MethodPost, "/changefeeds/"+url.PathEscape(changefeedId)+":editDownstreamConfig", nil, body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// doChangefeedRequest issues a single JSON request against the dedicated
// changefeed API using the shared digest-authenticated HTTP client. Non-2xx
// responses are turned into errors via parseError so they read the same as
// SDK-backed calls.
func (d *DedicatedClientDelegate) doChangefeedRequest(ctx context.Context, method, path string, query url.Values, body, out interface{}) error {
	var reqBody io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reqBody = bytes.NewReader(b)
	}

	u := d.changefeedBaseURL + path
	if len(query) > 0 {
		u += "?" + query.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, method, u, reqBody)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := d.httpClient.Do(req)
	if err != nil {
		return parseError(err, resp)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		statusCode := resp.StatusCode
		return &ChangefeedAPIError{
			StatusCode: statusCode,
			Err:        parseError(fmt.Errorf("unexpected status: %s", resp.Status), resp),
		}
	}
	defer resp.Body.Close()

	if out != nil {
		if err := json.NewDecoder(resp.Body).Decode(out); err != nil && err != io.EOF {
			return err
		}
	}
	return nil
}
