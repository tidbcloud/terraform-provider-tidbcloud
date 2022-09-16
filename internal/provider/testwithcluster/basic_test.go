package testwithcluster

import (
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/tidbcloud/terraform-provider-tidbcloud/internal/provider"
	"os"
	"testing"
)

var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"tidbcloud": providerserver.NewProtocol6WithError(provider.New("test")()),
}

var (
	projectId = os.Getenv("TIDBCLOUD_PROJECTID")
	clusterId = os.Getenv("TIDBCLOUD_CLUSTERID")
)

func testAccPreCheck(t *testing.T) {
	var username, password, projectId, clusterId string
	username = os.Getenv("TIDBCLOUD_USERNAME")
	password = os.Getenv("TIDBCLOUD_PASSWORD")
	projectId = os.Getenv("TIDBCLOUD_PROJECTID")
	clusterId = os.Getenv("TIDBCLOUD_CLUSTERID")
	if username == "" {
		t.Fatal("TIDBCLOUD_USERNAME must be set for acceptance tests")
	}
	if password == "" {
		t.Fatal("TIDBCLOUD_PASSWORD must be set for acceptance tests")
	}
	if projectId == "" {
		t.Fatal("TIDBCLOUD_PROJECTID must be set for acceptance tests")
	}
	if clusterId == "" {
		t.Fatal("TIDBCLOUD_CLUSTERID must be set for acceptance tests")
	}
}
