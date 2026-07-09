package provider

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	mockClient "github.com/tidbcloud/terraform-provider-tidbcloud/mock"
	"github.com/tidbcloud/terraform-provider-tidbcloud/tidbcloud"
)

func testUTChangefeed() *tidbcloud.Changefeed {
	return &tidbcloud.Changefeed{
		Id:                  Ptr("cf-1"),
		ClusterId:           "10001",
		Name:                "test-changefeed",
		ReplicationCapacity: "4rcu",
		DownstreamType:      tidbcloud.ChangefeedDownstreamTypeKafka,
		State:               Ptr(tidbcloud.ChangefeedStateRunning),
		CreateTime:          Ptr("2026-01-01T00:00:00Z"),
		CheckpointTso:       Ptr("457111558064308225"),
		CheckpointTs:        Ptr("2026-01-01T00:10:00Z"),
		NetworkInfo: &tidbcloud.ChangefeedNetworkInfo{
			NetworkType: "NETWORK_TYPE_PUBLIC",
		},
		StartPosition: &tidbcloud.ChangefeedStartPosition{
			Mode: "FROM_NOW",
		},
		Kafka: &tidbcloud.ChangefeedKafkaConfig{
			Broker: &tidbcloud.ChangefeedKafkaBroker{
				Version:         "KAFKA_VERSION_3XX",
				BrokerEndpoints: Ptr("broker1:9092"),
			},
			Authentication: &tidbcloud.ChangefeedKafkaAuthentication{
				AuthType: "DISABLE",
			},
			DataFormat: &tidbcloud.ChangefeedKafkaDataFormat{
				Protocol: "PROTOCOL_CANAL_JSON",
			},
			TopicPartitionConfig: &tidbcloud.ChangefeedKafkaTopicPartitionConfig{
				DispatchType:      "DISPATCH_TYPE_ONE_TOPIC",
				DefaultTopic:      Ptr("tidb-cdc"),
				ReplicationFactor: 3,
				PartitionNum:      6,
			},
		},
	}
}

func TestUTDedicatedChangefeedDataSource(t *testing.T) {
	setupTestEnv()

	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)
	s := mockClient.NewMockTiDBCloudDedicatedClient(ctrl)
	defer HookGlobal(&NewDedicatedClient, func(publicKey string, privateKey string, dedicatedEndpoint string, userAgent string) (tidbcloud.TiDBCloudDedicatedClient, error) {
		return s, nil
	})()

	s.EXPECT().GetChangefeed(gomock.Any(), "cf-1").Return(testUTChangefeed(), nil).AnyTimes()

	dataSourceName := "data.tidbcloud_dedicated_changefeed.test"
	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
data "tidbcloud_dedicated_changefeed" "test" {
	changefeed_id = "cf-1"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, "cluster_id", "10001"),
					resource.TestCheckResourceAttr(dataSourceName, "name", "test-changefeed"),
					resource.TestCheckResourceAttr(dataSourceName, "downstream_type", "KAFKA"),
					resource.TestCheckResourceAttr(dataSourceName, "state", "RUNNING"),
					resource.TestCheckResourceAttr(dataSourceName, "kafka.broker.broker_endpoints", "broker1:9092"),
					resource.TestCheckResourceAttr(dataSourceName, "kafka.topic_partition_config.partition_num", "6"),
					resource.TestCheckNoResourceAttr(dataSourceName, "mysql"),
				),
			},
		},
	})
}

func TestUTDedicatedChangefeedsDataSource(t *testing.T) {
	setupTestEnv()

	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)
	s := mockClient.NewMockTiDBCloudDedicatedClient(ctrl)
	defer HookGlobal(&NewDedicatedClient, func(publicKey string, privateKey string, dedicatedEndpoint string, userAgent string) (tidbcloud.TiDBCloudDedicatedClient, error) {
		return s, nil
	})()

	s.EXPECT().ListChangefeeds(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, params *tidbcloud.ListChangefeedsParams) (*tidbcloud.ListChangefeedsResponse, error) {
			if params.ClusterId != "10001" {
				return &tidbcloud.ListChangefeedsResponse{}, nil
			}
			return &tidbcloud.ListChangefeedsResponse{
				Changefeeds: []tidbcloud.Changefeed{*testUTChangefeed()},
				TotalSize:   Ptr(int32(1)),
			}, nil
		}).AnyTimes()

	dataSourceName := "data.tidbcloud_dedicated_changefeeds.test"
	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
data "tidbcloud_dedicated_changefeeds" "test" {
	cluster_id = "10001"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, "changefeeds.#", "1"),
					resource.TestCheckResourceAttr(dataSourceName, "changefeeds.0.changefeed_id", "cf-1"),
					resource.TestCheckResourceAttr(dataSourceName, "changefeeds.0.name", "test-changefeed"),
					resource.TestCheckResourceAttr(dataSourceName, "changefeeds.0.state", "RUNNING"),
				),
			},
		},
	})
}
