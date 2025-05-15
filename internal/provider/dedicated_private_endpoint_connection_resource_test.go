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

func TestUTDedicatedPrivateEndpointConnectionResource(t *testing.T) {
	setupTestEnv()

	ctrl := gomock.NewController(t)
	s := mockClient.NewMockTiDBCloudDedicatedClient(ctrl)
	defer HookGlobal(&NewDedicatedClient, func(publicKey string, privateKey string, dedicatedEndpoint string, userAgent string) (tidbcloud.TiDBCloudDedicatedClient, error) {
		return s, nil
	})()

	createPrivateEndpointConnectionResp := dedicated.V1beta1PrivateEndpointConnection{}
	createPrivateEndpointConnectionResp.UnmarshalJSON([]byte(testUTPrivateEndpointConnection(string(dedicated.PRIVATEENDPOINTCONNECTIONENDPOINTSTATE_PENDING))))
	getPrivateEndpointConnectionResp := dedicated.V1beta1PrivateEndpointConnection{}
	getPrivateEndpointConnectionResp.UnmarshalJSON([]byte(testUTPrivateEndpointConnection(string(dedicated.PRIVATEENDPOINTCONNECTIONENDPOINTSTATE_ACTIVE))))

	s.EXPECT().CreatePrivateEndpointConnection(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(&createPrivateEndpointConnectionResp, nil)
	s.EXPECT().GetPrivateEndpointConnection(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(&getPrivateEndpointConnectionResp, nil).AnyTimes()
	s.EXPECT().DeletePrivateEndpointConnection(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)

	testDedicatedPrivateEndpointConnectionResource(t)
}

func testDedicatedPrivateEndpointConnectionResource(t *testing.T) {
	dedicatedPrivateEndpointConnectionResourceName := "tidbcloud_dedicated_private_endpoint_connection.test"
	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read dedicated vpc peering resource
			{
				Config: testUTDedicatedPrivateEndpointConnectionResourceConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dedicatedPrivateEndpointConnectionResourceName, "region_id", "aws-us-west-2"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func testUTDedicatedPrivateEndpointConnectionResourceConfig() string {
	return `
resource "tidbcloud_dedicated_private_endpoint_connection" "test" {
	cluster_id = "clusterId"
	node_group_id = "nodeGroupId"
	endpoint_id = "endpointId"
}
`
}

func testUTPrivateEndpointConnection(state string) string {
	return fmt.Sprintf(`
{
    "name": "tidbNodeGroups/nodeGroupId/privateEndpointConnections/id",
    "tidbNodeGroupId": "nodeGroupId",
    "privateEndpointConnectionId": "id",
    "clusterId": "clusterId",
    "clusterDisplayName": "test-tf",
    "labels": {
        "tidb.cloud/project": "3100000"
    },
    "endpointId": "endpointId",
    "endpointState": "%s",
    "message": "",
    "regionId": "aws-us-west-2",
    "regionDisplayName": "Oregon (us-west-2)",
    "cloudProvider": "aws",
    "privateLinkServiceName": "com.amazonaws.vpce.us-west-2.vpce-svc-0e8a2cd00000",
    "privateLinkServiceState": "ACTIVE",
    "tidbNodeGroupDisplayName": "DefaultGroup",
    "host": "",
    "port": 4000
}
`, state)
}
