package provider

import (
	"fmt"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	mockClient "github.com/tidbcloud/terraform-provider-tidbcloud/mock"
	"github.com/tidbcloud/terraform-provider-tidbcloud/tidbcloud"
	clusterV1beta1 "github.com/tidbcloud/tidbcloud-cli/pkg/tidbcloud/v1beta1/serverless/cluster"
)

func TestAccServerlessClusterDataSource(t *testing.T) {
	serverlessClusterDataSourceName := "data.tidbcloud_serverless_cluster.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testServerlessClusterDataSourceConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(serverlessClusterDataSourceName, "display_name", "test-tf"),
					resource.TestCheckResourceAttr(serverlessClusterDataSourceName, "region.name", "regions/aws-us-east-1"),
					resource.TestCheckResourceAttr(serverlessClusterDataSourceName, "endpoints.public_endpoint.port", "4000"),
				),
			},
		},
	})
}

func TestUTServerlessClusterDataSource(t *testing.T) {
	setupTestEnv()

	ctrl := gomock.NewController(t)
	s := mockClient.NewMockTiDBCloudServerlessClient(ctrl)
	defer HookGlobal(&NewServerlessClient, func(publicKey string, privateKey string, serverlessEndpoint string, userAgent string) (tidbcloud.TiDBCloudServerlessClient, error) {
		return s, nil
	})()

	clusterId := "cluster_id"
	regionName := "regions/aws-us-east-1"
	displayName := "test-tf"

	getClusterResp := clusterV1beta1.TidbCloudOpenApiserverlessv1beta1Cluster{}
	getClusterResp.UnmarshalJSON([]byte(testUTTidbCloudOpenApiserverlessv1beta1Cluster(clusterId, regionName, displayName, string(clusterV1beta1.COMMONV1BETA1CLUSTERSTATE_ACTIVE))))

	s.EXPECT().GetCluster(gomock.Any(), clusterId, clusterV1beta1.SERVERLESSSERVICEGETCLUSTERVIEWPARAMETER_FULL).Return(&getClusterResp, nil).AnyTimes()

	testUTServerlessClusterDataSource(t, clusterId)
}

func testUTServerlessClusterDataSource(t *testing.T, clusterId string) {
	serverlessClusterDataSourceName := "data.tidbcloud_serverless_cluster.test"
	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testUTServerlessClusterDataSourceConfig(clusterId),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(serverlessClusterDataSourceName, "display_name", "test-tf"),
					resource.TestCheckResourceAttr(serverlessClusterDataSourceName, "region.name", "regions/aws-us-east-1"),
					resource.TestCheckResourceAttr(serverlessClusterDataSourceName, "endpoints.public_endpoint.port", "4000"),
				),
			},
		},
	})
}

const testServerlessClusterDataSourceConfig = `
resource "tidbcloud_serverless_cluster" "example" {
   display_name = "test-tf"
   region = {
      name = "regions/aws-us-east-1"
   }
}

data "tidbcloud_serverless_cluster" "test" {
	cluster_id = tidbcloud_serverless_cluster.example.cluster_id
}
`

func testUTServerlessClusterDataSourceConfig(cluster_id string) string {
	return fmt.Sprintf(`
data "tidbcloud_serverless_cluster" "test" {
	cluster_id = "%s" 
}
`, cluster_id)
}
