package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
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
