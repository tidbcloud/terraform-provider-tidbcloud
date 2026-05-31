package provider

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	mockClient "github.com/tidbcloud/terraform-provider-tidbcloud/mock"
	"github.com/tidbcloud/terraform-provider-tidbcloud/tidbcloud"
)

// TestAccMembersDataSource lists the real organization members (read-only).
func TestAccMembersDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `data "tidbcloud_members" "all" {}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					// At least one member (the API key's own org owner) must exist.
					resource.TestMatchResourceAttr("data.tidbcloud_members.all", "members.#", regexp.MustCompile(`^[1-9][0-9]*$`)),
					resource.TestCheckResourceAttrSet("data.tidbcloud_members.all", "members.0.email"),
					resource.TestCheckResourceAttrSet("data.tidbcloud_members.all", "members.0.user_id"),
					resource.TestCheckResourceAttrSet("data.tidbcloud_members.all", "members.0.org_role"),
					resource.TestCheckResourceAttrSet("data.tidbcloud_members.all", "members.0.status"),
				),
			},
		},
	})
}

// TestAccMemberDataSource looks up a single existing member by email (read-only).
// Set TIDBCLOUD_TEST_MEMBER_EMAIL to an email that exists in the organization.
func TestAccMemberDataSource(t *testing.T) {
	email := os.Getenv("TIDBCLOUD_TEST_MEMBER_EMAIL")
	if email == "" {
		t.Skip("TIDBCLOUD_TEST_MEMBER_EMAIL must be set for TestAccMemberDataSource")
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`data "tidbcloud_member" "test" {
  email = %q
}`, email),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.tidbcloud_member.test", "email", email),
					resource.TestCheckResourceAttrSet("data.tidbcloud_member.test", "user_id"),
					resource.TestCheckResourceAttrSet("data.tidbcloud_member.test", "org_role"),
					resource.TestCheckResourceAttrSet("data.tidbcloud_member.test", "status"),
				),
			},
		},
	})
}

func testUTMember(email, userId, orgRole, status string) tidbcloud.OpenApiUser {
	e, u, s := email, userId, status
	return tidbcloud.OpenApiUser{
		Email:   &e,
		UserId:  &u,
		Status:  &s,
		OrgRole: &tidbcloud.OpenApiRbacRole{RbacRole: orgRole},
	}
}

func TestUTMembersDataSource(t *testing.T) {
	setupTestEnv()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	s := mockClient.NewMockTiDBCloudIAMClient(ctrl)
	defer HookGlobal(&NewIAMClient, func(publicKey string, privateKey string, iamEndpoint string, userAgent string) (tidbcloud.TiDBCloudIAMClient, error) {
		return s, nil
	})()

	member := testUTMember("tf-test@example.com", "user-123", "org:owner", "Active")
	s.EXPECT().ListMembers(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, _ *tidbcloud.ListMembersParams) (*tidbcloud.OpenApiListUsersRsp, error) {
			return &tidbcloud.OpenApiListUsersRsp{Users: []tidbcloud.OpenApiUser{member}}, nil
		}).AnyTimes()

	dataSourceName := "data.tidbcloud_members.test"
	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `data "tidbcloud_members" "test" {}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, "members.#", "1"),
					resource.TestCheckResourceAttr(dataSourceName, "members.0.email", "tf-test@example.com"),
					resource.TestCheckResourceAttr(dataSourceName, "members.0.user_id", "user-123"),
					resource.TestCheckResourceAttr(dataSourceName, "members.0.org_role", "org:owner"),
					resource.TestCheckResourceAttr(dataSourceName, "members.0.status", "Active"),
				),
			},
		},
	})
}

func TestUTMemberDataSource(t *testing.T) {
	setupTestEnv()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	s := mockClient.NewMockTiDBCloudIAMClient(ctrl)
	defer HookGlobal(&NewIAMClient, func(publicKey string, privateKey string, iamEndpoint string, userAgent string) (tidbcloud.TiDBCloudIAMClient, error) {
		return s, nil
	})()

	member := testUTMember("tf-test@example.com", "user-123", "org:member", "Pending")
	s.EXPECT().ListMembers(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, _ *tidbcloud.ListMembersParams) (*tidbcloud.OpenApiListUsersRsp, error) {
			return &tidbcloud.OpenApiListUsersRsp{Users: []tidbcloud.OpenApiUser{member}}, nil
		}).AnyTimes()

	dataSourceName := "data.tidbcloud_member.test"
	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
data "tidbcloud_member" "test" {
	email = "tf-test@example.com"
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, "email", "tf-test@example.com"),
					resource.TestCheckResourceAttr(dataSourceName, "user_id", "user-123"),
					resource.TestCheckResourceAttr(dataSourceName, "org_role", "org:member"),
					resource.TestCheckResourceAttr(dataSourceName, "status", "Pending"),
				),
			},
		},
	})
}
