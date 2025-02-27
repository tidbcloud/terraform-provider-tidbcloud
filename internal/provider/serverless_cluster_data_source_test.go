package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccServerlessClusterDataSource(t *testing.T) {
	t.Parallel()

	testServerlessClusterDataSource(t)
}

func testServerlessClusterDataSource(t *testing.T) {
	serverlessClusterDataSourceName := "data.tidbcloud_serverless_cluster.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,

		Steps: []resource.TestStep{
			{
				Config: testServerlessClusterConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(serverlessClusterDataSourceName, "display_name", "test-tf"),
					resource.TestCheckResourceAttr(serverlessClusterDataSourceName, "region.name", "regions/aws-us-east-1"),
					resource.TestCheckResourceAttr(serverlessClusterDataSourceName, "endpoints.public.port", "4000"),
				),
			},
		},
	})
}

const testServerlessClusterConfig = `
resource "tidbcloud_serverless_cluster" "example" {
   display_name = "test-tf"
   region = {
      name = "regions/aws-us-east-1"
   }
   high_availability_type = "ZONAL"
}

data "tidbcloud_serverless_cluster" "test" {
	cluster_id = tidbcloud_serverless_cluster.example.cluster_id
}
`
