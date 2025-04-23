package provider

import (
	"fmt"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	mockClient "github.com/tidbcloud/terraform-provider-tidbcloud/mock"
	"github.com/tidbcloud/terraform-provider-tidbcloud/tidbcloud"
	exportV1beta1 "github.com/tidbcloud/tidbcloud-cli/pkg/tidbcloud/v1beta1/serverless/export"
)

func TestAccServerlessExportResource(t *testing.T) {
	serverlessExportResourceName := "tidbcloud_serverless_export.test"
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccServerlessExportResourceConfig(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(serverlessExportResourceName, "display_name", "test-tf"),
					resource.TestCheckResourceAttr(serverlessExportResourceName, "target.type", "LOCAL"),
				),
			},
		},
	})
}

func TestUTServerlessExportResource(t *testing.T) {
	setupTestEnv()

	ctrl := gomock.NewController(t)
	s := mockClient.NewMockTiDBCloudServerlessClient(ctrl)
	defer HookGlobal(&NewServerlessClient, func(publicKey string, privateKey string, serverlessEndpoint string, userAgent string) (tidbcloud.TiDBCloudServerlessClient, error) {
		return s, nil
	})()

	exportId := "export-id"

	createExportResp := exportV1beta1.Export{}
	createExportResp.UnmarshalJSON([]byte(testUTExport(string(exportV1beta1.EXPORTSTATEENUM_RUNNING))))
	getExportResp := exportV1beta1.Export{}
	getExportResp.UnmarshalJSON([]byte(testUTExport(string(exportV1beta1.EXPORTSTATEENUM_SUCCEEDED))))

	s.EXPECT().CreateExport(gomock.Any(), gomock.Any(), gomock.Any()).Return(&createExportResp, nil)
	s.EXPECT().GetExport(gomock.Any(), gomock.Any(), exportId).Return(&getExportResp, nil).AnyTimes()
	s.EXPECT().DeleteExport(gomock.Any(), gomock.Any(), exportId).Return(nil, nil)

	testServerlessExportResource(t)
}

func testServerlessExportResource(t *testing.T) {
	serverlessExportResourceName := "tidbcloud_serverless_export.test"
	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read serverless export resource
			{
				Config:             testUTServerlessExportResourceConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(serverlessExportResourceName, "target.type", "LOCAL"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func testAccServerlessExportResourceConfig() string {
	return `
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
`
}

func testUTServerlessExportResourceConfig() string {
	return `
resource "tidbcloud_serverless_export" "test" {
	cluster_id = "cluster_id"
}
`
}

func testUTExport(state string) string {
	return fmt.Sprintf(`
{
    "exportId": "export-id",
    "name": "clusters/cluster-id/exports/export-id",
    "clusterId": "cluster-id",
    "createdBy": "apikey-S22Jxxxxx",
    "state": "%s",
    "exportOptions": {
        "fileType": "CSV",
        "database": "*",
        "table": "*",
        "compression": "GZIP",
        "filter": null
    },
    "target": {
        "type": "LOCAL"
    },
    "displayName": "SNAPSHOT_2025-03-20T05:53:56Z",
    "createTime": "2025-03-20T05:53:57.152Z",
    "snapshotTime": "2025-03-20T05:53:56.111Z"
}
`, state)
}
