package provider

import (
	"fmt"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	mockClient "github.com/tidbcloud/terraform-provider-tidbcloud/mock"
	"github.com/tidbcloud/terraform-provider-tidbcloud/tidbcloud"
	"github.com/tidbcloud/tidbcloud-cli/pkg/tidbcloud/v1beta1/dedicated"
)

func TestUTDedicatedAuditLogConfigDataSource(t *testing.T) {
	setupTestEnv()

	ctrl := gomock.NewController(t)
	s := mockClient.NewMockTiDBCloudDedicatedClient(ctrl)
	defer HookGlobal(&NewDedicatedClient, func(publicKey string, privateKey string, dedicatedEndpoint string, userAgent string) (tidbcloud.TiDBCloudDedicatedClient, error) {
		return s, nil
	})()

	getResp := dedicated.Dedicatedv1beta1AuditLogConfig{}
	getResp.UnmarshalJSON([]byte(testUTDedicatedv1beta1AuditLogConfig(true)))

	s.EXPECT().GetAuditLogConfig(gomock.Any(), gomock.Any()).Return(&getResp, nil).AnyTimes()

	dedicatedAuditLogConfigDataSourceName := "data.tidbcloud_dedicated_audit_log_config.test"
	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testUTDedicatedAuditLogConfigDataSourceConfig(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(dedicatedAuditLogConfigDataSourceName, "enabled", "true"),
					resource.TestCheckResourceAttr(dedicatedAuditLogConfigDataSourceName, "bucket_region_id", "aws-us-west-2"),
					resource.TestCheckResourceAttr(dedicatedAuditLogConfigDataSourceName, "bucket_write_check.writable", "true"),
				),
			},
		},
	})
}

func testUTDedicatedAuditLogConfigDataSourceConfig() string {
	return `
data "tidbcloud_dedicated_audit_log_config" "test" {
    cluster_id = "cluster-id"
}
`
}

func testUTDedicatedv1beta1AuditLogConfig(enabled bool) string {
	return fmt.Sprintf(`
{
	"clusterId": "cluster-id",
	"enabled": %t,
	"bucketUri": "s3://xxxxxxxx/xxx",
	"bucketRegionId": "aws-us-west-2",
	"awsRoleArn": "arn:aws:iam::00000000:role/xxxxx",
	"azureSasToken": "",
	"bucketWriteCheck": {
		"writable": true,
		"errorReason": ""
	},
	"bucketManager": "CUSTOMER"
}
`, enabled)
}
