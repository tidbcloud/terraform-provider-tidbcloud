package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccDedicatedNodeGroupsDataSource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDedicatedNodeGroupsDataSourceConfig(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.tidbcloud_dedicated_node_groups.test", "node_groups.#", "1"),
					resource.TestCheckResourceAttr("data.tidbcloud_dedicated_node_groups.test", "node_groups.0.node_count", "1"),
					resource.TestCheckResourceAttr("data.tidbcloud_dedicated_node_groups.test", "node_groups.0.display_name", "test-node-group"),
				),
			},
		},
	})

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
