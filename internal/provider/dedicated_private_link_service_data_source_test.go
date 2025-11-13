package provider

import (
	"fmt"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	mockClient "github.com/tidbcloud/terraform-provider-tidbcloud/mock"
	"github.com/tidbcloud/terraform-provider-tidbcloud/tidbcloud"
	"github.com/tidbcloud/tidbcloud-cli/pkg/tidbcloud/v1beta1/dedicated"
)

func TestUTDedicatedPrivateLinkServiceDataSource(t *testing.T) {
	setupTestEnv()

	ctrl := gomock.NewController(t)
	s := mockClient.NewMockTiDBCloudDedicatedClient(ctrl)
	defer HookGlobal(&NewDedicatedClient, func(publicKey string, privateKey string, dedicatedEndpoint string, userAgent string) (tidbcloud.TiDBCloudDedicatedClient, error) {
		return s, nil
	})()

	privateLinkServiceResp := dedicated.Dedicatedv1beta1PrivateLinkService{}
	if err := privateLinkServiceResp.UnmarshalJSON([]byte(testUTPrivateLinkService(string(dedicated.DEDICATEDV1BETA1PRIVATELINKSERVICESTATE_ACTIVE)))); err != nil {
		t.Fatalf("failed to unmarshal private link service response: %v", err)
	}

	s.EXPECT().GetPrivateLinkService(gomock.Any(), gomock.Any(), gomock.Any()).Return(&privateLinkServiceResp, nil).AnyTimes()

	testUTDedicatedPrivateLinkServiceDataSource(t)
}

func testUTDedicatedPrivateLinkServiceDataSource(t *testing.T) {
	dataSourceName := "data.tidbcloud_dedicated_private_link_service.test"

	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testUTDedicatedPrivateLinkServiceDataSourceConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, "service_name", "com.amazonaws.vpce.us-west-2.vpce-svc-0e8a2cd00000"),
					resource.TestCheckResourceAttr(dataSourceName, "available_zones.#", "2"),
				),
			},
		},
	})
}

const testUTDedicatedPrivateLinkServiceDataSourceConfig = `
data "tidbcloud_dedicated_private_link_service" "test" {
    cluster_id    = "cluster-id"
    node_group_id = "node-group-id"
}
`

func testUTPrivateLinkService(state string) string {
	return fmt.Sprintf(`
{
    "serviceName": "com.amazonaws.vpce.us-west-2.vpce-svc-0e8a2cd00000",
    "serviceDnsName": "vpce-svc-0e8a2cd00000.us-west-2.vpce.amazonaws.com",
    "availableZones": ["us-west-2a", "us-west-2b"],
    "state": "%s",
    "regionId": "aws-us-west-2",
    "regionDisplayName": "Oregon (us-west-2)",
    "cloudProvider": "aws"
}
`, state)
}
