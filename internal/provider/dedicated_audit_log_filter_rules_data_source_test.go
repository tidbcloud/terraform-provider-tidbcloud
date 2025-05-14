package provider

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	mockClient "github.com/tidbcloud/terraform-provider-tidbcloud/mock"
	"github.com/tidbcloud/terraform-provider-tidbcloud/tidbcloud"
	"github.com/tidbcloud/tidbcloud-cli/pkg/tidbcloud/v1beta1/dedicated"
)

func TestUTDedicatedAuditLogFilterRulesDataSource(t *testing.T) {
	setupTestEnv()

	ctrl := gomock.NewController(t)
	s := mockClient.NewMockTiDBCloudDedicatedClient(ctrl)
	defer HookGlobal(&NewDedicatedClient, func(publicKey string, privateKey string, dedicatedEndpoint string, userAgent string) (tidbcloud.TiDBCloudDedicatedClient, error) {
		return s, nil
	})()

	var resp dedicated.V1beta1ListAuditLogFilterRulesResponse
	resp.UnmarshalJSON([]byte(testUTListV1beta1AuditLogFilterRuleResp))

	s.EXPECT().ListAuditLogFilterRules(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(&resp, nil).AnyTimes()

	dedicatedAuditLogFilterRulesDataSourceName := "data.tidbcloud_dedicated_audit_log_filter_rules.test"
	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testUTDedicatedAuditLogFilterRulesConfig(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(dedicatedAuditLogFilterRulesDataSourceName, "audit_log_filter_rules.#", "1"),
				),
			},
		},
	})
}

func testUTDedicatedAuditLogFilterRulesConfig() string {
	return `
data "tidbcloud_dedicated_audit_log_filter_rules" "test" {
    cluster_id = "cluster-id"
}
`
}

const testUTListV1beta1AuditLogFilterRuleResp = `
{
    "auditLogFilterRules": [
        {
            "name": "auditLogFilterRules/4",
            "auditLogFilterRuleId": "4",
            "clusterId": "cluster-id",
            "userExpr": ".*",
            "dbExpr": ".*",
            "tableExpr": ".*",
            "accessTypeList": [
                "All"
            ]
        }
    ]
}
`
