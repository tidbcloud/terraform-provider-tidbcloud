package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccServerlessClusterResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccServerlessClusterResourceConfig(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("tidbcloud_serverless_cluster.test", "name", "test-tf"),
					resource.TestCheckResourceAttr("tidbcloud_serverless_cluster.test", "region.region_id", "us-east-1"),
					resource.TestCheckResourceAttr("tidbcloud_serverless_cluster.test", "endpoints.public.port", "4000"),
					resource.TestCheckResourceAttr("tidbcloud_serverless_cluster.test", "endpoints.private.port", "4000"),
					resource.TestCheckResourceAttr("tidbcloud_serverless_cluster.test", "high_availability_type.", "ZONAL"),
				),
			},
			// Update testing
			{
				Config: testAccServerlessClusterResourceConfig_update(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("tidbcloud_serverless_cluster.test", "name", "test-tf2"),
				),
			},
		},
	})
}

func testAccServerlessClusterResourceConfig() string {
	return `
resource "tidbcloud_serverless_cluster" "example" {
   display_name = "test-tf"
   region = {
      name = "regions/aws-us-east-1"
   }
   high_availability_type = "ZONAL"
}
`
}

func testAccServerlessClusterResourceConfig_update() string {
	return `
resource "tidbcloud_serverless_cluster" "example" {
   display_name = "test-tf2"
   region = {
      name = "regions/aws-us-east-1"
   }
   high_availability_type = "ZONAL"
   endpoints = {
      public_endpoint = {
         disabled = true
      }
   }
}
`
}