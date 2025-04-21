package provider

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	mockClient "github.com/tidbcloud/terraform-provider-tidbcloud/mock"
	"github.com/tidbcloud/terraform-provider-tidbcloud/tidbcloud"
	"github.com/tidbcloud/tidbcloud-cli/pkg/tidbcloud/v1beta1/iam"
)

func TestAccServerlessSQLUserResource(t *testing.T) {
	serverlessSQLUserResourceName := "tidbcloud_serverless_sql_user.test"
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccServerlessSQLUserResourceConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(serverlessSQLUserResourceName, "user_name", "test"),
					resource.TestCheckResourceAttr(serverlessSQLUserResourceName, "builtin_role", "role_admin"),
				),
			},
			// Update testing
			{
				Config: testAccServerlessSQLUserResourceUpdateConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("tidbcloud_serverless_sql_user.test", "password"),
				),
			},
		},
	})
}

func TestUTServerlessSQLUserResource(t *testing.T) {
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
	password := "123456"
	builtinRole := "role_admin"
	customRoles := []string{"role1", "role2"}
	customRolesJson, _ := json.Marshal(customRoles)
	customRolesStr := string(customRolesJson)

	createUserResp := iam.ApiSqlUser{}
	createUserResp.UnmarshalJSON([]byte(testUTApiSqlUser(userName, userPrefix, builtinRole, "")))
	getUserResp := iam.ApiSqlUser{}
	getUserResp.UnmarshalJSON([]byte(testUTApiSqlUser(userName, userPrefix, builtinRole, "")))
	getUserAfterUpdateResp := iam.ApiSqlUser{}
	getUserAfterUpdateResp.UnmarshalJSON([]byte(testUTApiSqlUser(userName, userPrefix, builtinRole, customRolesStr)))

	s.EXPECT().CreateSQLUser(gomock.Any(), clusterId, gomock.Any()).Return(&createUserResp, nil)
	gomock.InOrder(
		s.EXPECT().GetSQLUser(gomock.Any(), clusterId, fullName).Return(&getUserResp, nil).Times(2),
		s.EXPECT().GetSQLUser(gomock.Any(), clusterId, fullName).Return(&getUserAfterUpdateResp, nil).Times(1),
	)
	s.EXPECT().UpdateSQLUser(gomock.Any(), clusterId, fullName, gomock.Any()).Return(&getUserAfterUpdateResp, nil)
	s.EXPECT().DeleteSQLUser(gomock.Any(), clusterId, fullName).Return(nil, nil)

	testServerlessSQLUserResource(t, clusterId, fullName, password, builtinRole, customRolesStr)
}

func testServerlessSQLUserResource(t *testing.T, clusterId, userName, password, builtinRole, customRoles string) {
	serverlessSQLUserResourceName := "tidbcloud_serverless_sql_user.test"
	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read serverless SQL User resource
			{
				Config:             testUTServerlessSQLUserResourceConfig(clusterId, userName, password, builtinRole),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(serverlessSQLUserResourceName, "user_name", userName),
					resource.TestCheckResourceAttr(serverlessSQLUserResourceName, "password", password),
					resource.TestCheckResourceAttr(serverlessSQLUserResourceName, "builtin_role", builtinRole),
				),
			},
			// // Update correctly
			{
				Config:             testUTServerlessSQLUserResourceUpdateConfig(clusterId, userName, password, builtinRole, customRoles),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(serverlessSQLUserResourceName, "custom_roles.#"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

const testAccServerlessSQLUserResourceConfig = `
resource "tidbcloud_serverless_cluster" "example" {
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
`

const testAccServerlessSQLUserResourceUpdateConfig = `
resource "tidbcloud_serverless_cluster" "example" {
   display_name = "test-tf"
   region = {
      name = "regions/aws-us-east-1"
   }
}

resource "tidbcloud_serverless_sql_user" "test" {
	cluster_id = tidbcloud_serverless_cluster.example.cluster_id	
	user_name    = "${tidbcloud_serverless_cluster.example.user_prefix}.test"
	password     = "456789"
	builtin_role = "role_admin"
}	
`

func testUTServerlessSQLUserResourceConfig(clusterId, userName, password, builtinRole string) string {
	return fmt.Sprintf(`
resource "tidbcloud_serverless_sql_user" "test" {
	cluster_id   = "%s"
	user_name    = "%s"
	password     = "%s"
	builtin_role = "%s"
}
`, clusterId, userName, password, builtinRole)
}

func testUTServerlessSQLUserResourceUpdateConfig(clusterId, fullName, password, builtinRole, customRoles string) string {
	return fmt.Sprintf(`
resource "tidbcloud_serverless_sql_user" "test" {
	cluster_id   = "%s"
	user_name    = "%s"
	password     = "%s"
	builtin_role = "%s"
	custom_roles = %v
}
`, clusterId, fullName, password, builtinRole, customRoles)
}

func testUTApiSqlUser(userName, prefix, builtinRole, customRoles string) string {
	var res string
	if customRoles == "" {
		res = fmt.Sprintf(`
{
    "userName": "%s.%s",
    "builtinRole": "%s",
    "authMethod": "mysql_native_password"
}
`, prefix, userName, builtinRole)
	} else {
		res = fmt.Sprintf(`
{
    "userName": "%s.%s",
    "builtinRole": "%s",
    "authMethod": "mysql_native_password",
	"customRoles": %s
}
`, prefix, userName, builtinRole, customRoles)
	}
	return res
}
