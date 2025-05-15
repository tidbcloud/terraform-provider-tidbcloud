package provider

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	mockClient "github.com/tidbcloud/terraform-provider-tidbcloud/mock"
	"github.com/tidbcloud/terraform-provider-tidbcloud/tidbcloud"
	"github.com/tidbcloud/tidbcloud-cli/pkg/tidbcloud/v1beta1/dedicated"
)

func TestUTDedicatedNetworkContainerDataSource(t *testing.T) {
	setupTestEnv()

	ctrl := gomock.NewController(t)
	s := mockClient.NewMockTiDBCloudDedicatedClient(ctrl)
	defer HookGlobal(&NewDedicatedClient, func(publicKey string, privateKey string, dedicatedEndpoint string, userAgent string) (tidbcloud.TiDBCloudDedicatedClient, error) {
		return s, nil
	})()

	networkContainerId := "networkContainer-id"

	getNetworkContainerResp := dedicated.V1beta1NetworkContainer{}
	getNetworkContainerResp.UnmarshalJSON([]byte(testUTNetworkContainer(string(dedicated.DEDICATEDV1BETA1VPCPEERINGSTATE_ACTIVE))))

	s.EXPECT().GetNetworkContainer(gomock.Any(), networkContainerId).Return(&getNetworkContainerResp, nil).AnyTimes()

	testUTDedicatedNetworkContainerDataSource(t)
}

func testUTDedicatedNetworkContainerDataSource(t *testing.T) {
	dedicatedNetworkContainerDataSourceName := "data.tidbcloud_dedicated_network_container.test"
	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testUTDedicatedNetworkContainerDataSourceConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dedicatedNetworkContainerDataSourceName, "cloud_provider", "aws"),
				),
			},
		},
	})
}

const testUTDedicatedNetworkContainerDataSourceConfig = `
data "tidbcloud_dedicated_network_container" "test" {
    network_container_id = "networkContainer-id"
}
`
