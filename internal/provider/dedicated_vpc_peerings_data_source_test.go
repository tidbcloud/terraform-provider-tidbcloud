package provider

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	mockClient "github.com/tidbcloud/terraform-provider-tidbcloud/mock"
	"github.com/tidbcloud/terraform-provider-tidbcloud/tidbcloud"
	"github.com/tidbcloud/tidbcloud-cli/pkg/tidbcloud/v1beta1/dedicated"
)

func TestUTDedicatedVPCPeeringsDataSource(t *testing.T) {
	setupTestEnv()

	ctrl := gomock.NewController(t)
	s := mockClient.NewMockTiDBCloudDedicatedClient(ctrl)
	defer HookGlobal(&NewDedicatedClient, func(publicKey string, privateKey string, dedicatedEndpoint string, userAgent string) (tidbcloud.TiDBCloudDedicatedClient, error) {
		return s, nil
	})()

	listResp := dedicated.Dedicatedv1beta1ListVpcPeeringsResponse{}
	listResp.UnmarshalJSON([]byte(testUTDedicatedv1beta1ListVpcPeeringsResponse))

	s.EXPECT().ListVPCPeerings(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(&listResp, nil).AnyTimes()

	testUTDedicatedVPCPeeringsDataSource(t)
}

func testUTDedicatedVPCPeeringsDataSource(t *testing.T) {
	dedicatedVPCPeeringDataSourceName := "data.tidbcloud_dedicated_vpc_peerings.test"
	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testUTDedicatedVPCPeeringsDataSourceConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dedicatedVPCPeeringDataSourceName, "dedicated_vpc_peerings.#", "0"),
				),
			},
		},
	})
}

const testUTDedicatedVPCPeeringsDataSourceConfig string = `
data "tidbcloud_dedicated_vpc_peerings" "test" {}
`

const testUTDedicatedv1beta1ListVpcPeeringsResponse = `
{
    "vpcPeerings": [
        {
            "name": "vpcPeerings/id1",
            "vpcPeeringId": "id1",
            "labels": {
                "tidb.cloud/project": "310000"
            },
            "tidbCloudRegionId": "aws-us-west-2",
            "customerRegionId": "aws-us-west-2",
            "customerAccountId": "98630000000",
            "customerVpcId": "vpc-0c0c0c0c0c0c0c0",
            "customerVpcCidr": "172.16.32.0/21",
            "tidbCloudCloudProvider": "aws",
            "tidbCloudAccountId": "00000000000",
            "tidbCloudVpcId": "vpc-00e00000000000000",
            "tidbCloudVpcCidr": "172.16.0.0/21",
            "state": "ACTIVE",
            "awsVpcPeeringConnectionId": "pcx-0e6880000000"
        },
        {
            "name": "vpcPeerings/id2",
            "vpcPeeringId": "id2",
            "labels": {
                "tidb.cloud/project": "310000"
            },
            "tidbCloudRegionId": "gcp-us-west-2",
            "customerRegionId": "gcp-us-west-2",
            "customerAccountId": "",
            "customerVpcId": "",
            "customerVpcCidr": "172.16.32.0/21",
            "tidbCloudCloudProvider": "gcp",
            "tidbCloudAccountId": "",
            "tidbCloudVpcId": "",
            "tidbCloudVpcCidr": "172.16.0.0/21",
            "state": "ACTIVE"
        }
    ]
}
`
