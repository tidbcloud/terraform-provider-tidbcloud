package provider

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	mockClient "github.com/tidbcloud/terraform-provider-tidbcloud/mock"
	"github.com/tidbcloud/terraform-provider-tidbcloud/tidbcloud"
	"github.com/tidbcloud/tidbcloud-cli/pkg/tidbcloud/v1beta1/dedicated"
)

func TestAccDedicatedClusterDataSource(t *testing.T) {
	dedicatedClusterDataSourceName := "data.tidbcloud_dedicated_cluster.test"
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testDedicatedClusterDataSourceConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dedicatedClusterDataSourceName, "display_name", "test-tf"),
				),
			},
		},
	})
}

func TestUTDedicatedClusterDataSource(t *testing.T) {
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

	getClusterResp := dedicated.TidbCloudOpenApidedicatedv1beta1Cluster{}
	getClusterResp.UnmarshalJSON([]byte(testUTTidbCloudOpenApidedicatedv1beta1Cluster(clusterId, displayName, string(dedicated.COMMONV1BETA1CLUSTERSTATE_ACTIVE), nodeSpec, nodeSpecDisplayName)))

	s.EXPECT().GetCluster(gomock.Any(), clusterId).Return(&getClusterResp, nil).AnyTimes()

	testUTDedicatedClusterDataSource(t)
}

func testUTDedicatedClusterDataSource(t *testing.T) {
	dedicatedClusterDataSourceName := "data.tidbcloud_dedicated_cluster.test"
	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testUTDedicatedClusterDataSourceConfig(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(dedicatedClusterDataSourceName, "display_name", "test-tf"),
				),
			},
		},
	})
}

const testDedicatedClusterDataSourceConfig = `
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


data "tidbcloud_dedicated_cluster" "test" {
  cluster_id = tidbcloud_dedicated_cluster.test.id
}
`

func testUTDedicatedClusterDataSourceConfig() string {
	return `
data "tidbcloud_dedicated_cluster" "test" {
	cluster_id = "cluster_id"
}
`
}
