package testwithcluster

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/tidbcloud/terraform-provider-tidbcloud/internal/provider"
)

var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"tidbcloud": providerserver.NewProtocol6WithError(provider.New("test")()),
}

var (
	projectId = os.Getenv("TIDBCLOUD_PROJECT_ID")
	clusterId = os.Getenv("TIDBCLOUD_CLUSTER_ID")
)

func testAccPreCheck(t *testing.T) {
	var publicKey, privateKey, projectId, clusterId string
	publicKey = os.Getenv("TIDBCLOUD_PUBLIC_KEY")
	privateKey = os.Getenv("TIDBCLOUD_PRIVATE_KEY")
	projectId = os.Getenv("TIDBCLOUD_PROJECT_ID")
	clusterId = os.Getenv("TIDBCLOUD_CLUSTER_ID")
	if publicKey == "" {
		t.Fatal("TIDBCLOUD_PUBLIC_KEY must be set for acceptance tests")
	}
	if privateKey == "" {
		t.Fatal("TIDBCLOUD_PRIVATE_KEY must be set for acceptance tests")
	}
	if projectId == "" {
		t.Fatal("TIDBCLOUD_PROJECT_ID must be set for acceptance tests")
	}
	if clusterId == "" {
		t.Fatal("TIDBCLOUD_CLUSTER_ID must be set for acceptance tests")
	}
}
