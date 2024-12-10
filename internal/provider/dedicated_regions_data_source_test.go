package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccDedicatedRegionsDataSource(t *testing.T) {
	t.Parallel()

	testDedicatedRegionsDataSource(t)
}

func testDedicatedRegionsDataSource(t *testing.T) {
	dedicatedRegionsDataSourceName := "data.tidbcloud_dedicated_regions.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,

		Steps: []resource.TestStep{
			{
				Config: testDedicatedRegionsConfig,
				Check: resource.ComposeTestCheckFunc(
					func(s *terraform.State) error {
						_, ok := s.RootModule().Resources[dedicatedRegionsDataSourceName]
						if !ok {
							return fmt.Errorf("Not found: %s", dedicatedRegionsDataSourceName)
						}
						return nil
					},
				),
			},
		},
	})
}

const testDedicatedRegionsConfig = `
data "tidbcloud_dedicated_regions" "test" {}
`
