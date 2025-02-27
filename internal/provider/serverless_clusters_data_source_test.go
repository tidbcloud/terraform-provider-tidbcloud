package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccServerlessClustersDataSource(t *testing.T) {
	t.Parallel()

	testServerlessClustersDataSource(t)
}

func testServerlessClustersDataSource(t *testing.T) {
	serverlessClustersDataSourceName := "data.tidbcloud_serverless_clusters.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,

		Steps: []resource.TestStep{
			{
				Config: testServerlessClustersConfig,
				Check: resource.ComposeTestCheckFunc(
					func(s *terraform.State) error {
						_, ok := s.RootModule().Resources[serverlessClustersDataSourceName]
						if !ok {
							return fmt.Errorf("Not found: %s", serverlessClustersDataSourceName)
						}
						return nil
					},
				),
			},
		},
	})
}

const testServerlessClustersConfig = `
data "tidbcloud_serverless_clusters" "test" {}
`
