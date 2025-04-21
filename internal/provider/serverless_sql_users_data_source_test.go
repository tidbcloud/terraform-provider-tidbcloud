package provider

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	mockClient "github.com/tidbcloud/terraform-provider-tidbcloud/mock"
	"github.com/tidbcloud/terraform-provider-tidbcloud/tidbcloud"
	"github.com/tidbcloud/tidbcloud-cli/pkg/tidbcloud/v1beta1/iam"
)

func TestAccServerlessSQLUsersDataSource(t *testing.T) {
	serverlessSQLUsersDataSourceName := "data.tidbcloud_serverless_sql_users.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testServerlessSQLUsersDataSourceConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(serverlessSQLUsersDataSourceName, "builtin_role", "role_admin"),
				),
			},
		},
	})
}

func TestUTServerlessSQLUsersDataSource(t *testing.T) {
	setupTestEnv()

	ctrl := gomock.NewController(t)
	s := mockClient.NewMockTiDBCloudIAMClient(ctrl)
	defer HookGlobal(&NewIAMClient, func(publicKey string, privateKey string, iamEndpoint string, userAgent string) (tidbcloud.TiDBCloudIAMClient, error) {
		return s, nil
	})()

	clusterId := "cluster_id"

	listUserResp := iam.ApiListSqlUsersRsp{}
	listUserResp.UnmarshalJSON([]byte(testUTApiListSqlUsersResponse))

	s.EXPECT().ListSQLUsers(gomock.Any(), clusterId, gomock.Any(), gomock.Any()).Return(&listUserResp, nil).AnyTimes()

	testUTServerlessSQLUsersDataSource(t)
}

func testUTServerlessSQLUsersDataSource(t *testing.T) {
	serverlessSQLUserDataSourceName := "data.tidbcloud_serverless_sql_users.test"
	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testUTServerlessSQLUsersDataSourceConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(serverlessSQLUserDataSourceName, "serverless_sql_users.#", "0"),
				),
			},
		},
	})
}

const testServerlessSQLUsersDataSourceConfig = `
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

data "tidbcloud_serverless_sql_users" "test" {
	cluster_id = tidbcloud_serverless_cluster.example.cluster_id
}
`

const testUTServerlessSQLUsersDataSourceConfig string = `
data "tidbcloud_serverless_sql_users" "test" {
	cluster_id = "cluster_id"
}
`

const testUTApiListSqlUsersResponse = `
	"sqlUsers": [
        {
            "userName": "xxxxxxxxxxxxxxx.root",
            "builtinRole": "role_admin",
            "authMethod": "mysql_native_password"
        },
        {
            "userName": "xxxxxxxxxxxxxxx.test",
            "builtinRole": "role_admin",
            "authMethod": "mysql_native_password"
        }
    ]
`
