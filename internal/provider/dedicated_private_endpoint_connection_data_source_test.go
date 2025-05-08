package provider

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	mockClient "github.com/tidbcloud/terraform-provider-tidbcloud/mock"
	"github.com/tidbcloud/terraform-provider-tidbcloud/tidbcloud"
	"github.com/tidbcloud/tidbcloud-cli/pkg/tidbcloud/v1beta1/dedicated"
)

func TestUTDedicatedPrivateEndpointConnectionDataSource(t *testing.T) {
	setupTestEnv()

	ctrl := gomock.NewController(t)
	s := mockClient.NewMockTiDBCloudDedicatedClient(ctrl)
	defer HookGlobal(&NewDedicatedClient, func(publicKey string, privateKey string, dedicatedEndpoint string, userAgent string) (tidbcloud.TiDBCloudDedicatedClient, error) {
		return s, nil
	})()

	getPrivateEndpointConnectionResp := dedicated.V1beta1PrivateEndpointConnection{}
	getPrivateEndpointConnectionResp.UnmarshalJSON([]byte(testUTPrivateEndpointConnection(string(dedicated.PRIVATEENDPOINTCONNECTIONENDPOINTSTATE_ACTIVE))))

	s.EXPECT().GetPrivateEndpointConnection(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(&getPrivateEndpointConnectionResp, nil).AnyTimes()

	testUTDedicatedPrivateEndpointConnectionDataSource(t)
}

func testUTDedicatedPrivateEndpointConnectionDataSource(t *testing.T) {
	dedicatedPrivateEndpointConnectionDataSourceName := "data.tidbcloud_dedicated_private_endpoint_connection.test"
	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testUTDedicatedPrivateEndpointConnectionDataSourceConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dedicatedPrivateEndpointConnectionDataSourceName, "region_id", "aws-us-west-2"),
				),
			},
		},
	})
}

const testUTDedicatedPrivateEndpointConnectionDataSourceConfig = `
data "tidbcloud_dedicated_private_endpoint_connection" "test" {
	cluster_id                     = "cluster-id"
	node_group_id                  = "node-group-id"
    private_endpoint_connection_id = "privateEndpointConnection-id"
}
`
