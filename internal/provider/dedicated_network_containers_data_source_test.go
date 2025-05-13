package provider

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	mockClient "github.com/tidbcloud/terraform-provider-tidbcloud/mock"
	"github.com/tidbcloud/terraform-provider-tidbcloud/tidbcloud"
	"github.com/tidbcloud/tidbcloud-cli/pkg/tidbcloud/v1beta1/dedicated"
)

func TestUTDedicatedNetworkContainersDataSource(t *testing.T) {
	setupTestEnv()

	ctrl := gomock.NewController(t)
	s := mockClient.NewMockTiDBCloudDedicatedClient(ctrl)
	defer HookGlobal(&NewDedicatedClient, func(publicKey string, privateKey string, dedicatedEndpoint string, userAgent string) (tidbcloud.TiDBCloudDedicatedClient, error) {
		return s, nil
	})()

	listResp := dedicated.V1beta1ListNetworkContainersResponse{}
	listResp.UnmarshalJSON([]byte(testUTV1beta1ListNetworkContainersResponse))

	s.EXPECT().ListNetworkContainers(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(&listResp, nil).AnyTimes()

	testUTDedicatedNetworkContainersDataSource(t)
}

func testUTDedicatedNetworkContainersDataSource(t *testing.T) {
	dedicatedNetworkContainerDataSourceName := "data.tidbcloud_dedicated_network_containers.test"
	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testUTDedicatedNetworkContainersDataSourceConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dedicatedNetworkContainerDataSourceName, "dedicated_network_containers.#", "0"),
				),
			},
		},
	})
}

const testUTDedicatedNetworkContainersDataSourceConfig string = `
data "tidbcloud_dedicated_network_containers" "test" {}
`

const testUTV1beta1ListNetworkContainersResponse = `
{
    "networkContainers": [
        {
            "name": "networkContainers/189000000000000",
            "networkContainerId": "18990000000000000",
            "labels": {
                "tidb.cloud/project": "310000"
            },
            "regionId": "azure-eastus2",
            "cidrNotion": "172.16.32.0/19",
            "cloudProvider": "azure",
            "state": "INACTIVE",
            "regionDisplayName": "(US) East US 2",
            "vpcId": ""
        },
        {
            "name": "networkContainers/1769500000000",
            "networkContainerId": "176000000000",
            "labels": {
                "tidb.cloud/project": "310000"
            },
            "regionId": "aws-us-west-2",
            "cidrNotion": "172.16.0.0/21",
            "cloudProvider": "aws",
            "state": "ACTIVE",
            "regionDisplayName": "Oregon (us-west-2)",
            "vpcId": "vpc-00ef3a000000000"
        }
	]
}
`
