package provider

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	mockClient "github.com/tidbcloud/terraform-provider-tidbcloud/mock"
	"github.com/tidbcloud/terraform-provider-tidbcloud/tidbcloud"
)

// changefeedStore is a minimal stateful fake of the changefeed API used by the
// unit tests so that Create/Get/Scale/Pause/Resume/Edit/Delete behave
// consistently across the many reads the testing framework performs. State
// transitions complete instantly (e.g. Create lands directly in RUNNING) so
// the state waiters return on their first poll.
type changefeedStore struct {
	changefeed tidbcloud.Changefeed
	deleted    bool
}

func (s *changefeedStore) get() *tidbcloud.Changefeed {
	cf := s.changefeed
	return &cf
}

func newChangefeedMock(t *testing.T) (*mockClient.MockTiDBCloudDedicatedClient, *changefeedStore) {
	t.Helper()
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)
	s := mockClient.NewMockTiDBCloudDedicatedClient(ctrl)
	store := &changefeedStore{deleted: true}
	changefeedId := "cf-1"

	s.EXPECT().CreateChangefeed(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, body *tidbcloud.CreateChangefeedRequest) (*tidbcloud.Changefeed, error) {
			store.changefeed = *body.Changefeed
			store.changefeed.Id = Ptr(changefeedId)
			store.changefeed.State = Ptr(tidbcloud.ChangefeedStateRunning)
			store.changefeed.CreateTime = Ptr("2026-01-01T00:00:00Z")
			store.changefeed.CheckpointTso = Ptr("0")
			store.changefeed.CheckpointTs = Ptr("2026-01-01T00:00:00Z")
			store.deleted = false
			return store.get(), nil
		}).AnyTimes()

	s.EXPECT().GetChangefeed(gomock.Any(), changefeedId).DoAndReturn(
		func(_ context.Context, _ string) (*tidbcloud.Changefeed, error) {
			if store.deleted {
				return nil, &tidbcloud.ChangefeedAPIError{StatusCode: http.StatusNotFound}
			}
			return store.get(), nil
		}).AnyTimes()

	s.EXPECT().ScaleChangefeed(gomock.Any(), changefeedId, gomock.Any()).DoAndReturn(
		func(_ context.Context, _ string, replicationCapacity string) (*tidbcloud.Changefeed, error) {
			store.changefeed.ReplicationCapacity = replicationCapacity
			return store.get(), nil
		}).AnyTimes()

	s.EXPECT().EditChangefeedDownstreamConfig(gomock.Any(), changefeedId, gomock.Any()).DoAndReturn(
		func(_ context.Context, _ string, body *tidbcloud.EditChangefeedDownstreamConfigRequest) (*tidbcloud.Changefeed, error) {
			// The real API only accepts downstream-config edits while the
			// changefeed is PAUSED — enforce it here so the resource's
			// auto pause -> edit -> resume sequencing is actually exercised.
			if store.changefeed.State == nil || *store.changefeed.State != tidbcloud.ChangefeedStatePaused {
				return nil, &tidbcloud.ChangefeedAPIError{StatusCode: http.StatusBadRequest}
			}
			store.changefeed.TableConfig = body.TableConfig
			store.changefeed.Kafka = body.Kafka
			store.changefeed.Mysql = body.Mysql
			return store.get(), nil
		}).AnyTimes()

	s.EXPECT().PauseChangefeed(gomock.Any(), changefeedId).DoAndReturn(
		func(_ context.Context, _ string) error {
			store.changefeed.State = Ptr(tidbcloud.ChangefeedStatePaused)
			return nil
		}).AnyTimes()

	s.EXPECT().ResumeChangefeed(gomock.Any(), changefeedId).DoAndReturn(
		func(_ context.Context, _ string) error {
			store.changefeed.State = Ptr(tidbcloud.ChangefeedStateRunning)
			return nil
		}).AnyTimes()

	s.EXPECT().DeleteChangefeed(gomock.Any(), changefeedId).DoAndReturn(
		func(_ context.Context, _ string) error {
			store.deleted = true
			return nil
		}).AnyTimes()

	return s, store
}

func TestUTDedicatedChangefeedResource(t *testing.T) {
	setupTestEnv()

	s, _ := newChangefeedMock(t)
	defer HookGlobal(&NewDedicatedClient, func(publicKey string, privateKey string, dedicatedEndpoint string, userAgent string) (tidbcloud.TiDBCloudDedicatedClient, error) {
		return s, nil
	})()

	changefeedResourceName := "tidbcloud_dedicated_changefeed.test"
	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read
			{
				Config: testUTDedicatedChangefeedResourceConfig("4rcu", "tidb-cdc", ""),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(changefeedResourceName, "changefeed_id", "cf-1"),
					resource.TestCheckResourceAttr(changefeedResourceName, "replication_capacity", "4rcu"),
					resource.TestCheckResourceAttr(changefeedResourceName, "downstream_type", "KAFKA"),
					resource.TestCheckResourceAttr(changefeedResourceName, "state", "RUNNING"),
					resource.TestCheckResourceAttr(changefeedResourceName, "kafka.topic_partition_config.default_topic", "tidb-cdc"),
				),
			},
			// Scale in place
			{
				Config: testUTDedicatedChangefeedResourceConfig("8rcu", "tidb-cdc", ""),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(changefeedResourceName, "replication_capacity", "8rcu"),
				),
			},
			// Edit the downstream config in place
			{
				Config: testUTDedicatedChangefeedResourceConfig("8rcu", "tidb-cdc-v2", ""),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(changefeedResourceName, "kafka.topic_partition_config.default_topic", "tidb-cdc-v2"),
				),
			},
			// Pause
			{
				Config: testUTDedicatedChangefeedResourceConfig("8rcu", "tidb-cdc-v2", "paused = true"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(changefeedResourceName, "paused", "true"),
					resource.TestCheckResourceAttr(changefeedResourceName, "state", "PAUSED"),
				),
			},
			// Resume
			{
				Config: testUTDedicatedChangefeedResourceConfig("8rcu", "tidb-cdc-v2", "paused = false"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(changefeedResourceName, "paused", "false"),
					resource.TestCheckResourceAttr(changefeedResourceName, "state", "RUNNING"),
				),
			},
			// Import
			{
				ResourceName:                         changefeedResourceName,
				ImportState:                          true,
				ImportStateId:                        "cf-1",
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "changefeed_id",
				// paused is preserved from state rather than derived from the
				// API outside of import, and the credentials are input-only.
				ImportStateVerifyIgnore: []string{"paused"},
			},
			// Delete is performed automatically by the test framework.
		},
	})
}

func TestUTDedicatedChangefeedResourceDownstreamMismatch(t *testing.T) {
	setupTestEnv()

	s, _ := newChangefeedMock(t)
	defer HookGlobal(&NewDedicatedClient, func(publicKey string, privateKey string, dedicatedEndpoint string, userAgent string) (tidbcloud.TiDBCloudDedicatedClient, error) {
		return s, nil
	})()

	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "tidbcloud_dedicated_changefeed" "test" {
	cluster_id           = "10001"
	name                 = "test-changefeed"
	replication_capacity = "4rcu"
	downstream_type      = "MYSQL"
	network_info = {
		network_type = "NETWORK_TYPE_PUBLIC"
	}
	start_position = {
		mode = "FROM_NOW"
	}
}
`,
				ExpectError: regexp.MustCompile("`mysql` must be configured"),
			},
		},
	})
}

func TestUTDedicatedChangefeedResourceFailedState(t *testing.T) {
	setupTestEnv()

	s, store := newChangefeedMock(t)
	defer HookGlobal(&NewDedicatedClient, func(publicKey string, privateKey string, dedicatedEndpoint string, userAgent string) (tidbcloud.TiDBCloudDedicatedClient, error) {
		return s, nil
	})()

	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testUTDedicatedChangefeedResourceConfig("4rcu", "tidb-cdc", ""),
			},
			// Simulate the changefeed failing out of band, then attempt an
			// in-place downstream edit: the resource must refuse with a clear
			// FAILED-state error instead of an opaque API failure.
			{
				PreConfig: func() {
					store.changefeed.State = Ptr(tidbcloud.ChangefeedStateFailed)
				},
				Config:      testUTDedicatedChangefeedResourceConfig("4rcu", "tidb-cdc-v2", ""),
				ExpectError: regexp.MustCompile("FAILED state and cannot be modified"),
			},
			// Recover the fake so the framework's automatic destroy works.
			{
				PreConfig: func() {
					store.changefeed.State = Ptr(tidbcloud.ChangefeedStateRunning)
				},
				Config: testUTDedicatedChangefeedResourceConfig("4rcu", "tidb-cdc", ""),
			},
		},
	})
}

// TestAccDedicatedChangefeedResource exercises the full changefeed lifecycle
// against a live cluster. It requires a running TiDB Cloud Dedicated cluster
// and a Kafka broker reachable from it, so it is gated behind extra
// environment variables in addition to TF_ACC.
func TestAccDedicatedChangefeedResource(t *testing.T) {
	clusterId := os.Getenv("TIDBCLOUD_DEDICATED_CLUSTER_ID")
	brokerEndpoints := os.Getenv("TIDBCLOUD_CHANGEFEED_KAFKA_BROKER_ENDPOINTS")
	if clusterId == "" || brokerEndpoints == "" {
		t.Skip("skipping: TIDBCLOUD_DEDICATED_CLUSTER_ID and TIDBCLOUD_CHANGEFEED_KAFKA_BROKER_ENDPOINTS are required")
	}

	changefeedResourceName := "tidbcloud_dedicated_changefeed.test"
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDedicatedChangefeedResourceConfig(clusterId, brokerEndpoints, "4rcu"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(changefeedResourceName, "changefeed_id"),
					resource.TestCheckResourceAttr(changefeedResourceName, "replication_capacity", "4rcu"),
					resource.TestCheckResourceAttrSet(changefeedResourceName, "state"),
				),
			},
			{
				Config: testAccDedicatedChangefeedResourceConfig(clusterId, brokerEndpoints, "8rcu"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(changefeedResourceName, "replication_capacity", "8rcu"),
				),
			},
		},
	})
}

func testUTDedicatedChangefeedResourceConfig(replicationCapacity, defaultTopic, extra string) string {
	return fmt.Sprintf(`
resource "tidbcloud_dedicated_changefeed" "test" {
	cluster_id           = "10001"
	name                 = "test-changefeed"
	replication_capacity = "%s"
	downstream_type      = "KAFKA"
	%s
	network_info = {
		network_type = "NETWORK_TYPE_PUBLIC"
	}
	start_position = {
		mode = "FROM_NOW"
	}
	kafka = {
		broker = {
			version          = "KAFKA_VERSION_3XX"
			broker_endpoints = "broker1:9092,broker2:9092"
		}
		authentication = {
			auth_type = "DISABLE"
		}
		data_format = {
			protocol = "PROTOCOL_CANAL_JSON"
		}
		topic_partition_config = {
			dispatch_type      = "DISPATCH_TYPE_ONE_TOPIC"
			default_topic      = "%s"
			replication_factor = 3
			partition_num      = 6
		}
	}
}
`, replicationCapacity, extra, defaultTopic)
}

func testAccDedicatedChangefeedResourceConfig(clusterId, brokerEndpoints, replicationCapacity string) string {
	return fmt.Sprintf(`
resource "tidbcloud_dedicated_changefeed" "test" {
	cluster_id           = "%s"
	name                 = "tf-acc-changefeed"
	replication_capacity = "%s"
	downstream_type      = "KAFKA"
	network_info = {
		network_type = "NETWORK_TYPE_PUBLIC"
	}
	start_position = {
		mode = "FROM_NOW"
	}
	kafka = {
		broker = {
			version          = "KAFKA_VERSION_3XX"
			broker_endpoints = "%s"
		}
		authentication = {
			auth_type = "DISABLE"
		}
		data_format = {
			protocol = "PROTOCOL_CANAL_JSON"
		}
		topic_partition_config = {
			dispatch_type      = "DISPATCH_TYPE_ONE_TOPIC"
			default_topic      = "tidb-cdc"
			replication_factor = 3
			partition_num      = 6
		}
	}
}
`, clusterId, replicationCapacity, brokerEndpoints)
}
