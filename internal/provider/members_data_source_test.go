package provider

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	mockClient "github.com/tidbcloud/terraform-provider-tidbcloud/mock"
	"github.com/tidbcloud/terraform-provider-tidbcloud/tidbcloud"
)

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
