package provider

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	mockClient "github.com/tidbcloud/terraform-provider-tidbcloud/mock"
	"github.com/tidbcloud/terraform-provider-tidbcloud/tidbcloud"
	branchV1beta1 "github.com/tidbcloud/tidbcloud-cli/pkg/tidbcloud/v1beta1/serverless/branch"
)

func TestAccServerlessBranchDataSource(t *testing.T) {
	serverlessBranchDataSourceName := "data.tidbcloud_serverless_branch.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testServerlessBranchDataSourceConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(serverlessBranchDataSourceName, "display_name", "test-tf"),
				),
			},
		},
	})
}

func TestUTServerlessBranchDataSource(t *testing.T) {
	setupTestEnv()

	ctrl := gomock.NewController(t)
	s := mockClient.NewMockTiDBCloudServerlessClient(ctrl)
	defer HookGlobal(&NewServerlessClient, func(publicKey string, privateKey string, serverlessEndpoint string, userAgent string) (tidbcloud.TiDBCloudServerlessClient, error) {
		return s, nil
	})()

	branchId := "branchId"

	getBranchResp := branchV1beta1.Branch{}
	getBranchResp.UnmarshalJSON([]byte(testUTBranchFull(string(branchV1beta1.BRANCHSTATE_ACTIVE))))

	s.EXPECT().GetBranch(gomock.Any(), gomock.Any(), branchId, branchV1beta1.BRANCHSERVICEGETBRANCHVIEWPARAMETER_FULL).Return(&getBranchResp, nil).AnyTimes()

	testUTServerlessBranchDataSource(t, branchId)
}

func testUTServerlessBranchDataSource(t *testing.T, branchId string) {
	serverlessBranchDataSourceName := "data.tidbcloud_serverless_branch.test"
	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testUTServerlessBranchDataSourceConfig(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(serverlessBranchDataSourceName, "display_name", "test"),
				),
			},
		},
	})
}

const testServerlessBranchDataSourceConfig = `
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

data "tidbcloud_serverless_branch" "test" {
	cluster_id = tidbcloud_serverless_cluster.example.cluster_id
	branch_id = tidbcloud_serverless_branch.test.branch_id
}
`

func testUTServerlessBranchDataSourceConfig() string {
	return `
data "tidbcloud_serverless_branch" "test" {
	cluster_id = "clusterId"
	branch_id = "branchId" 
}
`
}
