package provider

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	mockClient "github.com/tidbcloud/terraform-provider-tidbcloud/mock"
	"github.com/tidbcloud/terraform-provider-tidbcloud/tidbcloud"
	"github.com/tidbcloud/tidbcloud-cli/pkg/tidbcloud/v1beta1/dedicated"
)

func TestUTDedicatedAuditLogFilterRuleResource(t *testing.T) {
	setupTestEnv()

	ctrl := gomock.NewController(t)
	s := mockClient.NewMockTiDBCloudDedicatedClient(ctrl)
	defer HookGlobal(&NewDedicatedClient, func(publicKey string, privateKey string, dedicatedEndpoint string, userAgent string) (tidbcloud.TiDBCloudDedicatedClient, error) {
		return s, nil
	})()

	resp := &dedicated.V1beta1AuditLogFilterRule{}
	resp.UnmarshalJSON([]byte(testUtV1beta1AuditLogFilterRuleResp))

	s.EXPECT().CreateAuditLogFilterRule(gomock.Any(), gomock.Any(), gomock.Any()).Return(resp, nil)
	s.EXPECT().GetAuditLogFilterRule(gomock.Any(), gomock.Any(), gomock.Any()).Return(resp, nil).AnyTimes()
	s.EXPECT().DeleteAuditLogFilterRule(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)

	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testUTDedicatedAuditLogFilterRuleResourceConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("tidbcloud_dedicated_audit_log_filter_rule.test", "user_expr", ".*"),
					resource.TestCheckResourceAttr("tidbcloud_dedicated_audit_log_filter_rule.test", "db_expr", ".*"),
					resource.TestCheckResourceAttr("tidbcloud_dedicated_audit_log_filter_rule.test", "table_expr", ".*"),
				),
			},
			// Delete testing is automatic
		},
	})
}

func testUTDedicatedAuditLogFilterRuleResourceConfig() string {
	return `
resource "tidbcloud_dedicated_audit_log_filter_rule" "test" {
    cluster_id       = "cluster-id"
    user_expr        = ".*"
    db_expr          = ".*"
    table_expr       = ".*"
    access_type_list = ["All"]
}
`
}

const testUtV1beta1AuditLogFilterRuleResp = `
{
    "name": "auditLogFilterRules/1",
    "auditLogFilterRuleId": "1",
    "clusterId": "cluster-id",
    "userExpr": ".*",
    "dbExpr": ".*",
    "tableExpr": ".*",
    "accessTypeList": [
        "All"
    ]
}
`
