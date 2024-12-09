package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccDedicatedCloudProvidersDataSource(t *testing.T) {
	t.Parallel()

	testDedicatedCloudProvidersDataSource(t)
}

func testDedicatedCloudProvidersDataSource(t *testing.T) {
	dedicatedCloudProvidersDataSourceName := "data.tidbcloud_dedicated_cloud_providers.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,

		Steps: []resource.TestStep{
			{
				Config: testDedicatedCloudProvidersConfig,
				Check: resource.ComposeTestCheckFunc(
					// 这里可以添加多个检查条件
					func(s *terraform.State) error {
						_, ok := s.RootModule().Resources[dedicatedCloudProvidersDataSourceName]
						if !ok {
							return fmt.Errorf("Not found: %s", dedicatedCloudProvidersDataSourceName)
						}
						return nil
					},
				),
			},
		},
	})
}

const testDedicatedCloudProvidersConfig = `
data "tidbcloud_dedicated_cloud_providers" "test" {}
`
