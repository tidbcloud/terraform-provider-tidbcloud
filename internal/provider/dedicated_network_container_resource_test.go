package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	mockClient "github.com/tidbcloud/terraform-provider-tidbcloud/mock"
	"github.com/tidbcloud/terraform-provider-tidbcloud/tidbcloud"
	"github.com/tidbcloud/tidbcloud-cli/pkg/tidbcloud/v1beta1/dedicated"
)

func TestUTDedicatedNetworkContainerResource(t *testing.T) {
	setupTestEnv()

	ctrl := gomock.NewController(t)
	s := mockClient.NewMockTiDBCloudDedicatedClient(ctrl)
	defer HookGlobal(&NewDedicatedClient, func(publicKey string, privateKey string, dedicatedEndpoint string, userAgent string) (tidbcloud.TiDBCloudDedicatedClient, error) {
		return s, nil
	})()

	createNetworkContainerResp := dedicated.V1beta1NetworkContainer{}
	createNetworkContainerResp.UnmarshalJSON([]byte(testUTNetworkContainer(string(dedicated.V1BETA1NETWORKCONTAINERSTATE_INACTIVE))))
	getNetworkContainerResp := dedicated.V1beta1NetworkContainer{}
	getNetworkContainerResp.UnmarshalJSON([]byte(testUTNetworkContainer(string(dedicated.V1BETA1NETWORKCONTAINERSTATE_INACTIVE))))

	s.EXPECT().CreateNetworkContainer(gomock.Any(), gomock.Any()).Return(&createNetworkContainerResp, nil)
	s.EXPECT().GetNetworkContainer(gomock.Any(), gomock.Any()).Return(&getNetworkContainerResp, nil).AnyTimes()
	s.EXPECT().DeleteNetworkContainer(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	testDedicatedNetworkContainerResource(t)
}

func testDedicatedNetworkContainerResource(t *testing.T) {
	dedicatedNetworkContainerResourceName := "tidbcloud_dedicated_network_container.test"
	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read dedicated vpc peering resource
			{
				Config: testUTDedicatedNetworkContainerResourceConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dedicatedNetworkContainerResourceName, "cloud_provider", "aws"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
		ErrorCheck: func(err error) error {
			if err != nil {
				if regexp.MustCompile(`Network container can not be deleted`).MatchString(err.Error()) {
					return nil
				}
			}
			return nil
		},
	})
}

func testUTDedicatedNetworkContainerResourceConfig() string {
	return `
resource "tidbcloud_dedicated_network_container" "test" {
	region_id = "aws-ap-northeast-3"
    cidr_notation = "172.16.0.0/21"
}
`
}

func testUTNetworkContainer(state string) string {
	return fmt.Sprintf(`
{
    "name": "networkContainers/18600000000",
    "networkContainerId": "1860000000000",
    "labels": {
        "tidb.cloud/project": "300000"
    },
    "regionId": "aws-ap-northeast-3",
    "cidrNotation": "172.16.0.0/21",
    "cloudProvider": "aws",
    "state": "%s",
    "regionDisplayName": "Osaka (ap-northeast-3)",
    "vpcId": "vpc-0f43ee000000000"
}
`, state)
}
