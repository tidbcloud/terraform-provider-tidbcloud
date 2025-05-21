package provider

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	mockClient "github.com/tidbcloud/terraform-provider-tidbcloud/mock"
	"github.com/tidbcloud/terraform-provider-tidbcloud/tidbcloud"
	"github.com/tidbcloud/tidbcloud-cli/pkg/tidbcloud/v1beta1/dedicated"
)

func TestAccDedicatedNodeGroupDataSource(t *testing.T) {
	dedicatedNodeGroupDataSourceName := "data.tidbcloud_dedicated_node_group.test"
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testDedicatedNodeGroupDataSourceConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dedicatedNodeGroupDataSourceName, "display_name", "test_group"),
				),
			},
		},
	})
}

func TestUTDedicatedNodeGroupDataSource(t *testing.T) {
	setupTestEnv()

	ctrl := gomock.NewController(t)
	s := mockClient.NewMockTiDBCloudDedicatedClient(ctrl)
	defer HookGlobal(&NewDedicatedClient, func(publicKey string, privateKey string, dedicatedEndpoint string, userAgent string) (tidbcloud.TiDBCloudDedicatedClient, error) {
		return s, nil
	})()

	nodeGroupId := "node_group_id"

	getNodeGroupResp := dedicated.Dedicatedv1beta1TidbNodeGroup{}
	getNodeGroupResp.UnmarshalJSON([]byte(testUTTidbCloudOpenApidedicatedv1beta1NodeGroup("cluster_id", "test_group", string(dedicated.DEDICATEDV1BETA1TIDBNODEGROUPSTATE_ACTIVE), 1)))
	publicEndpointResp := dedicated.V1beta1PublicEndpointSetting{}
	publicEndpointResp.UnmarshalJSON([]byte(testUTV1beta1PublicEndpointSetting()))

	s.EXPECT().GetTiDBNodeGroup(gomock.Any(), gomock.Any(), nodeGroupId).Return(&getNodeGroupResp, nil).AnyTimes()
	s.EXPECT().GetPublicEndpoint(gomock.Any(), gomock.Any(), gomock.Any()).Return(&publicEndpointResp, nil).AnyTimes()

	testUTDedicatedNodeGroupDataSource(t)
}

func testUTDedicatedNodeGroupDataSource(t *testing.T) {
	dedicatedNodeGroupDataSourceName := "data.tidbcloud_dedicated_node_group.test"
	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testUTDedicatedNodeGroupDataSourceConfig(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(dedicatedNodeGroupDataSourceName, "display_name", "test_group"),
				),
			},
		},
	})
}

const testDedicatedNodeGroupDataSourceConfig = `
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
    display_name = "test_group"
}

data "tidbcloud_dedicated_node_group" "test" {
  cluster_id = tidbcloud_dedicated_cluster.test.id
  node_group_id = tidbcloud_dedicated_node_group.test_group.id
}
`

func testUTDedicatedNodeGroupDataSourceConfig() string {
	return `
data "tidbcloud_dedicated_node_group" "test" {
	cluster_id = "cluster_id"
	node_group_id = "node_group_id" 
}
`
}
