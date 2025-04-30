package provider

import (
	"fmt"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	mockClient "github.com/tidbcloud/terraform-provider-tidbcloud/mock"
	"github.com/tidbcloud/terraform-provider-tidbcloud/tidbcloud"
	branchV1beta1 "github.com/tidbcloud/tidbcloud-cli/pkg/tidbcloud/v1beta1/serverless/branch"
)

func TestAccServerlessBranchResource(t *testing.T) {
	serverlessBranchResourceName := "tidbcloud_serverless_branch.test"
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccServerlessBranchResourceConfig(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(serverlessBranchResourceName, "display_name", "test-tf"),
				),
			},
		},
	})
}

func TestUTServerlessBranchResource(t *testing.T) {
	setupTestEnv()

	ctrl := gomock.NewController(t)
	s := mockClient.NewMockTiDBCloudServerlessClient(ctrl)
	defer HookGlobal(&NewServerlessClient, func(publicKey string, privateKey string, serverlessEndpoint string, userAgent string) (tidbcloud.TiDBCloudServerlessClient, error) {
		return s, nil
	})()

	branchId := "branchId"

	createBranchResp := branchV1beta1.Branch{}
	createBranchResp.UnmarshalJSON([]byte(testUTBranch(string(branchV1beta1.BRANCHSTATE_CREATING))))
	getBranchResp := branchV1beta1.Branch{}
	getBranchResp.UnmarshalJSON([]byte(testUTBranch(string(branchV1beta1.BRANCHSTATE_ACTIVE))))
	getBranchFullResp := branchV1beta1.Branch{}
	getBranchFullResp.UnmarshalJSON([]byte(testUTBranchFull(string(branchV1beta1.BRANCHSTATE_ACTIVE))))
	s.EXPECT().CreateBranch(gomock.Any(), gomock.Any(), gomock.Any()).Return(&createBranchResp, nil)

	s.EXPECT().GetBranch(gomock.Any(), gomock.Any(), branchId, branchV1beta1.BRANCHSERVICEGETBRANCHVIEWPARAMETER_BASIC).Return(&getBranchResp, nil).AnyTimes()
	s.EXPECT().GetBranch(gomock.Any(), gomock.Any(), branchId, branchV1beta1.BRANCHSERVICEGETBRANCHVIEWPARAMETER_FULL).Return(&getBranchFullResp, nil).Times(2)
	s.EXPECT().DeleteBranch(gomock.Any(), gomock.Any(), branchId).Return(nil, nil)

	testServerlessBranchResource(t)
}

func testServerlessBranchResource(t *testing.T) {
	serverlessBranchResourceName := "tidbcloud_serverless_branch.test"
	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read serverless branch resource
			{
				Config:             testUTServerlessBranchResourceConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(serverlessBranchResourceName, "display_name", "test"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func testAccServerlessBranchResourceConfig() string {
	return `
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
`
}

func testUTServerlessBranchResourceConfig() string {
	return `
resource "tidbcloud_serverless_branch" "test" {
	cluster_id = "clusterId"
	display_name = "test"
	parent_id = "clusterId"
}
`
}

func testUTBranch(state string) string {
	return fmt.Sprintf(`{
    "name": "clusters/10163479507301863242/branches/branchId",
    "branchId": "branchId",
    "displayName": "test",
    "clusterId": "clusterId",
    "parentId": "clusterId",
    "createdBy": "apikey-xxxxxx",
    "state": "%s",
    "createTime": "2025-03-13T16:13:49Z",
    "updateTime": "2025-03-13T16:13:49Z",
    "annotations": {
        "tidb.cloud/has-set-password": "false"
    },
    "parentDisplayName": "Cluster0",
    "parentTimestamp": "2025-03-13T16:13:49Z"
}`, state)
}

func testUTBranchFull(state string) string {
	return fmt.Sprintf(`
{
    "name": "clusters/10163479507301863242/branches/branchId",
    "branchId": "branchId",
    "displayName": "test",
    "clusterId": "clusterId",
    "parentId": "clusterId",
    "createdBy": "apikey-KxxxxxC0",
    "state": "%s",
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
                "serviceName": "com.amazonaws.vpce.us-east-1.vpce-svc-0334xxxxxbc4d4",
                "availabilityZone": [
                    "use1-az1"
                ]
            }
        }
    },
    "userPrefix": "4FceZMwmFxxxxx",
    "usage": {
        "requestUnit": "0",
        "rowStorage": 1176692,
        "columnarStorage": 0
    },
    "createTime": "2025-03-13T16:13:49Z",
    "updateTime": "2025-03-13T16:14:27Z",
    "annotations": {
        "tidb.cloud/has-set-password": "false"
    },
    "parentDisplayName": "Cluster0",
    "parentTimestamp": "2025-03-13T16:13:49Z"
}
`, state)
}