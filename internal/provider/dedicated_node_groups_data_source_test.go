package provider

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	mockClient "github.com/tidbcloud/terraform-provider-tidbcloud/mock"
	"github.com/tidbcloud/terraform-provider-tidbcloud/tidbcloud"
	"github.com/tidbcloud/tidbcloud-cli/pkg/tidbcloud/v1beta1/dedicated"
)

func TestAccDedicatedNodeGroupsDataSource_basic(t *testing.T) {
	dedicatedNodeGroupsDataSourceName := "data.tidbcloud_dedicated_node_groups.test"
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDedicatedNodeGroupsDataSourceConfig(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(dedicatedNodeGroupsDataSourceName, "node_groups.#", "1"),
					resource.TestCheckResourceAttr(dedicatedNodeGroupsDataSourceName, "node_groups.0.node_count", "1"),
					resource.TestCheckResourceAttr(dedicatedNodeGroupsDataSourceName, "node_groups.0.display_name", "test-node-group"),
				),
			},
		},
	})
}

func TestUTDedicatedNodeGroupsDataSource(t *testing.T) {
	setupTestEnv()

	ctrl := gomock.NewController(t)
	s := mockClient.NewMockTiDBCloudDedicatedClient(ctrl)
	defer HookGlobal(&NewDedicatedClient, func(publicKey string, privateKey string, dedicatedEndpoint string, userAgent string) (tidbcloud.TiDBCloudDedicatedClient, error) {
		return s, nil
	})()

	var resp dedicated.Dedicatedv1beta1ListTidbNodeGroupsResponse
	resp.UnmarshalJSON([]byte(testUTListDedicatedv1beta1TidbNodeGroupResp))
    publicEndpointResp := dedicated.V1beta1PublicEndpointSetting{}
	publicEndpointResp.UnmarshalJSON([]byte(testUTV1beta1PublicEndpointSetting()))

	s.EXPECT().ListTiDBNodeGroups(gomock.Any(), gomock.Any(), gomock.Any(), nil).Return(&resp, nil).AnyTimes()
	s.EXPECT().GetPublicEndpoint(gomock.Any(), gomock.Any(), gomock.Any()).Return(&publicEndpointResp, nil).AnyTimes()

	testUTDedicatedNodeGroupsDataSource(t)
}

func testUTDedicatedNodeGroupsDataSource(t *testing.T) {
	dedicatedNodeGroupsDataSourceName := "data.tidbcloud_dedicated_node_groups.test"
	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testUTDedicatedNodeGroupsConfig(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(dedicatedNodeGroupsDataSourceName, "dedicated_node_groups.#", "0"),
				),
			},
		},
	})
}

func testUTDedicatedNodeGroupsConfig() string {
	return `
data "tidbcloud_dedicated_node_groups" "test" {
    cluster_id = "cluster_id"
}
`
}

func testAccDedicatedNodeGroupsDataSourceConfig() string {
	return `
resource "tidbcloud_dedicated_cluster" "test" {
    name = "test-tf"
    region_id = "aws-us-west-2"
    port = 4000
    root_password = "123456789"
    tidb_node_setting = {
      node_spec_key = "8C16G"
      node_count = 2
    }
    tikv_node_setting = {
      node_spec_key = "8C32G"
      node_count = 3
      storage_size_gi = 10
      storage_type = "BASIC"
    }
}

resource "tidbcloud_dedicated_node_group" "test_group" {
    cluster_id = tidbcloud_dedicated_cluster.test.id
    node_count = 1
    display_name = "test-node-group"
}

data "tidbcloud_dedicated_node_groups" "test" {
  cluster_id = tidbcloud_dedicated_cluster.test.id
}
`
}

const testUTListDedicatedv1beta1TidbNodeGroupResp = `
{
    "tidbNodeGroups": [
        {
            "name": "tidbNodeGroups/191",
            "tidbNodeGroupId": "191",
            "clusterId": "cluster_id",
            "displayName": "DefaultGroup",
            "nodeCount": 3,
            "endpoints": [
                {
                    "host": "tidb.xxx.clusters.dev.tidb-cloud.com",
                    "port": 4000,
                    "connectionType": "PUBLIC"
                },
                {
                    "host": "private-tidb.xxx.clusters.dev.tidb-cloud.com",
                    "port": 4000,
                    "connectionType": "VPC_PEERING"
                },
                {
                    "host": "privatelink-191.xxx.clusters.dev.tidb-cloud.com",
                    "port": 4000,
                    "connectionType": "PRIVATE_ENDPOINT"
                }
            ],
            "nodeSpecKey": "8C16G",
            "nodeSpecDisplayName": "8 vCPU, 16 GiB",
            "isDefaultGroup": true,
            "state": "ACTIVE"
        },
        {
            "name": "tidbNodeGroups/192",
            "tidbNodeGroupId": "192",
            "clusterId": "cluster_id",
            "displayName": "test-node-group2",
            "nodeCount": 2,
            "endpoints": [
                {
                    "host": "",
                    "port": 0,
                    "connectionType": "PUBLIC"
                },
                {
                    "host": "private-tidb.dzwx5w.clusters.dev.tidb-cloud.com",
                    "port": 4000,
                    "connectionType": "VPC_PEERING"
                },
                {
                    "host": "privatelink-191.dzw5w.clusters.dev.tidb-cloud.com",
                    "port": 4000,
                    "connectionType": "PRIVATE_ENDPOINT"
                }
            ],
            "nodeSpecKey": "8C16G",
            "nodeSpecDisplayName": "8 vCPU, 16 GiB",
            "isDefaultGroup": false,
            "state": "ACTIVE"
        }
    ]
}
`
