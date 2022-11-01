package testwithcluster

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccBackupDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Read testing
			{
				Config: testAccBackupDataSourceConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.tidbcloud_backups.test", "total"),
				),
			},
		},
	})
}

var testAccBackupDataSourceConfig = fmt.Sprintf(`
data "tidbcloud_backups" "test" {
  project_id = %s
  cluster_id = %s
}
`, projectId, clusterId)
