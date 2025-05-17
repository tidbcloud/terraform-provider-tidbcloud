package provider

import (
	"fmt"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	mockClient "github.com/tidbcloud/terraform-provider-tidbcloud/mock"
	"github.com/tidbcloud/terraform-provider-tidbcloud/tidbcloud"
	"github.com/tidbcloud/tidbcloud-cli/pkg/tidbcloud/v1beta1/dedicated"
)

func TestAccDedicatedNodeGroupResource(t *testing.T) {
	dedicatedNodeGroupResourceName := "tidbcloud_dedicated_node_group.test_group"
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccDedicatedNodeGroupsResourceConfig(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(dedicatedNodeGroupResourceName, "display_name", "test-node-group"),
					resource.TestCheckResourceAttr(dedicatedNodeGroupResourceName, "node_count", "1"),
				),
			},
			// Update testing
			{
				Config: testAccDedicatedNodeGroupsResourceConfig_update(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(dedicatedNodeGroupResourceName, "display_name", "test-node-group2"),
					resource.TestCheckResourceAttr(dedicatedNodeGroupResourceName, "node_count", "2"),
				),
			},
		},
	})
}

func TestUTDedicatedNodeGroupResource(t *testing.T) {
	setupTestEnv()

	ctrl := gomock.NewController(t)
	s := mockClient.NewMockTiDBCloudDedicatedClient(ctrl)
	defer HookGlobal(&NewDedicatedClient, func(publicKey string, privateKey string, dedicatedEndpoint string, userAgent string) (tidbcloud.TiDBCloudDedicatedClient, error) {
		return s, nil
	})()

	clusterId := "cluster_id"
	displayName := "test_group"
	nodeGroupId := "node_group_id"

	createNodeGroupResp := dedicated.Dedicatedv1beta1TidbNodeGroup{}
	createNodeGroupResp.UnmarshalJSON([]byte(testUTTidbCloudOpenApidedicatedv1beta1NodeGroup(clusterId, displayName, string(dedicated.DEDICATEDV1BETA1TIDBNODEGROUPSTATE_MODIFYING), 1)))
	getNodeGroupResp := dedicated.Dedicatedv1beta1TidbNodeGroup{}
	getNodeGroupResp.UnmarshalJSON([]byte(testUTTidbCloudOpenApidedicatedv1beta1NodeGroup(clusterId, displayName, string(dedicated.DEDICATEDV1BETA1TIDBNODEGROUPSTATE_ACTIVE), 1)))
	getNodeGroupAfterUpdateResp := dedicated.Dedicatedv1beta1TidbNodeGroup{}
	getNodeGroupAfterUpdateResp.UnmarshalJSON([]byte(testUTTidbCloudOpenApidedicatedv1beta1NodeGroup(clusterId, "test_group2", string(dedicated.COMMONV1BETA1CLUSTERSTATE_ACTIVE), 2)))
	updateNodeGroupSuccessResp := dedicated.Dedicatedv1beta1TidbNodeGroup{}
	updateNodeGroupSuccessResp.UnmarshalJSON([]byte(testUTTidbCloudOpenApidedicatedv1beta1NodeGroup(clusterId, "test_group2", string(dedicated.COMMONV1BETA1CLUSTERSTATE_MODIFYING), 2)))
	publicEndpointResp := dedicated.V1beta1PublicEndpointSetting{}
	publicEndpointResp.UnmarshalJSON([]byte(testUTV1beta1PublicEndpointSetting()))

	s.EXPECT().CreateTiDBNodeGroup(gomock.Any(), clusterId, gomock.Any()).Return(&createNodeGroupResp, nil)
	gomock.InOrder(
		s.EXPECT().GetTiDBNodeGroup(gomock.Any(), clusterId, nodeGroupId).Return(&getNodeGroupResp, nil).Times(3),
		s.EXPECT().GetTiDBNodeGroup(gomock.Any(), clusterId, nodeGroupId).Return(&getNodeGroupAfterUpdateResp, nil).Times(2),
	)
	s.EXPECT().UpdateTiDBNodeGroup(gomock.Any(), clusterId, nodeGroupId, gomock.Any()).Return(&updateNodeGroupSuccessResp, nil)
	s.EXPECT().DeleteTiDBNodeGroup(gomock.Any(), clusterId, gomock.Any()).Return(nil)

	s.EXPECT().GetPublicEndpoint(gomock.Any(), gomock.Any(), gomock.Any()).Return(&publicEndpointResp, nil).AnyTimes()
	s.EXPECT().UpdatePublicEndpoint(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(&publicEndpointResp, nil).AnyTimes()

	testDedicatedNodeGroupResource(t)
}

func testDedicatedNodeGroupResource(t *testing.T) {
	dedicatedNodeGroupResourceName := "tidbcloud_dedicated_node_group.test_group"
	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read serverless node group resource
			{
				Config: testUTDedicatedNodeGroupsResourceConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dedicatedNodeGroupResourceName, "display_name", "test_group"),
					resource.TestCheckResourceAttr(dedicatedNodeGroupResourceName, "node_count", "1"),
				),
			},
			// Update correctly
			{
				Config: testUTDedicatedNodeGroupsResourceUpdateConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dedicatedNodeGroupResourceName, "display_name", "test_group2"),
					resource.TestCheckResourceAttr(dedicatedNodeGroupResourceName, "node_count", "2"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func testUTDedicatedNodeGroupsResourceUpdateConfig() string {
	return `
resource "tidbcloud_dedicated_node_group" "test_group" {
    cluster_id = "cluster_id"
    node_count = 2
    display_name = "test_group2"
}
`
}

func testUTDedicatedNodeGroupsResourceConfig() string {
	return `
resource "tidbcloud_dedicated_node_group" "test_group" {
    cluster_id = "cluster_id"
    node_count = 1
    display_name = "test_group"
}
`
}

func testAccDedicatedNodeGroupsResourceConfig() string {
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
      storage_type = "Standard"
    }
}

resource "tidbcloud_dedicated_node_group" "test_group" {
    cluster_id = tidbcloud_dedicated_cluster.test.id
    node_count = 1
    display_name = "test-node-group"
}
`
}

func testAccDedicatedNodeGroupsResourceConfig_update() string {
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
      storage_type = "Standard"
    }
}

resource "tidbcloud_dedicated_node_group" "test_group" {
    cluster_id = tidbcloud_dedicated_cluster.test.id
    node_count = 2
    display_name = "test-node-group2"
}
`
}

func testUTTidbCloudOpenApidedicatedv1beta1NodeGroup(clusterId, displayName, state string, nodeCount int) string {
	return fmt.Sprintf(`{
    "name": "tidbNodeGroups/%s",
    "tidbNodeGroupId": "node_group_id",
    "clusterId": "%s",
    "displayName": "%s",
    "nodeCount": %d,
    "endpoints": [
        {
            "host": "",
            "port": 0,
            "connectionType": "PUBLIC"
        }
    ],
    "nodeSpecKey": "8C16G",
    "nodeSpecDisplayName": "8 vCPU, 16 GiB",
    "isDefaultGroup": false,
    "state": "%s",
    "nodeChangingProgress": {
        "matchingNodeSpecNodeCount": 0,
        "remainingDeletionNodeCount": 0
    }
}`, clusterId, clusterId, displayName, nodeCount, state)
}

func testUTV1beta1PublicEndpointSetting() string {
	return `
{
  "enabled": true,
  "ipAccessList": [
    {
      "cidrNotation": "0.0.0.0/32",
      "description": "test"
    }
  ]
}`
}
