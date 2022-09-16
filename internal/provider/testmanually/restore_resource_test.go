package manually

import (
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/tidbcloud/terraform-provider-tidbcloud/internal/provider"
	"regexp"
	"testing"
)

// Test restore resource, if you want to test it:
// 1. delete the t.Skip
// 2. make sure a backup is set up already, set project_id and backup_id in testAccRestoreResourceConfig
// 3. test with `TF_ACC=1 go test -v ./internal/provider/testmanually/restore_resource_test.go`
// 4. ignore the delete error, because restore task can not be deleted. Check the console. if the cluster is in creating, the test is regarded as success
// 5. delete the restored cluster manually
func TestAccRestoreResource(t *testing.T) {
	t.Skip("skip for restored can't be delete")
	reg, _ := regexp.Compile(".*restore can't be updated.*")
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: map[string]func() (tfprotov6.ProviderServer, error){
			"tidbcloud": providerserver.NewProtocol6WithError(provider.New("test")()),
		},
		Steps: []resource.TestStep{
			// Create and Read restore
			{
				Config: testAccRestoreResourceConfig("restore-test"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("tidbcloud_restore.test", "id"),
					resource.TestCheckResourceAttr("tidbcloud_restore.test", "error_message", ""),
				),
			},
			// Update is not supported
			{
				Config:      testAccRestoreResourceConfig("restore-test2"),
				ExpectError: reg,
			},
		},
	})
}

// full in the project_id and backup_id
func testAccRestoreResourceConfig(name string) string {
	return fmt.Sprintf(`
resource "tidbcloud_restore" "test" {
  project_id     = "1372813089189561287"
  backup_id      = "1320143"
  name           = "%s"
  config = {
      root_password = "Shiyuhang1."
      port = 4002
      components = {
            tidb = {
              node_size : "8C16G"
              node_quantity : 1
            }
            tikv = {
              node_size : "8C32G"
              storage_size_gib : 500
              node_quantity : 3
            }
            tiflash = {
               node_size : "8C64G"
               storage_size_gib : 500
               node_quantity : 1
            }
          }
      ip_access_list = [{
          cidr        = "0.0.0.0/0"
          description = "all"
          }
        ]
    }
}`, name)
}
