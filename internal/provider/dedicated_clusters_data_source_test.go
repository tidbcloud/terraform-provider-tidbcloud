package provider

import (
	"fmt"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	mockClient "github.com/tidbcloud/terraform-provider-tidbcloud/mock"
	"github.com/tidbcloud/terraform-provider-tidbcloud/tidbcloud"
	"github.com/tidbcloud/tidbcloud-cli/pkg/tidbcloud/v1beta1/dedicated"
)

func TestAccDedicatedClustersDataSource(t *testing.T) {
	dedicatedClustersDataSourceName := "data.tidbcloud_dedicated_clusters.test"
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testDedicatedClustersConfig,
				Check: resource.ComposeTestCheckFunc(
					func(s *terraform.State) error {
						_, ok := s.RootModule().Resources[dedicatedClustersDataSourceName]
						if !ok {
							return fmt.Errorf("Not found: %s", dedicatedClustersDataSourceName)
						}
						return nil
					},
				),
			},
		},
	})
}

func TestUTDedicatedClustersDataSource(t *testing.T) {
	setupTestEnv()

	ctrl := gomock.NewController(t)
	s := mockClient.NewMockTiDBCloudDedicatedClient(ctrl)
	defer HookGlobal(&NewDedicatedClient, func(publicKey string, privateKey string, dedicatedEndpoint string, userAgent string) (tidbcloud.TiDBCloudDedicatedClient, error) {
		return s, nil
	})()

	var resp dedicated.TidbCloudOpenApidedicatedv1beta1ListClustersResponse
	resp.UnmarshalJSON([]byte(testUTTidbCloudOpenApidedicatedv1beta1ListClustersResponse))

	s.EXPECT().ListClusters(gomock.Any(), gomock.Any(), gomock.Any(), nil).Return(&resp, nil).AnyTimes()

	testUTDedicatedClustersDataSource(t)
}

func testUTDedicatedClustersDataSource(t *testing.T) {
	dedicatedClustersDataSourceName := "data.tidbcloud_dedicated_clusters.test"
	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testUTDedicatedClustersConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(dedicatedClustersDataSourceName, "dedicated_clusters.#", "0"),
				),
			},
		},
	})
}

const testDedicatedClustersConfig = `
resource "tidbcloud_dedicated_cluster" "example" {
   display_name = "test-tf"
   region = {
      name = "regions/aws-us-east-1"
   }
}

data "tidbcloud_dedicated_clusters" "test" {}
`

const testUTDedicatedClustersConfig = `
data "tidbcloud_dedicated_clusters" "test" {}
`

const testUTTidbCloudOpenApidedicatedv1beta1ListClustersResponse = `
{
    "clusters": [
        {
            "name": "clusters/1067097xxx",
            "clusterId": "1067097xxx",
            "displayName": "test-tf2",
            "regionId": "aws-us-west-2",
            "labels": {
                "tidb.cloud/organization": "00000",
                "tidb.cloud/project": "00000000"
            },
            "tidbNodeSetting": {
                "nodeSpecKey": "2C4G",
                "tidbNodeGroups": [
                    {
                        "name": "tidbNodeGroups/19171xxxx",
                        "tidbNodeGroupId": "191713xxxx",
                        "clusterId": "1067097xxxxx",
                        "displayName": "DefaultGroup",
                        "nodeCount": 1,
                        "endpoints": [
                            {
                                "host": "tidb.xxx.clusters.dev.tidb-cloud.com",
                                "port": 4000,
                                "connectionType": "PUBLIC"
                            },
                            {
                                "host": "private-tidb.xxx.clusters.dev.tidb-cloud.com",
                                "port": 4000,
                                "connectionType": "VPC_PEERING"
                            },
                            {
                                "host": "privatelink-19171382.xxx.clusters.dev.tidb-cloud.com",
                                "port": 4000,
                                "connectionType": "PRIVATE_ENDPOINT"
                            }
                        ],
                        "nodeSpecKey": "2C4G",
                        "nodeSpecDisplayName": "2 vCPU, 4 GiB beta",
                        "isDefaultGroup": true,
                        "state": "ACTIVE"
                    }
                ],
                "nodeSpecDisplayName": "2 vCPU, 4 GiB beta"
            },
            "tikvNodeSetting": {
                "nodeCount": 3,
                "nodeSpecKey": "2C4G",
                "storageSizeGi": 60,
                "storageType": "Standard",
                "nodeSpecDisplayName": "2 vCPU, 4 GiB"
            },
            "port": 4000,
            "rootPassword": "",
            "state": "ACTIVE",
            "version": "v8.1.2",
            "createdBy": "apikey-Kxxxxx",
            "createTime": "2025-04-29T08:45:41.048Z",
            "updateTime": "2025-04-29T08:51:03.081Z",
            "regionDisplayName": "Oregon (us-west-2)",
            "cloudProvider": "aws",
            "annotations": {
                "tidb.cloud/available-features": "DELEGATE_USER,DISABLE_PUBLIC_LB,PRIVATELINK",
                "tidb.cloud/has-set-password": "false"
            }
        },
        {
            "name": "clusters/10659xxxxx",
            "clusterId": "10659xxxxx",
            "displayName": "xxxxx1",
            "regionId": "aws-us-west-2",
            "labels": {
                "tidb.cloud/organization": "00000",
                "tidb.cloud/project": "0000000"
            },
            "tidbNodeSetting": {
                "nodeSpecKey": "2C4G",
                "tidbNodeGroups": [
                    {
                        "name": "tidbNodeGroups/191",
                        "tidbNodeGroupId": "191",
                        "clusterId": "106597xxxx",
                        "displayName": "DefaultGroup",
                        "nodeCount": 1,
                        "endpoints": [
                            {
                                "host": "tidb.xxx.clusters.dev.tidb-cloud.com",
                                "port": 4000,
                                "connectionType": "PUBLIC"
                            },
                            {
                                "host": "private-tidb.xxx.clusters.dev.tidb-cloud.com",
                                "port": 4000,
                                "connectionType": "VPC_PEERING"
                            },
                            {
                                "host": "privatelink-191.xxx.clusters.dev.tidb-cloud.com",
                                "port": 4000,
                                "connectionType": "PRIVATE_ENDPOINT"
                            }
                        ],
                        "nodeSpecKey": "2C4G",
                        "nodeSpecDisplayName": "2 vCPU, 4 GiB beta",
                        "isDefaultGroup": true,
                        "state": "ACTIVE"
                    }
                ],
                "nodeSpecDisplayName": "2 vCPU, 4 GiB beta"
            },
            "tikvNodeSetting": {
                "nodeCount": 3,
                "nodeSpecKey": "2C4G",
                "storageSizeGi": 60,
                "storageType": "Standard",
                "nodeSpecDisplayName": "2 vCPU, 4 GiB"
            },
            "port": 4000,
            "rootPassword": "",
            "state": "ACTIVE",
            "version": "v8.1.2",
            "createdBy": "xxxxx@pingcap.com",
            "createTime": "2025-04-29T03:54:39.856Z",
            "updateTime": "2025-04-29T04:00:42.387Z",
            "regionDisplayName": "Oregon (us-west-2)",
            "cloudProvider": "aws",
            "annotations": {
                "tidb.cloud/available-features": "DELEGATE_USER,DISABLE_PUBLIC_LB,PRIVATELINK",
                "tidb.cloud/has-set-password": "false"
            }
        }
    ]
}
`
