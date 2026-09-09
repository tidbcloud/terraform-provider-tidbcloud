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
	"github.com/hashicorp/terraform-plugin-testing/terraform"
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
	// failEdit makes EditChangefeedDownstreamConfig return an error, for
	// exercising the failure-path resume.
	failEdit bool
	// failGet makes GetChangefeed return an error, for exercising the
	// create-succeeded-but-read-failed path.
	failGet bool
	// call counters, for asserting that an operation was skipped
	pauseCalls  int
	resumeCalls int
	deleteCalls int
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
			if store.failGet {
				return nil, &tidbcloud.ChangefeedAPIError{StatusCode: http.StatusInternalServerError}
			}
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
			if store.failEdit {
				return nil, &tidbcloud.ChangefeedAPIError{StatusCode: http.StatusInternalServerError}
			}
			store.changefeed.TableConfig = body.TableConfig
			store.changefeed.Kafka = body.Kafka
			store.changefeed.Mysql = body.Mysql
			return store.get(), nil
		}).AnyTimes()

	s.EXPECT().PauseChangefeed(gomock.Any(), changefeedId).DoAndReturn(
		func(_ context.Context, _ string) error {
			store.pauseCalls++
			store.changefeed.State = Ptr(tidbcloud.ChangefeedStatePaused)
			return nil
		}).AnyTimes()

	s.EXPECT().ResumeChangefeed(gomock.Any(), changefeedId).DoAndReturn(
		func(_ context.Context, _ string) error {
			store.resumeCalls++
			store.changefeed.State = Ptr(tidbcloud.ChangefeedStateRunning)
			return nil
		}).AnyTimes()

	s.EXPECT().DeleteChangefeed(gomock.Any(), changefeedId).DoAndReturn(
		func(_ context.Context, _ string) error {
			store.deleted = true
			store.deleteCalls++
			return nil
		}).AnyTimes()

	return s, store
}

func TestUTDedicatedChangefeedResource(t *testing.T) {
	setupTestEnv()

	s, store := newChangefeedMock(t)
	defer HookGlobal(&NewDedicatedClient, func(publicKey string, privateKey string, dedicatedEndpoint string, userAgent string) (tidbcloud.TiDBCloudDedicatedClient, error) {
		return s, nil
	})()

	changefeedResourceName := "tidbcloud_dedicated_changefeed.test"
	var pauseCallsBefore, resumeCallsBefore int
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
			// Edit the downstream config and resume in the same update: the
			// edit applies while the feed is still paused, the resume runs
			// last (the mock rejects an edit on a non-paused feed, so the
			// ordering is actually exercised)
			{
				Config: testUTDedicatedChangefeedResourceConfig("8rcu", "tidb-cdc-v3", "paused = false"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(changefeedResourceName, "kafka.topic_partition_config.default_topic", "tidb-cdc-v3"),
					resource.TestCheckResourceAttr(changefeedResourceName, "paused", "false"),
					resource.TestCheckResourceAttr(changefeedResourceName, "state", "RUNNING"),
				),
			},
			// Pause and edit in the same update: the pause runs first and the
			// feed stays paused after the edit
			{
				Config: testUTDedicatedChangefeedResourceConfig("8rcu", "tidb-cdc-v4", "paused = true"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(changefeedResourceName, "kafka.topic_partition_config.default_topic", "tidb-cdc-v4"),
					resource.TestCheckResourceAttr(changefeedResourceName, "paused", "true"),
					resource.TestCheckResourceAttr(changefeedResourceName, "state", "PAUSED"),
				),
			},
			// A pause-state flip combined with a capacity change stays
			// rejected: scaling requires RUNNING, so the two cannot be
			// sequenced in one update
			{
				Config:      testUTDedicatedChangefeedResourceConfig("16rcu", "tidb-cdc-v4", "paused = false"),
				ExpectError: regexp.MustCompile(`Cannot change changefeed pause state along with[\s\n]+replication_capacity`),
			},
			// Out-of-band recovery: the live feed was edited and resumed by
			// hand after a failed apply; the same edit+resume config still
			// applies (pause -> re-edit -> resume)
			{
				PreConfig: func() { store.changefeed.State = Ptr(tidbcloud.ChangefeedStateRunning) },
				Config:    testUTDedicatedChangefeedResourceConfig("8rcu", "tidb-cdc-v5", "paused = false"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(changefeedResourceName, "kafka.topic_partition_config.default_topic", "tidb-cdc-v5"),
					resource.TestCheckResourceAttr(changefeedResourceName, "state", "RUNNING"),
				),
			},
			// A flip to paused skips the API call when the live feed is
			// already paused out of band
			{
				PreConfig: func() {
					store.changefeed.State = Ptr(tidbcloud.ChangefeedStatePaused)
					pauseCallsBefore = store.pauseCalls
				},
				Config: testUTDedicatedChangefeedResourceConfig("8rcu", "tidb-cdc-v5", "paused = true"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(changefeedResourceName, "state", "PAUSED"),
					func(_ *terraform.State) error {
						if store.pauseCalls != pauseCallsBefore {
							return fmt.Errorf("PauseChangefeed was called %d times on an already-paused changefeed", store.pauseCalls-pauseCallsBefore)
						}
						return nil
					},
				),
			},
			// ... and a flip to running skips the call when it is already
			// running out of band
			{
				PreConfig: func() {
					store.changefeed.State = Ptr(tidbcloud.ChangefeedStateRunning)
					resumeCallsBefore = store.resumeCalls
				},
				Config: testUTDedicatedChangefeedResourceConfig("8rcu", "tidb-cdc-v5", "paused = false"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(changefeedResourceName, "state", "RUNNING"),
					func(_ *terraform.State) error {
						if store.resumeCalls != resumeCallsBefore {
							return fmt.Errorf("ResumeChangefeed was called %d times on an already-running changefeed", store.resumeCalls-resumeCallsBefore)
						}
						return nil
					},
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

// TestUTDedicatedChangefeedResourceCreateKeepsStateOnReadFailure reproduces
// the create-succeeded-but-readiness-read-failed path: CreateChangefeed
// returns cf-1, the waiter's first GetChangefeed errors. The apply must fail
// AND the created changefeed must be persisted to state, so the next apply
// replaces the tracked (tainted) changefeed instead of leaving an unmanaged
// one behind and creating a second one.
func TestUTDedicatedChangefeedResourceCreateKeepsStateOnReadFailure(t *testing.T) {
	setupTestEnv()

	s, store := newChangefeedMock(t)
	defer HookGlobal(&NewDedicatedClient, func(publicKey string, privateKey string, dedicatedEndpoint string, userAgent string) (tidbcloud.TiDBCloudDedicatedClient, error) {
		return s, nil
	})()

	changefeedResourceName := "tidbcloud_dedicated_changefeed.test"
	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// CreateChangefeed succeeds, the readiness GetChangefeed fails:
			// the step errors, but cf-1 is now tracked in (tainted) state.
			{
				PreConfig:   func() { store.failGet = true },
				Config:      testUTDedicatedChangefeedResourceConfig("4rcu", "tidb-cdc", ""),
				ExpectError: regexp.MustCompile("is not ready"),
			},
			// With reads working again, the tainted changefeed is replaced:
			// the tracked cf-1 is deleted first, then created cleanly.
			{
				PreConfig: func() { store.failGet = false },
				Config:    testUTDedicatedChangefeedResourceConfig("4rcu", "tidb-cdc", ""),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(changefeedResourceName, "changefeed_id", "cf-1"),
					resource.TestCheckResourceAttr(changefeedResourceName, "state", "RUNNING"),
				),
			},
		},
	})
	// Two deletes prove step 1 persisted state: one for the tainted replace,
	// one for the framework's final destroy. Without the persisted state the
	// first changefeed is never tracked, so only the final destroy deletes.
	if store.deleteCalls != 2 {
		t.Fatalf("expected 2 DeleteChangefeed calls (tainted replace + destroy), got %d", store.deleteCalls)
	}
}

func TestUTDedicatedChangefeedResourceEditFailureResumes(t *testing.T) {
	setupTestEnv()

	s, store := newChangefeedMock(t)
	defer HookGlobal(&NewDedicatedClient, func(publicKey string, privateKey string, dedicatedEndpoint string, userAgent string) (tidbcloud.TiDBCloudDedicatedClient, error) {
		return s, nil
	})()

	changefeedResourceName := "tidbcloud_dedicated_changefeed.test"
	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testUTDedicatedChangefeedResourceConfig("4rcu", "tidb-cdc", ""),
			},
			// A failing downstream edit must not leave the changefeed paused:
			// the resource pauses it for the edit, the edit blows up, and the
			// best-effort recovery resumes it.
			{
				PreConfig: func() {
					store.failEdit = true
				},
				Config:      testUTDedicatedChangefeedResourceConfig("4rcu", "tidb-cdc-v2", ""),
				ExpectError: regexp.MustCompile("Unable to call EditChangefeedDownstreamConfig"),
			},
			// The refresh in this step reads the live state: RUNNING proves
			// the failure-path resume ran (it would be PAUSED otherwise).
			{
				PreConfig: func() {
					store.failEdit = false
				},
				Config: testUTDedicatedChangefeedResourceConfig("4rcu", "tidb-cdc", ""),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(changefeedResourceName, "state", "RUNNING"),
				),
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

func TestUTScaleRequiresRunningError(t *testing.T) {
	cases := []struct {
		name             string
		capacityChanging bool
		willBePaused     bool
		wantErr          bool
	}{
		{name: "scale while paused is rejected", capacityChanging: true, willBePaused: true, wantErr: true},
		{name: "scale while running is allowed", capacityChanging: true, willBePaused: false, wantErr: false},
		{name: "no scale while paused is fine", capacityChanging: false, willBePaused: true, wantErr: false},
		{name: "no scale while running is fine", capacityChanging: false, willBePaused: false, wantErr: false},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			err := scaleRequiresRunningError(tt.capacityChanging, tt.willBePaused)
			if (err != nil) != tt.wantErr {
				t.Fatalf("scaleRequiresRunningError(%v,%v) err=%v, wantErr=%v", tt.capacityChanging, tt.willBePaused, err, tt.wantErr)
			}
		})
	}
}
