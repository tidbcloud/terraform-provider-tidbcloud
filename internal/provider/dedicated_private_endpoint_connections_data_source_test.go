package provider

import (
	"fmt"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	mockClient "github.com/tidbcloud/terraform-provider-tidbcloud/mock"
	"github.com/tidbcloud/terraform-provider-tidbcloud/tidbcloud"
	"github.com/tidbcloud/tidbcloud-cli/pkg/tidbcloud/v1beta1/dedicated"
)

func TestUTDedicatedPrivateEndpointConnectionsDataSource(t *testing.T) {
	setupTestEnv()

	ctrl := gomock.NewController(t)
	s := mockClient.NewMockTiDBCloudDedicatedClient(ctrl)
	defer HookGlobal(&NewDedicatedClient, func(publicKey string, privateKey string, dedicatedEndpoint string, userAgent string) (tidbcloud.TiDBCloudDedicatedClient, error) {
		return s, nil
	})()

	listResp := dedicated.V1beta1ListPrivateEndpointConnectionsResponse{}
	listResp.UnmarshalJSON([]byte(testUTListPrivateEndpointConnectionsResponse))

	s.EXPECT().ListPrivateEndpointConnections(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(&listResp, nil).AnyTimes()

	testUTDedicatedPrivateEndpointConnectionsDataSource(t)
}

func testUTDedicatedPrivateEndpointConnectionsDataSource(t *testing.T) {
	dedicatedPrivateEndpointConnectionDataSourceName := "data.tidbcloud_dedicated_private_endpoint_connections.test"
	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testUTDedicatedPrivateEndpointConnectionsDataSourceConfig,
				Check: resource.ComposeTestCheckFunc(
					func(s *terraform.State) error {
						_, ok := s.RootModule().Resources[dedicatedPrivateEndpointConnectionDataSourceName]
						if !ok {
							return fmt.Errorf("Not found: %s", dedicatedPrivateEndpointConnectionDataSourceName)
						}
						return nil
					},
				),
			},
		},
	})
}

const testUTDedicatedPrivateEndpointConnectionsDataSourceConfig string = `
data "tidbcloud_dedicated_private_endpoint_connections" "test" {
	cluster_id = "cluster-id"
	node_group_id = "node-group-id"
}
`

const testUTListPrivateEndpointConnectionsResponse string = `
{
    "privateEndpointConnections": [
        {
            "name": "tidbNodeGroups/idg/privateEndpointConnections/id",
            "tidbNodeGroupId": "idg",
            "privateEndpointConnectionId": "id",
            "clusterId": "10400000000000",
            "clusterDisplayName": "test-tf",
            "labels": {
                "tidb.cloud/project": "300000"
            },
            "endpointId": "vpce-03367000000",
            "endpointState": "FAILED",
            "message": "The VPC Endpoint Id '{vpce-03367e000000 []}' does not exist",
            "regionId": "aws-us-west-2",
            "regionDisplayName": "Oregon (us-west-2)",
            "cloudProvider": "aws",
            "privateLinkServiceName": "com.amazonaws.vpce.us-west-2.vpce-svc-0e8000000000",
            "privateLinkServiceState": "ACTIVE",
            "tidbNodeGroupDisplayName": "DefaultGroup",
            "host": "privatelink-00002.yd000000i7u.clusters.dev.tidb-cloud.com",
            "port": 4000
        },
        {
            "name": "tidbNodeGroups/idg2/privateEndpointConnections/id2",
            "tidbNodeGroupId": "idg2",
            "privateEndpointConnectionId": "id2",
            "clusterId": "id2",
            "clusterDisplayName": "test-tf",
            "labels": {
                "tidb.cloud/project": "3100000"
            },
            "endpointId": "vpce-010000000000",
            "endpointState": "ACTIVE",
            "message": "",
            "regionId": "aws-us-west-2",
            "regionDisplayName": "Oregon (us-west-2)",
            "cloudProvider": "aws",
            "privateLinkServiceName": "com.amazonaws.vpce.us-west-2.vpce-svc-0e8a00000000",
            "privateLinkServiceState": "ACTIVE",
            "tidbNodeGroupDisplayName": "DefaultGroup",
            "host": "privatelink-1900000.y0000000u.clusters.dev.tidb-cloud.com",
            "port": 4000
        }
    ]
}
`
