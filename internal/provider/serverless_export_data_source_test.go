package provider

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	mockClient "github.com/tidbcloud/terraform-provider-tidbcloud/mock"
	"github.com/tidbcloud/terraform-provider-tidbcloud/tidbcloud"
	exportV1beta1 "github.com/tidbcloud/tidbcloud-cli/pkg/tidbcloud/v1beta1/serverless/export"
)

func TestAccServerlessExportDataSource(t *testing.T) {
	serverlessExportDataSourceName := "data.tidbcloud_serverless_export.test"
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testServerlessExportDataSourceConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(serverlessExportDataSourceName, "display_name", "test-tf"),
				),
			},
		},
	})
}

func TestUTServerlessExportDataSource(t *testing.T) {
	setupTestEnv()

	ctrl := gomock.NewController(t)
	s := mockClient.NewMockTiDBCloudServerlessClient(ctrl)
	defer HookGlobal(&NewServerlessClient, func(publicKey string, privateKey string, serverlessEndpoint string, userAgent string) (tidbcloud.TiDBCloudServerlessClient, error) {
		return s, nil
	})()

	exportId := "export-id"

	getExportResp := exportV1beta1.Export{}
	getExportResp.UnmarshalJSON([]byte(testUTExport(string(exportV1beta1.EXPORTSTATEENUM_SUCCEEDED))))

	s.EXPECT().GetExport(gomock.Any(), gomock.Any(), exportId).Return(&getExportResp, nil).AnyTimes()

	testUTServerlessExportDataSource(t)
}

func testUTServerlessExportDataSource(t *testing.T) {
	serverlessExportDataSourceName := "data.tidbcloud_serverless_export.test"
	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testUTServerlessExportDataSourceConfig(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(serverlessExportDataSourceName, "target.type", "LOCAL"),
				),
			},
		},
	})
}

const testServerlessExportDataSourceConfig = `
resource "tidbcloud_serverless_cluster" "example" {
   display_name = "test-tf"
   region = {
      name = "regions/aws-us-east-1"
   }
}
resource "tidbcloud_serverless_export" "test" {
	cluster_id = tidbcloud_serverless_cluster.example.cluster_id
	display_name = "test-tf"
}
data "tidbcloud_serverless_export" "test" {
	cluster_id = tidbcloud_serverless_cluster.example.cluster_id
	export_id = tidbcloud_serverless_export.test.export_id
}
`

func testUTServerlessExportDataSourceConfig() string {
	return `
data "tidbcloud_serverless_export" "test" {
	cluster_id = "cluster-id"
	export_id = "export-id" 
}
`
}