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
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
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
	// failPause makes PauseChangefeed return an error.
	failPause bool
	// failGetAfterPause makes the first GetChangefeed after each accepted
	// PauseChangefeed return an error, as a transient read failure would.
	failGetAfterPause bool
	failNextGet       bool
	// advanceCheckpoint moves the checkpoint on every GetChangefeed, as a
	// replicating changefeed does.
	advanceCheckpoint bool
	// call counters, for asserting that an operation was skipped
	pauseCalls  int
	resumeCalls int
	deleteCalls int
	scaleCalls  int
	checkpoints int
}

func (s *changefeedStore) get() *tidbcloud.Changefeed {
	cf := s.changefeed
	return &cf
}

// withoutChangefeedPasswords returns the downstream configs as the API stores
// and returns them: without the input-only passwords.
func withoutChangefeedPasswords(kafka *tidbcloud.ChangefeedKafkaConfig, mysql *tidbcloud.ChangefeedMySQLConfig) (*tidbcloud.ChangefeedKafkaConfig, *tidbcloud.ChangefeedMySQLConfig) {
	if kafka != nil && kafka.Authentication != nil {
		k, auth := *kafka, *kafka.Authentication
		auth.Password = nil
		k.Authentication = &auth
		kafka = &k
	}
	if mysql != nil && mysql.Connection != nil {
		m, conn := *mysql, *mysql.Connection
		conn.Password = nil
		m.Connection = &conn
		mysql = &m
	}
	return kafka, mysql
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
			store.changefeed.Kafka, store.changefeed.Mysql = withoutChangefeedPasswords(body.Changefeed.Kafka, body.Changefeed.Mysql)
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
			if store.failGet || store.failNextGet {
				store.failNextGet = false
				return nil, &tidbcloud.ChangefeedAPIError{StatusCode: http.StatusInternalServerError}
			}
			if store.deleted {
				return nil, &tidbcloud.ChangefeedAPIError{StatusCode: http.StatusNotFound}
			}
			if store.advanceCheckpoint {
				store.checkpoints++
				store.changefeed.CheckpointTso = Ptr(fmt.Sprintf("%d", store.checkpoints))
			}
			return store.get(), nil
		}).AnyTimes()

	s.EXPECT().ScaleChangefeed(gomock.Any(), changefeedId, gomock.Any()).DoAndReturn(
		func(_ context.Context, _ string, replicationCapacity string) (*tidbcloud.Changefeed, error) {
			// The real API only scales a RUNNING changefeed — enforce it so
			// the resource's resume -> scale / scale -> pause ordering is
			// actually exercised.
			if store.changefeed.State == nil || (*store.changefeed.State != tidbcloud.ChangefeedStateRunning && *store.changefeed.State != tidbcloud.ChangefeedStateWarning) {
				return nil, &tidbcloud.ChangefeedAPIError{StatusCode: http.StatusBadRequest}
			}
			store.scaleCalls++
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
			store.changefeed.Kafka, store.changefeed.Mysql = withoutChangefeedPasswords(body.Kafka, body.Mysql)
			return store.get(), nil
		}).AnyTimes()

	s.EXPECT().PauseChangefeed(gomock.Any(), changefeedId).DoAndReturn(
		func(_ context.Context, _ string) error {
			if store.failPause {
				return &tidbcloud.ChangefeedAPIError{StatusCode: http.StatusInternalServerError}
			}
			store.pauseCalls++
			store.changefeed.State = Ptr(tidbcloud.ChangefeedStatePaused)
			store.failNextGet = store.failGetAfterPause
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
			// A resume combined with a capacity change is sequenced: resume,
			// then scale (the mock rejects a scale on a paused feed)
			{
				Config: testUTDedicatedChangefeedResourceConfig("16rcu", "tidb-cdc-v4", "paused = false"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(changefeedResourceName, "replication_capacity", "16rcu"),
					resource.TestCheckResourceAttr(changefeedResourceName, "paused", "false"),
					resource.TestCheckResourceAttr(changefeedResourceName, "state", "RUNNING"),
				),
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
			// A capacity change combined with a pause is sequenced: scale
			// while still running, then pause
			{
				Config: testUTDedicatedChangefeedResourceConfig("16rcu", "tidb-cdc-v5", "paused = true"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(changefeedResourceName, "replication_capacity", "16rcu"),
					resource.TestCheckResourceAttr(changefeedResourceName, "paused", "true"),
					resource.TestCheckResourceAttr(changefeedResourceName, "state", "PAUSED"),
				),
			},
			// The one combination that cannot be sequenced — scaling a feed
			// that is paused and stays paused — is rejected at plan time
			{
				Config:      testUTDedicatedChangefeedResourceConfig("32rcu", "tidb-cdc-v5", "paused = true"),
				ExpectError: regexp.MustCompile(`paused and stays paused`),
			},
			// Resumed out of band while the state still said paused: the
			// refresh picks up RUNNING, so a capacity change with
			// paused = false is a plain scale (no resume call)
			{
				PreConfig: func() {
					store.changefeed.State = Ptr(tidbcloud.ChangefeedStateRunning)
					resumeCallsBefore = store.resumeCalls
				},
				Config: testUTDedicatedChangefeedResourceConfig("32rcu", "tidb-cdc-v5", "paused = false"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(changefeedResourceName, "replication_capacity", "32rcu"),
					resource.TestCheckResourceAttr(changefeedResourceName, "state", "RUNNING"),
					func(_ *terraform.State) error {
						if store.resumeCalls != resumeCallsBefore {
							return fmt.Errorf("ResumeChangefeed was called %d times on an already-running changefeed", store.resumeCalls-resumeCallsBefore)
						}
						return nil
					},
				),
			},
			// Paused out of band while the config says paused = false: the
			// refresh shows the drift and the apply resumes it
			{
				PreConfig: func() {
					store.changefeed.State = Ptr(tidbcloud.ChangefeedStatePaused)
					resumeCallsBefore = store.resumeCalls
				},
				Config: testUTDedicatedChangefeedResourceConfig("32rcu", "tidb-cdc-v5", "paused = false"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(changefeedResourceName, "state", "RUNNING"),
					func(_ *terraform.State) error {
						if store.resumeCalls != resumeCallsBefore+1 {
							return fmt.Errorf("want 1 ResumeChangefeed call to revert the out-of-band pause, got %d", store.resumeCalls-resumeCallsBefore)
						}
						return nil
					},
				),
			},
			// paused unset: the pause state is not managed. A feed paused out
			// of band stays paused through an unrelated edit
			{
				PreConfig: func() {
					store.changefeed.State = Ptr(tidbcloud.ChangefeedStatePaused)
					resumeCallsBefore = store.resumeCalls
				},
				Config: testUTDedicatedChangefeedResourceConfig("32rcu", "tidb-cdc-v6", ""),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(changefeedResourceName, "kafka.topic_partition_config.default_topic", "tidb-cdc-v6"),
					resource.TestCheckResourceAttr(changefeedResourceName, "paused", "true"),
					resource.TestCheckResourceAttr(changefeedResourceName, "state", "PAUSED"),
					func(_ *terraform.State) error {
						if store.resumeCalls != resumeCallsBefore {
							return fmt.Errorf("ResumeChangefeed was called %d times with paused unmanaged", store.resumeCalls-resumeCallsBefore)
						}
						return nil
					},
				),
			},
			// ... and a feed resumed out of band plans as no change
			{
				PreConfig: func() { store.changefeed.State = Ptr(tidbcloud.ChangefeedStateRunning) },
				Config:    testUTDedicatedChangefeedResourceConfig("32rcu", "tidb-cdc-v6", ""),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(changefeedResourceName, "paused", "false"),
					resource.TestCheckResourceAttr(changefeedResourceName, "state", "RUNNING"),
				),
			},
			// Import: paused is derived from the live state on every refresh,
			// so it round-trips like any other attribute
			{
				ResourceName:                         changefeedResourceName,
				ImportState:                          true,
				ImportStateId:                        "cf-1",
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "changefeed_id",
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

// TestUTDedicatedChangefeedResourceCreatePaused creates a changefeed with
// paused = true: Create waits for RUNNING, pauses once and waits for PAUSED.
func TestUTDedicatedChangefeedResourceCreatePaused(t *testing.T) {
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
				Config: testUTDedicatedChangefeedResourceConfig("4rcu", "tidb-cdc", "paused = true"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(changefeedResourceName, "changefeed_id", "cf-1"),
					resource.TestCheckResourceAttr(changefeedResourceName, "paused", "true"),
					resource.TestCheckResourceAttr(changefeedResourceName, "state", "PAUSED"),
					func(_ *terraform.State) error {
						if store.pauseCalls != 1 || store.resumeCalls != 0 {
							return fmt.Errorf("expected 1 PauseChangefeed and 0 ResumeChangefeed calls, got %d and %d", store.pauseCalls, store.resumeCalls)
						}
						return nil
					},
				),
			},
		},
	})
}

// TestUTDedicatedChangefeedResourceCreatePausedPauseFailure fails the pause of
// a paused = true create: the apply must fail and keep the created changefeed
// in (tainted) state, so the next apply replaces it instead of creating a
// second one.
func TestUTDedicatedChangefeedResourceCreatePausedPauseFailure(t *testing.T) {
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
				PreConfig:   func() { store.failPause = true },
				Config:      testUTDedicatedChangefeedResourceConfig("4rcu", "tidb-cdc", "paused = true"),
				ExpectError: regexp.MustCompile("Unable to call PauseChangefeed"),
			},
			{
				PreConfig: func() { store.failPause = false },
				Config:    testUTDedicatedChangefeedResourceConfig("4rcu", "tidb-cdc", "paused = true"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(changefeedResourceName, "paused", "true"),
					resource.TestCheckResourceAttr(changefeedResourceName, "state", "PAUSED"),
				),
			},
		},
	})
	// One delete for the tainted replace, one for the final destroy.
	if store.deleteCalls != 2 {
		t.Fatalf("expected 2 DeleteChangefeed calls (tainted replace + destroy), got %d", store.deleteCalls)
	}
}

// TestUTDedicatedChangefeedResourcePausedDrift reproduces an out-of-band
// pause: refresh must surface the live PAUSED state as paused = true, so the
// unchanged configuration (paused = false) plans a resume and the apply
// restores the declared running state. (With paused unset the pause state is
// not managed and the drift is only reported; see TestUTDedicatedChangefeedResource.)
func TestUTDedicatedChangefeedResourcePausedDrift(t *testing.T) {
	setupTestEnv()

	s, store := newChangefeedMock(t)
	defer HookGlobal(&NewDedicatedClient, func(publicKey string, privateKey string, dedicatedEndpoint string, userAgent string) (tidbcloud.TiDBCloudDedicatedClient, error) {
		return s, nil
	})()

	changefeedResourceName := "tidbcloud_dedicated_changefeed.test"
	resumeCallsBefore := 0
	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testUTDedicatedChangefeedResourceConfig("4rcu", "tidb-cdc", "paused = false"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(changefeedResourceName, "paused", "false"),
				),
			},
			// The changefeed is paused out of band: the refresh reports the
			// drift and the apply resumes it back to the declared state.
			{
				PreConfig: func() {
					store.changefeed.State = Ptr(tidbcloud.ChangefeedStatePaused)
					resumeCallsBefore = store.resumeCalls
				},
				Config: testUTDedicatedChangefeedResourceConfig("4rcu", "tidb-cdc", "paused = false"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(changefeedResourceName, "paused", "false"),
					resource.TestCheckResourceAttr(changefeedResourceName, "state", "RUNNING"),
					func(_ *terraform.State) error {
						if store.resumeCalls != resumeCallsBefore+1 {
							return fmt.Errorf("expected exactly one ResumeChangefeed call for the drift, got %d", store.resumeCalls-resumeCallsBefore)
						}
						return nil
					},
				),
			},
		},
	})
}

// TestUTDedicatedChangefeedResourceComputedFieldsKnownOnUpdate checks that an
// update plan keeps the refreshed computed attributes instead of showing them
// as "(known after apply)", while the checkpoint keeps moving: the apply must
// not report an inconsistent result. Only a change of paused leaves state
// unknown.
func TestUTDedicatedChangefeedResourceComputedFieldsKnownOnUpdate(t *testing.T) {
	setupTestEnv()

	s, store := newChangefeedMock(t)
	defer HookGlobal(&NewDedicatedClient, func(publicKey string, privateKey string, dedicatedEndpoint string, userAgent string) (tidbcloud.TiDBCloudDedicatedClient, error) {
		return s, nil
	})()
	store.advanceCheckpoint = true

	changefeedResourceName := "tidbcloud_dedicated_changefeed.test"
	knownComputed := func(attrs ...string) []plancheck.PlanCheck {
		var checks []plancheck.PlanCheck
		for _, a := range attrs {
			checks = append(checks, plancheck.ExpectKnownValue(changefeedResourceName, tfjsonpath.New(a), knownvalue.NotNull()))
		}
		return append(checks, plancheck.ExpectKnownValue(changefeedResourceName, tfjsonpath.New("error"), knownvalue.Null()))
	}
	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testUTDedicatedChangefeedResourceConfig("4rcu", "tidb-cdc", ""),
			},
			// Edit the downstream config with paused unset: the feed is
			// paused for the edit and resumed, and no computed attribute is
			// unknown in the plan.
			{
				Config: testUTDedicatedChangefeedResourceConfig("4rcu", "tidb-cdc-v2", ""),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: append(knownComputed("state", "create_time", "checkpoint_tso", "checkpoint_ts"),
						plancheck.ExpectResourceAction(changefeedResourceName, plancheck.ResourceActionUpdate)),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(changefeedResourceName, "kafka.topic_partition_config.default_topic", "tidb-cdc-v2"),
					resource.TestCheckResourceAttr(changefeedResourceName, "state", "RUNNING"),
				),
			},
			// Pausing changes state, so only state is unknown.
			{
				Config: testUTDedicatedChangefeedResourceConfig("4rcu", "tidb-cdc-v2", "paused = true"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: append(knownComputed("create_time", "checkpoint_tso", "checkpoint_ts"),
						plancheck.ExpectUnknownValue(changefeedResourceName, tfjsonpath.New("state"))),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(changefeedResourceName, "state", "PAUSED"),
				),
			},
			// Editing with paused set and unchanged keeps state known.
			{
				Config: testUTDedicatedChangefeedResourceConfig("4rcu", "tidb-cdc-v3", "paused = true"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: knownComputed("state", "create_time", "checkpoint_tso", "checkpoint_ts"),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(changefeedResourceName, "kafka.topic_partition_config.default_topic", "tidb-cdc-v3"),
					resource.TestCheckResourceAttr(changefeedResourceName, "state", "PAUSED"),
				),
			},
		},
	})
}

// TestUTDedicatedChangefeedResourceDownstreamDrift reproduces an out-of-band
// edit of a readable downstream field: the MySQL username is changed through
// the API, the refresh must surface it as drift, and the next apply must
// restore the configured value via the pause -> edit -> resume sequence. The
// input-only password must survive the refresh untouched.
func TestUTDedicatedChangefeedResourceDownstreamDrift(t *testing.T) {
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
				Config: testUTDedicatedChangefeedResourceMysqlConfig("app-writer"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(changefeedResourceName, "mysql.connection.username", "app-writer"),
				),
			},
			// The username is changed through the API: the refresh reports
			// the drift and the apply edits the downstream config back to the
			// declared value, keeping the input-only password intact.
			{
				PreConfig: func() {
					store.changefeed.Mysql.Connection.Username = "changed-out-of-band"
				},
				Config: testUTDedicatedChangefeedResourceMysqlConfig("app-writer"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(changefeedResourceName, "mysql.connection.username", "app-writer"),
					resource.TestCheckResourceAttr(changefeedResourceName, "mysql.connection.password", "pw-secret"),
					resource.TestCheckResourceAttr(changefeedResourceName, "state", "RUNNING"),
					func(_ *terraform.State) error {
						if got := store.changefeed.Mysql.Connection.Username; got != "app-writer" {
							return fmt.Errorf("expected the drifted username to be edited back to %q, live value is %q", "app-writer", got)
						}
						return nil
					},
				),
			},
		},
	})
}

// TestUTDedicatedChangefeedResourceKafkaSaslPassword creates a Kafka
// changefeed with SASL credentials. The API never returns the password, so it
// must be kept from the prior state on every read, without a diff.
func TestUTDedicatedChangefeedResourceKafkaSaslPassword(t *testing.T) {
	setupTestEnv()

	s, store := newChangefeedMock(t)
	defer HookGlobal(&NewDedicatedClient, func(publicKey string, privateKey string, dedicatedEndpoint string, userAgent string) (tidbcloud.TiDBCloudDedicatedClient, error) {
		return s, nil
	})()

	changefeedResourceName := "tidbcloud_dedicated_changefeed.test"
	checkCredentials := resource.ComposeAggregateTestCheckFunc(
		resource.TestCheckResourceAttr(changefeedResourceName, "kafka.authentication.auth_type", "SASL_PLAIN"),
		resource.TestCheckResourceAttr(changefeedResourceName, "kafka.authentication.username", "cdc-writer"),
		resource.TestCheckResourceAttr(changefeedResourceName, "kafka.authentication.password", "kafka-secret"),
		func(_ *terraform.State) error {
			if got := store.changefeed.Kafka.Authentication.Password; got != nil {
				return fmt.Errorf("the fake API must not return the password, got %q", *got)
			}
			return nil
		},
	)
	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testUTDedicatedChangefeedResourceKafkaSaslConfig(),
				Check:  checkCredentials,
			},
			{
				RefreshState: true,
				Check:        checkCredentials,
			},
			{
				Config: testUTDedicatedChangefeedResourceKafkaSaslConfig(),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
		},
	})
}

// TestUTDedicatedChangefeedResourceServerDefaultNoDiff guards the other half
// of the read-time reconciliation: an attribute the configuration leaves
// unset (kafka.broker.use_tls) that the API starts reporting with a
// server-side default must stay null in state and produce no diff, i.e. no
// spurious pause/edit/resume cycle.
func TestUTDedicatedChangefeedResourceServerDefaultNoDiff(t *testing.T) {
	setupTestEnv()

	s, store := newChangefeedMock(t)
	defer HookGlobal(&NewDedicatedClient, func(publicKey string, privateKey string, dedicatedEndpoint string, userAgent string) (tidbcloud.TiDBCloudDedicatedClient, error) {
		return s, nil
	})()

	changefeedResourceName := "tidbcloud_dedicated_changefeed.test"
	pauseCallsBefore := 0
	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// use_tls is not configured.
			{
				Config: testUTDedicatedChangefeedResourceConfig("4rcu", "tidb-cdc", ""),
			},
			// The API now reports a server-side default for use_tls: the
			// refresh must keep the unset attribute null, so the unchanged
			// configuration plans no update at all.
			{
				PreConfig: func() {
					store.changefeed.Kafka.Broker.UseTls = Ptr(false)
					pauseCallsBefore = store.pauseCalls
				},
				Config: testUTDedicatedChangefeedResourceConfig("4rcu", "tidb-cdc", ""),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckNoResourceAttr(changefeedResourceName, "kafka.broker.use_tls"),
					func(_ *terraform.State) error {
						if store.pauseCalls != pauseCallsBefore {
							return fmt.Errorf("a server-side default triggered a downstream edit: PauseChangefeed called %d times", store.pauseCalls-pauseCallsBefore)
						}
						return nil
					},
				),
			},
		},
	})
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

// TestUTDedicatedChangefeedResourcePauseWaitFailureResumes fails the read that
// follows the temporary pause of a downstream edit, with paused unset: the
// pause was accepted, so the recovery must resume the changefeed. Left paused,
// the retried apply would take the paused state for the unmanaged one and
// never resume it.
func TestUTDedicatedChangefeedResourcePauseWaitFailureResumes(t *testing.T) {
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
			{
				PreConfig:   func() { store.failGetAfterPause = true },
				Config:      testUTDedicatedChangefeedResourceConfig("4rcu", "tidb-cdc-v2", ""),
				ExpectError: regexp.MustCompile("is not paused"),
			},
			// The retried apply edits a running changefeed and leaves it
			// running.
			{
				PreConfig: func() {
					store.failGetAfterPause = false
					if store.pauseCalls != 1 || store.resumeCalls != 1 {
						t.Errorf("expected the failed apply to pause and resume once, got %d PauseChangefeed and %d ResumeChangefeed calls", store.pauseCalls, store.resumeCalls)
					}
				},
				Config: testUTDedicatedChangefeedResourceConfig("4rcu", "tidb-cdc-v2", ""),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(changefeedResourceName, "kafka.topic_partition_config.default_topic", "tidb-cdc-v2"),
					resource.TestCheckResourceAttr(changefeedResourceName, "paused", "false"),
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

func testUTDedicatedChangefeedResourceMysqlConfig(username string) string {
	return fmt.Sprintf(`
resource "tidbcloud_dedicated_changefeed" "test" {
	cluster_id           = "10001"
	name                 = "test-changefeed-mysql"
	replication_capacity = "4rcu"
	downstream_type      = "MYSQL"
	network_info = {
		network_type = "NETWORK_TYPE_PUBLIC"
	}
	start_position = {
		mode = "FROM_NOW"
	}
	mysql = {
		connection = {
			endpoint = "mysql.example.com:3306"
			username = "%s"
			password = "pw-secret"
		}
	}
}
`, username)
}

func testUTDedicatedChangefeedResourceKafkaSaslConfig() string {
	return `
resource "tidbcloud_dedicated_changefeed" "test" {
	cluster_id           = "10001"
	name                 = "test-changefeed-sasl"
	replication_capacity = "4rcu"
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
			broker_endpoints = "broker1:9092,broker2:9092"
		}
		authentication = {
			auth_type = "SASL_PLAIN"
			username  = "cdc-writer"
			password  = "kafka-secret"
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
`
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
		currentlyPaused  bool
		willBePaused     bool
		wantErr          bool
	}{
		{name: "scale while paused and staying paused is rejected", capacityChanging: true, currentlyPaused: true, willBePaused: true, wantErr: true},
		{name: "scale with a resume is sequenced", capacityChanging: true, currentlyPaused: true, willBePaused: false, wantErr: false},
		{name: "scale with a pause is sequenced", capacityChanging: true, currentlyPaused: false, willBePaused: true, wantErr: false},
		{name: "scale while running is allowed", capacityChanging: true, currentlyPaused: false, willBePaused: false, wantErr: false},
		{name: "no scale while paused is fine", capacityChanging: false, currentlyPaused: true, willBePaused: true, wantErr: false},
		{name: "no scale while running is fine", capacityChanging: false, currentlyPaused: false, willBePaused: false, wantErr: false},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			err := scaleRequiresRunningError(tt.capacityChanging, tt.currentlyPaused, tt.willBePaused)
			if (err != nil) != tt.wantErr {
				t.Fatalf("scaleRequiresRunningError(%v,%v,%v) err=%v, wantErr=%v", tt.capacityChanging, tt.currentlyPaused, tt.willBePaused, err, tt.wantErr)
			}
		})
	}
}

func TestUTPausedFromChangefeedState(t *testing.T) {
	cases := []struct {
		state      *string
		wantPaused bool
		wantOK     bool
	}{
		{Ptr(tidbcloud.ChangefeedStatePaused), true, true},
		{Ptr(tidbcloud.ChangefeedStatePausing), true, true},
		{Ptr(tidbcloud.ChangefeedStateEditing), true, true},
		{Ptr(tidbcloud.ChangefeedStateRunning), false, true},
		{Ptr(tidbcloud.ChangefeedStateWarning), false, true},
		{Ptr(tidbcloud.ChangefeedStateScaling), false, true},
		{Ptr(tidbcloud.ChangefeedStateCreating), false, true},
		{Ptr(tidbcloud.ChangefeedStateFailed), false, false},
		{Ptr(tidbcloud.ChangefeedStateError), false, false},
		{Ptr(tidbcloud.ChangefeedStateDeleting), false, false},
		{nil, false, false},
	}
	for _, tt := range cases {
		name := "nil"
		if tt.state != nil {
			name = *tt.state
		}
		t.Run(name, func(t *testing.T) {
			paused, ok := pausedFromChangefeedState(tt.state)
			if paused != tt.wantPaused || ok != tt.wantOK {
				t.Fatalf("pausedFromChangefeedState(%s) = (%v, %v), want (%v, %v)", name, paused, ok, tt.wantPaused, tt.wantOK)
			}
		})
	}
}
