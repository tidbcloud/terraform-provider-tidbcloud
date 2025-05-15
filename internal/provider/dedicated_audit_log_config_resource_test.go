package provider

// import (
// 	"fmt"
// 	"testing"

// 	"github.com/golang/mock/gomock"
// 	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
// 	mockClient "github.com/tidbcloud/terraform-provider-tidbcloud/mock"
// 	"github.com/tidbcloud/terraform-provider-tidbcloud/tidbcloud"
// 	"github.com/tidbcloud/tidbcloud-cli/pkg/tidbcloud/v1beta1/dedicated"
// )

// func TestUTDedicatedAuditLogConfigResource(t *testing.T) {
// 	setupTestEnv()

// 	ctrl := gomock.NewController(t)
// 	s := mockClient.NewMockTiDBCloudDedicatedClient(ctrl)
// 	defer HookGlobal(&NewDedicatedClient, func(publicKey string, privateKey string, dedicatedEndpoint string, userAgent string) (tidbcloud.TiDBCloudDedicatedClient, error) {
// 		return s, nil
// 	})()

// 	createAuditLogConfigResp := &dedicated.Dedicatedv1beta1AuditLogConfig{}
// 	createAuditLogConfigResp.UnmarshalJSON([]byte(testUTDedicatedv1beta1AuditLogConfig(true)))
// 	getAuditLogConfigResp := &dedicated.Dedicatedv1beta1AuditLogConfig{}
// 	getAuditLogConfigResp.UnmarshalJSON([]byte(testUTDedicatedv1beta1AuditLogConfig(true)))
// 	updateAuditLogConfigResp := &dedicated.Dedicatedv1beta1AuditLogConfig{}
// 	updateAuditLogConfigResp.UnmarshalJSON([]byte(testUTDedicatedv1beta1AuditLogConfig(false)))
// 	getAuditLogConfigAfterUpdateResp := &dedicated.Dedicatedv1beta1AuditLogConfig{}
// 	getAuditLogConfigAfterUpdateResp.UnmarshalJSON([]byte(testUTDedicatedv1beta1AuditLogConfig(false)))

// 	s.EXPECT().CreateAuditLogConfig(gomock.Any(), gomock.Any(), gomock.Any()).Return(createAuditLogConfigResp, nil)
// 	s.EXPECT().GetAuditLogConfig(gomock.Any(), gomock.Any()).Return(getAuditLogConfigResp, nil).Times(2)
// 	s.EXPECT().UpdateAuditLogConfig(gomock.Any(), gomock.Any(), gomock.Any()).Return(updateAuditLogConfigResp, nil)
// 	s.EXPECT().GetAuditLogConfig(gomock.Any(), gomock.Any()).Return(getAuditLogConfigAfterUpdateResp, nil).Times(2)

// 	testDedicatedAuditLogConfigResource(t)
// }

// func testDedicatedAuditLogConfigResource(t *testing.T) {
// 	resourceName := "tidbcloud_dedicated_audit_log_config.test"
// 	resource.Test(t, resource.TestCase{
// 		IsUnitTest:               true,
// 		PreCheck:                 func() { testAccPreCheck(t) },
// 		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
// 		Steps: []resource.TestStep{
// 			// Create and Read testing
// 			{
// 				Config: testUTDedicatedAuditLogConfigResourceConfig(),
// 				Check: resource.ComposeAggregateTestCheckFunc(
// 					resource.TestCheckResourceAttr(resourceName, "enabled", "true"),
// 					resource.TestCheckResourceAttr(resourceName, "bucket_region_id", "aws-us-west-2"),
// 				),
// 			},
// 			// Update testing
// 			{
// 				Config: testUTDedicatedAuditLogConfigResourceUpdateConfig(),
// 				Check: resource.ComposeAggregateTestCheckFunc(
// 					resource.TestCheckResourceAttr(resourceName, "enabled", "false"),
// 				),
// 			},
// 		},
// 	})
// }

// func testUTDedicatedAuditLogConfigResourceConfig() string {
// 	return `
// resource "tidbcloud_dedicated_audit_log_config" "test" {
//     cluster_id = "10750412092269866027"
//     enabled = true
//     bucket_uri = "s3://pingcap-test-tf/test-tf"
//     bucket_region_id = "aws-us-west-2"
//     aws_role_arn = "arn:aws:iam::986330900858:role/test-tf"
// }
// `
// }

// func testUTDedicatedAuditLogConfigResourceUpdateConfig() string {
// 	return `
// resource "tidbcloud_dedicated_audit_log_config" "test" {
//     cluster_id = "10750412092269866027"
//     enabled = false
//     bucket_uri = "s3://pingcap-test-tf/test-tf-updated"
//     bucket_region_id = "aws-us-west-2"
//     aws_role_arn = "arn:aws:iam::986330900858:role/test-tf"
// }
// `
// }

// func testUTDedicatedv1beta1AuditLogConfig(enabled bool) string {
// 	return fmt.Sprintf(`
// {
// 	"clusterId": "cluster-id",
// 	"enabled": %t,
// 	"bucketUri": "s3://pingcap-test-tf/test-tf",
// 	"bucketRegionId": "aws-us-west-2",
// 	"awsRoleArn": "arn:aws:iam::98600000:role/test-tf",
// 	"azureSasToken": "",
// 	"bucketWriteCheck": {
// 		"writable": true,
// 		"errorReason": ""
// 	},
// 	"bucketManager": "CUSTOMER"
// }
// `, enabled)
// }
