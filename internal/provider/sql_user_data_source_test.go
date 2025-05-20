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

func TestAccSQLUserDataSource(t *testing.T) {
	sqlUserDataSourceName := "data.tidbcloud_sql_user.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testSQLUserDataSourceConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(sqlUserDataSourceName, "builtin_role", "role_admin"),
				),
			},
		},
	})
}

func TestUTSQLUserDataSource(t *testing.T) {
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

	testUTSQLUserDataSource(t, clusterId, fullName, builtinRole)
}

func testUTSQLUserDataSource(t *testing.T, clusterId, fullname, builtinRole string) {
	sqlUserDataSourceName := "data.tidbcloud_sql_user.test"
	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:             testUTSQLUserDataSourceConfig(clusterId, fullname),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(sqlUserDataSourceName, "user_name", fullname),
					resource.TestCheckResourceAttr(sqlUserDataSourceName, "builtin_role", builtinRole),
				),
			},
		},
	})
}

const testSQLUserDataSourceConfig = `
resource "tidbcloud_serverless_cluster" "example" {
   display_name = "test-tf"
   region = {
      name = "regions/aws-us-east-1"
   }
}

resource "tidbcloud_sql_user" "test" {
	cluster_id = tidbcloud_serverless_cluster.example.cluster_id	
	user_name    = "${tidbcloud_serverless_cluster.example.user_prefix}.test"
	password     = "123456"
	builtin_role = "role_admin"
}

data "tidbcloud_sql_user" "test" {
	cluster_id = tidbcloud_serverless_cluster.example.cluster_id	
	user_name    = "${tidbcloud_serverless_cluster.example.user_prefix}.test"
}
`

func testUTSQLUserDataSourceConfig(clusterId, userName string) string {
	return fmt.Sprintf(`
data "tidbcloud_sql_user" "test" {
	cluster_id = "%s"
	user_name    = "%s" 
}
`, clusterId, userName)
}