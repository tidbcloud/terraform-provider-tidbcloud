package testwithcluster

import (
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"regexp"
	"testing"
	"time"
)

// make sure a dedicated cluster is set up already
func TestAccBackupResource(t *testing.T) {
	reg, _ := regexp.Compile(".*backup can't be updated.*")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read backup task
			{
				Config: testAccBackupResourceConfig("backup-test"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("tidbcloud_backup.test", "id"),
					resource.TestCheckResourceAttrSet("tidbcloud_backup.test", "create_timestamp"),
					resource.TestCheckResourceAttrSet("tidbcloud_backup.test", "type"),
					resource.TestCheckResourceAttrSet("tidbcloud_backup.test", "size"),
					resource.TestCheckResourceAttrSet("tidbcloud_backup.test", "status"),
				),
			},
			// Test import
			{
				ResourceName:        "tidbcloud_backup.test",
				ImportState:         true,
				ImportStateIdPrefix: fmt.Sprintf("%s,%s,", projectId, clusterId),
			},
			// Update is not supported
			{
				Config:                    testAccBackupResourceConfig("backup-test2"),
				ExpectError:               reg,
				PreventPostDestroyRefresh: true,
			},
			// just sleep 100s to wait backup ready, so that we can delete it
			{
				Config: testAccBackupResourceConfig("backup-test"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccBackupSleep(),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func testAccBackupResourceConfig(name string) string {
	return fmt.Sprintf(`
resource "tidbcloud_backup" "test" {
  project_id     = %s
  cluster_id     = %s
  name           = "%s"
}`, projectId, clusterId, name)
}

func testAccBackupSleep() resource.TestCheckFunc {
	return func(s *terraform.State) error {
		time.Sleep(time.Duration(100) * time.Second)
		return nil
	}
}
