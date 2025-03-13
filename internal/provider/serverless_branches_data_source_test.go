package provider

import (
	"fmt"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	mockClient "github.com/tidbcloud/terraform-provider-tidbcloud/mock"
	"github.com/tidbcloud/terraform-provider-tidbcloud/tidbcloud"
	branchV1beta1 "github.com/tidbcloud/tidbcloud-cli/pkg/tidbcloud/v1beta1/serverless/branch"
)

func TestAccServerlessBranchesDataSource(t *testing.T) {
	serverlessBranchesDataSourceName := "data.tidbcloud_serverless_branches.test"
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testServerlessBranchesConfig,
				Check: resource.ComposeTestCheckFunc(
					func(s *terraform.State) error {
						_, ok := s.RootModule().Resources[serverlessBranchesDataSourceName]
						if !ok {
							return fmt.Errorf("Not found: %s", serverlessBranchesDataSourceName)
						}
						return nil
					},
				),
			},
		},
	})
}

func TestUTServerlessBranchesDataSource(t *testing.T) {
	setupTestEnv()

	ctrl := gomock.NewController(t)
	s := mockClient.NewMockTiDBCloudServerlessClient(ctrl)
	defer HookGlobal(&NewServerlessClient, func(publicKey string, privateKey string, serverlessEndpoint string, userAgent string) (tidbcloud.TiDBCloudServerlessClient, error) {
		return s, nil
	})()

	resp := branchV1beta1.ListBranchesResponse{}
	resp.UnmarshalJSON([]byte(testUTListBranchesResponse))

	s.EXPECT().ListBranches(gomock.Any(), gomock.Any(), gomock.Any(), nil).
		Return(&resp, nil).AnyTimes()

	testUTServerlessBranchesDataSource(t)
}

func testUTServerlessBranchesDataSource(t *testing.T) {
	serverlessBranchesDataSourceName := "data.tidbcloud_serverless_branches.test"
	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testUTServerlessBranchesConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(serverlessBranchesDataSourceName, "serverless_branches.#", "0"),
				),
			},
		},
	})
}

const testServerlessBranchesConfig = `
resource "tidbcloud_serverless_cluster" "example" {
   display_name = "test-tf"
   region = {
      name = "regions/aws-us-east-1"
   }
}

resource "tidbcloud_serverless_branch" "test" {
	cluster_id = tidbcloud_serverless_cluster.example.cluster_id
	display_name = "test-tf"
	parent_id = tidbcloud_serverless_cluster.example.cluster_id
}

data "tidbcloud_serverless_branches" "test" {
	cluster_id = tidbcloud_serverless_cluster.example.cluster_id
}
`

const testUTServerlessBranchesConfig = `
data "tidbcloud_serverless_branches" "test" {
	cluster_id = "clusterId"
}
`

const testUTListBranchesResponse = `
{
    "branches": [
        {
            "name": "clusters/10163479507301863242/branches/branchId",
            "branchId": "branchId",
            "displayName": "test",
            "clusterId": "clusterId",
            "parentId": "clusterId",
            "createdBy": "apikey-Kxxxx",
            "state": "ACTIVE",
            "endpoints": {
                "public": {
                    "host": "gateway01.us-east-1.dev.shared.aws.tidbcloud.com",
                    "port": 4000,
                    "disabled": false
                },
                "private": {
                    "host": "gateway01-privatelink.us-east-1.dev.shared.aws.tidbcloud.com",
                    "port": 4000,
                    "aws": {
                        "serviceName": "com.amazonaws.vpce.us-east-1.vpce-svc-03xxxxxx",
                        "availabilityZone": [
                            "use1-az1"
                        ]
                    }
                }
            },
            "userPrefix": "4Fcexxxxxxx",
            "createTime": "2025-03-13T16:13:49Z",
            "updateTime": "2025-03-13T16:14:27Z",
            "annotations": {
                "tidb.cloud/has-set-password": "false"
            },
            "parentDisplayName": "Cluster0",
            "parentTimestamp": "2025-03-13T16:13:49Z"
        }
    ]
}
`
