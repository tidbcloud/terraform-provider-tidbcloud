package provider

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	mockClient "github.com/tidbcloud/terraform-provider-tidbcloud/mock"
	"github.com/tidbcloud/terraform-provider-tidbcloud/tidbcloud"
	"github.com/tidbcloud/tidbcloud-cli/pkg/tidbcloud/v1beta1/dedicated"
)

func TestUTDedicatedVPCPeeringDataSource(t *testing.T) {
	setupTestEnv()

	ctrl := gomock.NewController(t)
	s := mockClient.NewMockTiDBCloudDedicatedClient(ctrl)
	defer HookGlobal(&NewDedicatedClient, func(publicKey string, privateKey string, dedicatedEndpoint string, userAgent string) (tidbcloud.TiDBCloudDedicatedClient, error) {
		return s, nil
	})()

	vpcPeeringId := "vpcPeering-id"

	getVPCPeeringResp := dedicated.Dedicatedv1beta1VpcPeering{}
	getVPCPeeringResp.UnmarshalJSON([]byte(testUTVPCPeering(string(dedicated.DEDICATEDV1BETA1VPCPEERINGSTATE_ACTIVE))))

	s.EXPECT().GetVPCPeering(gomock.Any(), vpcPeeringId).Return(&getVPCPeeringResp, nil).AnyTimes()

	testUTDedicatedVPCPeeringDataSource(t)
}

func testUTDedicatedVPCPeeringDataSource(t *testing.T) {
	dedicatedVPCPeeringDataSourceName := "data.tidbcloud_dedicated_vpc_peering.test"
	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testUTDedicatedVPCPeeringDataSourceConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dedicatedVPCPeeringDataSourceName, "tidb_cloud_vpc_id", "tidb_cloud_vpc_id"),
				),
			},
		},
	})
}

const testUTDedicatedVPCPeeringDataSourceConfig = `
data "tidbcloud_dedicated_vpc_peering" "test" {
    vpc_peering_id = "vpcPeering-id"
}
`
