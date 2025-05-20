package provider

import (
	"fmt"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	mockClient "github.com/tidbcloud/terraform-provider-tidbcloud/mock"
	"github.com/tidbcloud/terraform-provider-tidbcloud/tidbcloud"
	exportV1beta1 "github.com/tidbcloud/tidbcloud-cli/pkg/tidbcloud/v1beta1/serverless/export"
)

func TestAccServerlessExportsDataSource(t *testing.T) {
	serverlessExportsDataSourceName := "data.tidbcloud_serverless_exports.test"
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testServerlessExportsConfig,
				Check: resource.ComposeTestCheckFunc(
					func(s *terraform.State) error {
						_, ok := s.RootModule().Resources[serverlessExportsDataSourceName]
						if !ok {
							return fmt.Errorf("Not found: %s", serverlessExportsDataSourceName)
						}
						return nil
					},
				),
			},
		},
	})
}

func TestUTServerlessExportsDataSource(t *testing.T) {
	setupTestEnv()

	ctrl := gomock.NewController(t)
	s := mockClient.NewMockTiDBCloudServerlessClient(ctrl)
	defer HookGlobal(&NewServerlessClient, func(publicKey string, privateKey string, serverlessEndpoint string, userAgent string) (tidbcloud.TiDBCloudServerlessClient, error) {
		return s, nil
	})()

	resp := exportV1beta1.ListExportsResponse{}
	resp.UnmarshalJSON([]byte(testUTListExportsResponse))

	s.EXPECT().ListExports(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(&resp, nil).AnyTimes()

	testUTServerlessExportsDataSource(t)
}

func testUTServerlessExportsDataSource(t *testing.T) {
	serverlessExportsDataSourceName := "data.tidbcloud_serverless_exports.test"
	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testUTServerlessExportsConfig,
				Check: resource.ComposeTestCheckFunc(
					func(s *terraform.State) error {
						_, ok := s.RootModule().Resources[serverlessExportsDataSourceName]
						if !ok {
							return fmt.Errorf("Not found: %s", serverlessExportsDataSourceName)
						}
						return nil
					},
				),
			},
		},
	})
}

const testServerlessExportsConfig = `
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
data "tidbcloud_serverless_exports" "test" {
	cluster_id = tidbcloud_serverless_cluster.example.cluster_id
}
`

const testUTServerlessExportsConfig = `
data "tidbcloud_serverless_exports" "test" {
	cluster_id = "clusterId"
}
`

const testUTListExportsResponse = `
{
    "exports": [
        {
            "exportId": "export-id",
            "name": "clusters/cluster-id/exports/export-id",
            "clusterId": "cluster-id",
            "createdBy": "xxxxxxxx",
            "state": "FAILED",
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
	]
}
`