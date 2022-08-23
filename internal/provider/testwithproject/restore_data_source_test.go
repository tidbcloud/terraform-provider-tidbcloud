package testwithproject

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccRestoreDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Read testing
			{
				Config: testAccRestoreDataSourceConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.tidbcloud_restore.test", "total"),
				),
			},
		},
	})
}

var testAccRestoreDataSourceConfig = fmt.Sprintf(`
data "tidbcloud_restore" "test" {
  project_id = %s
}
`, projectId)
