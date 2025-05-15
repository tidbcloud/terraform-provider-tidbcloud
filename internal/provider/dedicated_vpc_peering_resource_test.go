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

func TestUTDedicatedVPCPeeringResource(t *testing.T) {
	setupTestEnv()

	ctrl := gomock.NewController(t)
	s := mockClient.NewMockTiDBCloudDedicatedClient(ctrl)
	defer HookGlobal(&NewDedicatedClient, func(publicKey string, privateKey string, dedicatedEndpoint string, userAgent string) (tidbcloud.TiDBCloudDedicatedClient, error) {
		return s, nil
	})()

	vpcPeeringId := "vpcPeering-id"

	createVPCPeeringResp := dedicated.Dedicatedv1beta1VpcPeering{}
	createVPCPeeringResp.UnmarshalJSON([]byte(testUTVPCPeering(string(dedicated.DEDICATEDV1BETA1VPCPEERINGSTATE_PENDING))))
	getVPCPeeringResp := dedicated.Dedicatedv1beta1VpcPeering{}
	getVPCPeeringResp.UnmarshalJSON([]byte(testUTVPCPeering(string(dedicated.DEDICATEDV1BETA1VPCPEERINGSTATE_ACTIVE))))

	s.EXPECT().CreateVPCPeering(gomock.Any(), gomock.Any()).Return(&createVPCPeeringResp, nil)
	s.EXPECT().GetVPCPeering(gomock.Any(), vpcPeeringId).Return(&getVPCPeeringResp, nil).AnyTimes()
	s.EXPECT().DeleteVPCPeering(gomock.Any(), vpcPeeringId).Return(nil)

	testDedicatedVPCPeeringResource(t)
}

func testDedicatedVPCPeeringResource(t *testing.T) {
	dedicatedVPCPeeringResourceName := "tidbcloud_dedicated_vpc_peering.test"
	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read dedicated vpc peering resource
			{
				Config: testUTDedicatedVPCPeeringResourceConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dedicatedVPCPeeringResourceName, "tidb_cloud_vpc_id", "tidb_cloud_vpc_id"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func testUTDedicatedVPCPeeringResourceConfig() string {
	return `
resource "tidbcloud_dedicated_vpc_peering" "test" {
	tidb_cloud_region_id = "aws-us-west-2"
    customer_region_id = "aws-us-west-2"
    customer_account_id = "customer_account_id"
    customer_vpc_id = "customer_vpc_id"
    customer_vpc_cidr = "172.16.32.0/21"
}
`
}

func testUTVPCPeering(state string) string {
	return fmt.Sprintf(`
{
    "name": "vpcPeerings/vpcPeering-id",
    "vpcPeeringId": "vpcPeering-id",
    "labels": {
        "tidb.cloud/project": "0000000"
    },
    "tidbCloudRegionId": "aws-us-west-2",
    "customerRegionId": "aws-us-west-2",
    "customerAccountId": "customer_account_id",
    "customerVpcId": "customer_vpc_id",
    "customerVpcCidr": "172.16.32.0/21",
    "tidbCloudCloudProvider": "aws",
    "tidbCloudAccountId": "385000000000",
    "tidbCloudVpcId": "tidb_cloud_vpc_id",
    "tidbCloudVpcCidr": "172.16.0.0/21",
    "state": "%s",
    "awsVpcPeeringConnectionId": "pcx-0e6887e02000000"
}
`, state)
}
