package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccDedicatedRegionDataSource(t *testing.T) {
	t.Parallel()

	testDedicatedRegionDataSource(t)
}

func testDedicatedRegionDataSource(t *testing.T) {
	dedicatedRegionDataSourceName := "data.tidbcloud_dedicated_region.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,

		Steps: []resource.TestStep{
			{
				Config: testDedicatedRegionConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dedicatedRegionDataSourceName, "cloud_provider", "aws"),
					resource.TestCheckResourceAttrSet(dedicatedRegionDataSourceName, "display_name"),
					resource.TestCheckResourceAttrSet(dedicatedRegionDataSourceName, "region_id"),
				),
			},
		},
	})
}

const testDedicatedRegionConfig = `
data "tidbcloud_dedicated_region" "test" {
	region_id = "aws-us-east-1"
}
`
