package testwithproject

import (
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"regexp"
	"testing"
)

// create dedicated cluster may cause cost, make sure you have enough balance
// update node_quantity is not tested for create dedicated tier needs too much time!
func TestAccClusterResource(t *testing.T) {
	reg, _ := regexp.Compile(".*Unable to update DEVELOPER cluster.*")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read dev-tier
			{
				Config: testAccDevClusterResourceConfig("developer-test"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("tidbcloud_cluster.test", "id"),
					resource.TestCheckResourceAttrSet("tidbcloud_cluster.test", "config.components.tidb.node_quantity"),
					resource.TestCheckResourceAttrSet("tidbcloud_cluster.test", "config.port"),
				),
			},
			// Test import
			{
				ResourceName:        "tidbcloud_cluster.test",
				ImportState:         true,
				ImportStateIdPrefix: fmt.Sprintf("%s,", projectId),
			},
			// Update is not supported
			{
				Config:      testAccDevClusterResourceConfig("developer-test2"),
				ExpectError: reg,
			},
			// Delete testing automatically occurs in TestCase
		},
	})
	reg2, _ := regexp.Compile(".*only components can be changed now.*")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read dedicated tier
			{
				Config: testAccDedicatedClusterResourceConfig("dedicated-test"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("tidbcloud_cluster.test", "id"),
					resource.TestCheckResourceAttrSet("tidbcloud_cluster.test", "config.port"),
				),
			},
			// Test import
			{
				ResourceName:        "tidbcloud_cluster.test",
				ImportState:         true,
				ImportStateIdPrefix: fmt.Sprintf("%s,", projectId),
			},
			// only node_quantity can be updated now
			{
				Config:      testAccDedicatedClusterResourceConfig("dedicated-test2"),
				ExpectError: reg2,
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func testAccDevClusterResourceConfig(name string) string {
	return fmt.Sprintf(`
resource "tidbcloud_cluster" "test" {
  project_id     = %s
  name           = "%s"
  cluster_type   = "DEVELOPER"
  cloud_provider = "AWS"
  region         = "us-east-1"
  config = {
    root_password = "Shiyuhang1."
    ip_access_list = [{
        cidr        = "0.0.0.0/0"
        description = "all"
        }
      ]
  }
}
`, projectId, name)
}

func testAccDedicatedClusterResourceConfig(name string) string {
	return fmt.Sprintf(`
resource "tidbcloud_cluster" "test" {
  project_id     = %s
  name           = "%s"
  cluster_type   = "DEDICATED"
  cloud_provider = "AWS"
  region         = "us-east-1"
  config = {
    root_password = "Shiyuhang1."
    components = {
      tidb = {
        node_size : "2C8G"
        node_quantity : 1
      }
      tikv = {
        node_size : "2C8G"
        storage_size_gib : 500,
        node_quantity : 3
      }
    }
  }
}
`, projectId, name)
}
