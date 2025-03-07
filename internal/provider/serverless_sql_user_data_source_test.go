package provider

import (
	"fmt"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	mockClient "github.com/tidbcloud/terraform-provider-tidbcloud/mock"
	"github.com/tidbcloud/terraform-provider-tidbcloud/tidbcloud"
	"github.com/tidbcloud/tidbcloud-cli/pkg/tidbcloud/v1beta1/iam"
)

func TestAccServerlessSQLUserDataSource(t *testing.T) {
	serverlessSQLUserDataSourceName := "data.tidbcloud_serverless_sql_user.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testServerlessSQLUserDataSourceConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(serverlessSQLUserDataSourceName, "builtin_role", "role_admin"),
				),
			},
		},
	})
}

func TestUTServerlessSQLUserDataSource(t *testing.T) {
	setupTestEnv()

	ctrl := gomock.NewController(t)
	s := mockClient.NewMockTiDBCloudIAMClient(ctrl)
	defer HookGlobal(&NewIAMClient, func(publicKey string, privateKey string, iamEndpoint string, userAgent string) (tidbcloud.TiDBCloudIAMClient, error) {
		return s, nil
	})()
	
	clusterId := "cluster_id"
	userName := "test"
	userPrefix := "prefix"
	fullName := fmt.Sprintf("%s.%s", userPrefix, userName)
	builtinRole := "role_admin"

	getUserResp := iam.ApiSqlUser{}
	getUserResp.UnmarshalJSON([]byte(testUTApiSqlUser(userName, userPrefix, builtinRole, "")))

	s.EXPECT().GetSQLUser(gomock.Any(), clusterId, fullName).Return(&getUserResp, nil).AnyTimes()

	testUTServerlessSQLUserDataSource(t, clusterId, fullName, builtinRole)
}

func testUTServerlessSQLUserDataSource(t *testing.T, clusterId, fullname, builtinRole string) {
	serverlessSQLUserDataSourceName := "data.tidbcloud_serverless_sql_user.test"
	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:             testUTServerlessSQLUserDataSourceConfig(clusterId, fullname),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(serverlessSQLUserDataSourceName, "user_name", fullname),
					resource.TestCheckResourceAttr(serverlessSQLUserDataSourceName, "builtin_role", builtinRole),
				),
			},
		},
	})
}

const testServerlessSQLUserDataSourceConfig = `
resource "tidbcloud_serverless_sql_user" "example" {
   display_name = "test-tf"
   region = {
      name = "regions/aws-us-east-1"
   }
}

resource "tidbcloud_serverless_sql_user" "test" {
	cluster_id = tidbcloud_serverless_cluster.example.cluster_id	
	user_name    = "${tidbcloud_serverless_cluster.example.user_prefix}.test"
	password     = "123456"
	builtin_role = "role_admin"
}

data "tidbcloud_serverless_sql_user" "test" {
	cluster_id = tidbcloud_serverless_cluster.example.cluster_id	
	user_name    = "${tidbcloud_serverless_cluster.example.user_prefix}.test"
}
`

func testUTServerlessSQLUserDataSourceConfig(clusterId, userName string) string {
	return fmt.Sprintf(`
data "tidbcloud_serverless_sql_user" "test" {
	cluster_id = "%s"
	user_name    = "%s" 
}
`, clusterId, userName)
}