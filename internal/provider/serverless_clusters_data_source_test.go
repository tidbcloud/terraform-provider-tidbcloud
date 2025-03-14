package provider

import (
	"fmt"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	mockClient "github.com/tidbcloud/terraform-provider-tidbcloud/mock"
	"github.com/tidbcloud/terraform-provider-tidbcloud/tidbcloud"
	clusterV1beta1 "github.com/tidbcloud/tidbcloud-cli/pkg/tidbcloud/v1beta1/serverless/cluster"
)

func TestAccServerlessClustersDataSource(t *testing.T) {
	serverlessClustersDataSourceName := "data.tidbcloud_serverless_clusters.test"
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testServerlessClustersConfig,
				Check: resource.ComposeTestCheckFunc(
					func(s *terraform.State) error {
						_, ok := s.RootModule().Resources[serverlessClustersDataSourceName]
						if !ok {
							return fmt.Errorf("Not found: %s", serverlessClustersDataSourceName)
						}
						return nil
					},
				),
			},
		},
	})
}

func TestUTServerlessClustersDataSource(t *testing.T) {
	setupTestEnv()

	ctrl := gomock.NewController(t)
	s := mockClient.NewMockTiDBCloudServerlessClient(ctrl)
	defer HookGlobal(&NewServerlessClient, func(publicKey string, privateKey string, serverlessEndpoint string, userAgent string) (tidbcloud.TiDBCloudServerlessClient, error) {
		return s, nil
	})()

	resp := clusterV1beta1.TidbCloudOpenApiserverlessv1beta1ListClustersResponse{}
	resp.UnmarshalJSON([]byte(testUTTidbCloudOpenApiserverlessv1beta1ListClustersResponse))

	s.EXPECT().ListClusters(gomock.Any(), gomock.Any(), gomock.Any(), nil, nil, nil).
		Return(&resp, nil).AnyTimes()

	testUTServerlessClustersDataSource(t)
}

func testUTServerlessClustersDataSource(t *testing.T) {
	serverlessClustersDataSourceName := "data.tidbcloud_serverless_clusters.test"
	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testUTServerlessClustersConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(serverlessClustersDataSourceName, "serverless_clusters.#", "0"),
				),
			},
		},
	})
}

const testServerlessClustersConfig = `
resource "tidbcloud_serverless_cluster" "example" {
   display_name = "test-tf"
   region = {
      name = "regions/aws-us-east-1"
   }
}

data "tidbcloud_serverless_clusters" "test" {}
`

const testUTServerlessClustersConfig = `
data "tidbcloud_serverless_clusters" "test" {}
`

const testUTTidbCloudOpenApiserverlessv1beta1ListClustersResponse = `
	"clusters": [
        {
            "name": "clusters/xxxxxxxxx",
            "clusterId": "xxxxxxxxxxx",
            "displayName": "test-tf",
            "region": {
                "name": "regions/aws-us-east-1",
                "regionId": "us-east-1",
                "cloudProvider": "aws",
                "displayName": "N. Virginia (us-east-1)",
                "provider": "aws"
            },
            "endpoints": {
                "public": {
                    "host": "gateway01.us-east-1.dev.shared.aws.tidbcloud.com",
                    "port": 4000,
                    "disabled": false,
                    "authorizedNetworks": [
                        {
                            "startIpAddress": "0.0.0.0",
                            "endIpAddress": "255.255.255.255",
                            "displayName": "Allow_all_public_connections"
                        }
                    ]
                },
                "private": {
                    "host": "gateway01-privatelink.us-east-1.dev.shared.aws.tidbcloud.com",
                    "port": 4000,
                    "aws": {
                        "serviceName": "com.amazonaws.vpce.us-east-1.vpce-svc-0334xxxxxxxxxx",
                        "availabilityZone": [
                            "use1-az1"
                        ]
                    }
                }
            },
            "rootPassword": "",
            "encryptionConfig": {
                "enhancedEncryptionEnabled": false
            },
            "highAvailabilityType": "ZONAL",
            "version": "v7.5.2",
            "createdBy": "xxxxxxx",
            "userPrefix": "xxxxxx",
            "state": "ACTIVE",
            "labels": {
                "tidb.cloud/organization": "xxxxx",
                "tidb.cloud/project": "xxxxx"
            },
            "annotations": {
                "tidb.cloud/available-features": "DISABLE_PUBLIC_LB,DELEGATE_USER",
                "tidb.cloud/has-set-password": "true"
            },
            "createTime": "2025-02-26T12:15:24.348Z",
            "updateTime": "2025-02-26T12:15:49Z",
            "auditLogConfig": {
                "enabled": false,
                "unredacted": false
            }
        }
	]
`
