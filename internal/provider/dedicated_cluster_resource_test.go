package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	mockClient "github.com/tidbcloud/terraform-provider-tidbcloud/mock"
	"github.com/tidbcloud/terraform-provider-tidbcloud/tidbcloud"
	"github.com/tidbcloud/tidbcloud-cli/pkg/tidbcloud/v1beta1/dedicated"
)

func TestAccDedicatedClusterResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccDedicatedClusterResourceConfig(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("tidbcloud_dedicated_cluster.test", "display_name", "test-tf"),
					resource.TestCheckResourceAttr("tidbcloud_dedicated_cluster.test", "region_id", "aws-us-west-2"),
					resource.TestCheckResourceAttr("tidbcloud_dedicated_cluster.test", "port", "4000"),
					resource.TestCheckResourceAttr("tidbcloud_dedicated_cluster.test", "root_password", "123456789"),
					resource.TestCheckResourceAttr("tidbcloud_dedicated_cluster.test", "tidb_node_setting.node_spec_key", "2C4G"),
					resource.TestCheckResourceAttr("tidbcloud_dedicated_cluster.test", "tidb_node_setting.node_count", "1"),
					resource.TestCheckResourceAttr("tidbcloud_dedicated_cluster.test", "tikv_node_setting.node_spec_key", "2C4G"),
					resource.TestCheckResourceAttr("tidbcloud_dedicated_cluster.test", "tikv_node_setting.node_count", "3"),
					resource.TestCheckResourceAttr("tidbcloud_dedicated_cluster.test", "tikv_node_setting.storage_size_gi", "10"),
					resource.TestCheckResourceAttr("tidbcloud_dedicated_cluster.test", "tikv_node_setting.storage_type", "BASIC"),
				),
			},
			// Update testing
			{
				Config: testAccDedicatedClusterResourceConfig_update(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("tidbcloud_dedicated_cluster.test", "tidb_node_setting.node_spec_key", "8C16G"),
					resource.TestCheckResourceAttr("tidbcloud_dedicated_cluster.test", "tidb_node_setting.node_count", "2"),
					resource.TestCheckResourceAttr("tidbcloud_dedicated_cluster.test", "tikv_node_setting.node_spec_key", "8C32G"),
				),
			},
			// Paused testing
			{
				Config: testAccDedicatedClusterResourceConfig_paused(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("tidbcloud_dedicated_cluster.test", "state", string(dedicated.COMMONV1BETA1CLUSTERSTATE_PAUSED)),
				),
			},
		},
	})
}

func TestUTDedicatedClusterResource(t *testing.T) {
	setupTestEnv()

	ctrl := gomock.NewController(t)
	s := mockClient.NewMockTiDBCloudDedicatedClient(ctrl)
	defer HookGlobal(&NewDedicatedClient, func(publicKey string, privateKey string, dedicatedEndpoint string, userAgent string) (tidbcloud.TiDBCloudDedicatedClient, error) {
		return s, nil
	})()

	clusterId := "cluster_id"
	displayName := "test-tf"
	nodeSpec := "2C4G"
	nodeSpecDisplayName := "2 vCPU, 4 GiB beta"
	updatedNodeSpec := "2C8G"
	updatedNodeSpecDisplayName := "2 vCPU, 8 GiB (Beta)"

	createClusterResp := dedicated.TidbCloudOpenApidedicatedv1beta1Cluster{}
	createClusterResp.UnmarshalJSON([]byte(testUTTidbCloudOpenApidedicatedv1beta1Cluster(clusterId, displayName, string(dedicated.COMMONV1BETA1CLUSTERSTATE_CREATING), nodeSpec, nodeSpecDisplayName)))
	getClusterResp := dedicated.TidbCloudOpenApidedicatedv1beta1Cluster{}
	getClusterResp.UnmarshalJSON([]byte(testUTTidbCloudOpenApidedicatedv1beta1Cluster(clusterId, displayName, string(dedicated.COMMONV1BETA1CLUSTERSTATE_ACTIVE), nodeSpec, nodeSpecDisplayName)))
	getClusterAfterUpdateResp := dedicated.TidbCloudOpenApidedicatedv1beta1Cluster{}
	getClusterAfterUpdateResp.UnmarshalJSON([]byte(testUTTidbCloudOpenApidedicatedv1beta1Cluster(clusterId, "test-tf2", string(dedicated.COMMONV1BETA1CLUSTERSTATE_ACTIVE), updatedNodeSpec, updatedNodeSpecDisplayName)))
	updateClusterSuccessResp := dedicated.TidbCloudOpenApidedicatedv1beta1Cluster{}
	updateClusterSuccessResp.UnmarshalJSON([]byte(testUTTidbCloudOpenApidedicatedv1beta1Cluster(clusterId, "test-tf2", string(dedicated.COMMONV1BETA1CLUSTERSTATE_MODIFYING), updatedNodeSpec, updatedNodeSpecDisplayName)))
	publicEndpointResp := dedicated.V1beta1PublicEndpointSetting{}
	publicEndpointResp.UnmarshalJSON([]byte(testUTV1beta1PublicEndpointSetting()))

	s.EXPECT().CreateCluster(gomock.Any(), gomock.Any()).Return(&createClusterResp, nil)
	gomock.InOrder(
		s.EXPECT().GetCluster(gomock.Any(), clusterId).Return(&getClusterResp, nil).Times(3),
		s.EXPECT().GetCluster(gomock.Any(), clusterId).Return(&getClusterAfterUpdateResp, nil).AnyTimes(),
	)
	s.EXPECT().UpdateCluster(gomock.Any(), gomock.Any(), gomock.Any()).Return(&updateClusterSuccessResp, nil)

	s.EXPECT().DeleteCluster(gomock.Any(), clusterId).Return(&getClusterResp, nil)

	s.EXPECT().GetPublicEndpoint(gomock.Any(), clusterId, gomock.Any()).Return(&publicEndpointResp, nil).AnyTimes()
	s.EXPECT().UpdatePublicEndpoint(gomock.Any(), clusterId, gomock.Any(), gomock.Any()).Return(&publicEndpointResp, nil).AnyTimes()

	testDedicatedClusterResource(t)
}

func testDedicatedClusterResource(t *testing.T) {
	dedicatedClusterResourceName := "tidbcloud_dedicated_cluster.test"
	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read serverless cluster resource
			{
				Config: testUTDedicatedClusterResourceConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(dedicatedClusterResourceName, "cluster_id"),
					resource.TestCheckResourceAttr(dedicatedClusterResourceName, "display_name", "test-tf"),
					resource.TestCheckResourceAttr(dedicatedClusterResourceName, "region_id", "aws-us-west-2"),
				),
			},
			// Update correctly
			{
				Config: testUTDedicatedClusterResourceUpdateConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dedicatedClusterResourceName, "display_name", "test-tf2"),
					resource.TestCheckResourceAttr(dedicatedClusterResourceName, "tidb_node_setting.node_spec_key", "2C8G"),
					resource.TestCheckResourceAttr(dedicatedClusterResourceName, "tidb_node_setting.node_spec_display_name", "2 vCPU, 8 GiB (Beta)"),
				),
			},
			// Update fields with paused
			{
				Config:      testUTDedicatedClusterResourcePausedWithUpdateFieldsConfig(),
				ExpectError: regexp.MustCompile(`.*Cannot change cluster pause state along with other attributes.*`),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func testUTDedicatedClusterResourcePausedWithUpdateFieldsConfig() string {
	return `
resource "tidbcloud_dedicated_cluster" "test" {
    display_name = "test-tf3"
    region_id = "aws-us-west-2"
    port = 4000
	  paused = true
    tidb_node_setting = {
      node_spec_key = "2C8G"
      node_count = 1
    }
    tikv_node_setting = {
      node_spec_key = "2C8G"
      node_count = 3
      storage_size_gi = 60
      storage_type = "Standard"
    }
}
`
}

func testUTDedicatedClusterResourceUpdateConfig() string {
	return `
resource "tidbcloud_dedicated_cluster" "test" {
    display_name = "test-tf2"
    region_id = "aws-us-west-2"
    port = 4000
    tidb_node_setting = {
      node_spec_key = "2C8G"
      node_count = 1
    }
    tikv_node_setting = {
      node_spec_key = "2C8G"
      node_count = 3
      storage_size_gi = 60
      storage_type = "Standard"
    }
}
`
}

func testUTDedicatedClusterResourceConfig() string {
	return `
resource "tidbcloud_dedicated_cluster" "test" {
    display_name = "test-tf"
    region_id = "aws-us-west-2"
    port = 4000
    tidb_node_setting = {
      node_spec_key = "2C4G"
      node_count = 1
    }
    tikv_node_setting = {
      node_spec_key = "2C4G"
      node_count = 3
      storage_size_gi = 60
      storage_type = "Standard"
    }
}
`
}

func testAccDedicatedClusterResourceConfig() string {
	return `
resource "tidbcloud_dedicated_cluster" "test" {
    display_name = "test-tf"
    region_id = "aws-us-west-2"
    port = 4000
    root_password = "123456789"
    tidb_node_setting = {
      node_spec_key = "2C4G"
      node_count = 1
    }
    tikv_node_setting = {
      node_spec_key = "2C4G"
      node_count = 3
      storage_size_gi = 10
      storage_type = "BASIC"
    }
}
`
}

func testAccDedicatedClusterResourceConfig_update() string {
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
`
}

func testAccDedicatedClusterResourceConfig_paused() string {
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
	paused = true
}
`
}

func testUTTidbCloudOpenApidedicatedv1beta1Cluster(clusterId, displayName, state, nodeSpec, nodeSpecDisplayName string) string {
	return fmt.Sprintf(`{
    "name": "clusters/%s",
    "clusterId": "%s",
    "displayName": "%s",
    "regionId": "aws-us-west-2",
    "labels": {
        "tidb.cloud/organization": "00000",
        "tidb.cloud/project": "00000000"
    },
    "tidbNodeSetting": {
        "nodeSpecKey": "%s",
        "tidbNodeGroups": [
            {
                "name": "tidbNodeGroups/0",
                "tidbNodeGroupId": "0",
                "clusterId": "%s",
                "displayName": "",
                "nodeCount": 1,
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
                "nodeSpecKey": "%s",
                "nodeSpecDisplayName": "%s",
                "isDefaultGroup": true,
                "state": "MODIFYING"
            }
        ],
        "nodeSpecDisplayName": "%s"
    },
    "tikvNodeSetting": {
        "nodeCount": 3,
        "nodeSpecKey": "%s",
        "storageSizeGi": 60,
        "storageType": "Standard",
        "nodeSpecDisplayName": "%s"
    },
    "port": 4000,
    "rootPassword": "",
    "state": "%s",
    "version": "v8.1.2",
    "createdBy": "apikey-xxxxxxx",
    "createTime": "2025-04-17T09:31:12.647Z",
    "updateTime": "2025-04-17T09:31:12.647Z",
    "regionDisplayName": "Oregon (us-west-2)",
    "cloudProvider": "aws",
    "annotations": {
        "tidb.cloud/available-features": "",
        "tidb.cloud/has-set-password": "false"
    }
}`, clusterId, clusterId, displayName, nodeSpec, clusterId,
		nodeSpec, nodeSpecDisplayName, nodeSpecDisplayName, nodeSpec, nodeSpecDisplayName, state)
}
