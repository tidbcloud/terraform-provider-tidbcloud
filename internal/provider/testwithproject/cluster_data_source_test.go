package testwithproject

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccClusterDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Read testing
			{
				Config: testAccClusterDataSourceConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.tidbcloud_cluster.test", "total"),
				),
			},
		},
	})
}

var testAccClusterDataSourceConfig = fmt.Sprintf(`
data "tidbcloud_cluster" "test" {
  project_id = %s
}
`, projectId)
