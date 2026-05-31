package provider

import (
	"context"
	"fmt"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	mockClient "github.com/tidbcloud/terraform-provider-tidbcloud/mock"
	"github.com/tidbcloud/terraform-provider-tidbcloud/tidbcloud"
)

func TestAccMemberResource(t *testing.T) {
	memberResourceName := "tidbcloud_member.test"
	// Use a unique email so repeated runs do not collide with stale invites.
	email := fmt.Sprintf("tf-acc-%s@example.com", GenerateRandomString(8))
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccMemberResourceConfig(email, "org:member"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(memberResourceName, "email", email),
					resource.TestCheckResourceAttr(memberResourceName, "org_role", "org:member"),
					resource.TestCheckResourceAttrSet(memberResourceName, "user_id"),
					resource.TestCheckResourceAttrSet(memberResourceName, "status"),
				),
			},
			// Update testing
			{
				Config: testAccMemberResourceConfig(email, "org:billing_viewer"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(memberResourceName, "org_role", "org:billing_viewer"),
				),
			},
		},
	})
}

// memberStore is a minimal stateful fake of the member API used by the unit
// tests so that List/Invite/Update/Delete behave consistently across the many
// reads the testing framework performs.
type memberStore struct {
	email   string
	userId  string
	orgRole string
	status  string
	deleted bool
}

func (m *memberStore) toUser() tidbcloud.OpenApiUser {
	email, userId, orgRole, status := m.email, m.userId, m.orgRole, m.status
	return tidbcloud.OpenApiUser{
		Email:   &email,
		UserId:  &userId,
		Status:  &status,
		OrgRole: &tidbcloud.OpenApiRbacRole{RbacRole: orgRole},
	}
}

func newMemberMock(t *testing.T) (*mockClient.MockTiDBCloudIAMClient, *memberStore) {
	t.Helper()
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)
	s := mockClient.NewMockTiDBCloudIAMClient(ctrl)
	store := &memberStore{
		email:   "tf-test@example.com",
		userId:  "user-123",
		orgRole: "org:member",
		status:  "Pending",
	}

	s.EXPECT().InviteMembers(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, body *tidbcloud.OpenApiInviteUsersReq) (*tidbcloud.OpenApiInviteUsersRsp, error) {
			store.deleted = false
			if body.OrgRole != nil {
				store.orgRole = body.OrgRole.RbacRole
			}
			return &tidbcloud.OpenApiInviteUsersRsp{
				Success: Ptr(true),
				Users:   []tidbcloud.OpenApiInviteUserResult{{Email: &store.email, UserId: &store.userId}},
			}, nil
		}).AnyTimes()

	s.EXPECT().ListMembers(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, _ *tidbcloud.ListMembersParams) (*tidbcloud.OpenApiListUsersRsp, error) {
			if store.deleted {
				return &tidbcloud.OpenApiListUsersRsp{}, nil
			}
			return &tidbcloud.OpenApiListUsersRsp{Users: []tidbcloud.OpenApiUser{store.toUser()}}, nil
		}).AnyTimes()

	s.EXPECT().UpdateMember(gomock.Any(), store.userId, gomock.Any()).DoAndReturn(
		func(_ context.Context, _ string, body *tidbcloud.OpenApiUpdateUserReq) error {
			if body.OrgRole != nil {
				store.orgRole = body.OrgRole.RbacRole
			}
			return nil
		}).AnyTimes()

	s.EXPECT().DeleteMember(gomock.Any(), store.userId).DoAndReturn(
		func(_ context.Context, _ string) error {
			store.deleted = true
			return nil
		}).AnyTimes()

	return s, store
}

func TestUTMemberResource(t *testing.T) {
	setupTestEnv()

	s, store := newMemberMock(t)
	defer HookGlobal(&NewIAMClient, func(publicKey string, privateKey string, iamEndpoint string, userAgent string) (tidbcloud.TiDBCloudIAMClient, error) {
		return s, nil
	})()

	memberResourceName := "tidbcloud_member.test"
	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read
			{
				Config: testUTMemberResourceConfig(store.email, "org:member"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(memberResourceName, "email", store.email),
					resource.TestCheckResourceAttr(memberResourceName, "org_role", "org:member"),
					resource.TestCheckResourceAttr(memberResourceName, "user_id", store.userId),
					resource.TestCheckResourceAttr(memberResourceName, "status", "Pending"),
				),
			},
			// Update org role in place
			{
				Config: testUTMemberResourceConfig(store.email, "org:owner"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(memberResourceName, "org_role", "org:owner"),
				),
			},
			// Delete is performed automatically by the test framework.
		},
	})
}

func testAccMemberResourceConfig(email, orgRole string) string {
	return fmt.Sprintf(`
resource "tidbcloud_member" "test" {
	email    = "%s"
	org_role = "%s"
}
`, email, orgRole)
}

func testUTMemberResourceConfig(email, orgRole string) string {
	return fmt.Sprintf(`
resource "tidbcloud_member" "test" {
	email    = "%s"
	org_role = "%s"
}
`, email, orgRole)
}
