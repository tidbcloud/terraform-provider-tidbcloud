package provider

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	mockClient "github.com/tidbcloud/terraform-provider-tidbcloud/mock"
	"github.com/tidbcloud/terraform-provider-tidbcloud/tidbcloud"
	"github.com/tidbcloud/tidbcloud-cli/pkg/tidbcloud/v1beta1/dedicated"
)

func TestUTDedicatedAuditLogFilterRuleDataSource(t *testing.T) {
	setupTestEnv()

	ctrl := gomock.NewController(t)
	s := mockClient.NewMockTiDBCloudDedicatedClient(ctrl)
	defer HookGlobal(&NewDedicatedClient, func(publicKey string, privateKey string, dedicatedEndpoint string, userAgent string) (tidbcloud.TiDBCloudDedicatedClient, error) {
		return s, nil
	})()

	resp := &dedicated.V1beta1AuditLogFilterRule{}
	resp.UnmarshalJSON([]byte(testUtV1beta1AuditLogFilterRuleResp))

	s.EXPECT().GetAuditLogFilterRule(gomock.Any(), gomock.Any(), gomock.Any()).Return(resp, nil).AnyTimes()

	dedicatedAuditLogFilterRuleDataSourceName := "data.tidbcloud_dedicated_audit_log_filter_rule.test"
	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testUTDedicatedAuditLogFilterRuleDataSourceConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dedicatedAuditLogFilterRuleDataSourceName, "user_expr", ".*"),
					resource.TestCheckResourceAttr(dedicatedAuditLogFilterRuleDataSourceName, "db_expr", ".*"),
					resource.TestCheckResourceAttr(dedicatedAuditLogFilterRuleDataSourceName, "table_expr", ".*"),
				),
			},
		},
	})
}

func testUTDedicatedAuditLogFilterRuleDataSourceConfig() string {
	return `
data "tidbcloud_dedicated_audit_log_filter_rule" "test" {
    cluster_id = "cluster-id"
    audit_log_filter_rule_id = "audit-log-filter-rule-id"
}
`
}
