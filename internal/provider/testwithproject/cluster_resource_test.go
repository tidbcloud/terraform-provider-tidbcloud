package testwithproject

import (
	"crypto/rand"
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"regexp"
	"testing"
)

// create dedicated cluster may cause cost, make sure you have enough balance
// update node_quantity is not tested for create dedicated tier needs too much time!
func TestAccClusterResource(t *testing.T) {
	if !enableCost {
		t.Skip("Skip test due to cost")
	}
	// Reuse one password across create and update steps.
	password := "Aa1!" + rand.Text()
	reg, _ := regexp.Compile(".*Unable to update Serverless cluster.*")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read serverless tier
			{
				Config: testAccServerlessClusterResourceConfig("serverless-test", password),
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
				Config:      testAccServerlessClusterResourceConfig("serverless-test2", password),
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
				Config: testAccDedicatedClusterResourceConfig("dedicated-test", password),
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
				Config:      testAccDedicatedClusterResourceConfig("dedicated-test2", password),
				ExpectError: reg2,
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func testAccServerlessClusterResourceConfig(name, password string) string {
	return fmt.Sprintf(`
resource "tidbcloud_cluster" "test" {
 project_id     = %s
 name           = "%s"
 cluster_type   = "DEVELOPER"
 cloud_provider = "AWS"
 region         = "us-east-1"
 config = {
   root_password = %q
 }
}
`, projectId, name, password)
}

func testAccDedicatedClusterResourceConfig(name, password string) string {
	return fmt.Sprintf(`
resource "tidbcloud_cluster" "test" {
 project_id     = %s
 name           = "%s"
 cluster_type   = "DEDICATED"
 cloud_provider = "AWS"
 region         = "us-east-1"
 config = {
   root_password = %q
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
`, projectId, name, password)
}
