package testwithproject

import (
	"github.com/tidbcloud/terraform-provider-tidbcloud/internal/provider"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

// testAccProtoV6ProviderFactories are used to instantiate a provider during
// acceptance testing. The factory function will be invoked for every Terraform
// CLI command executed to create a provider server to which the CLI can
// reattach.
var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"tidbcloud": providerserver.NewProtocol6WithError(provider.New("test")()),
}

var (
	projectId = os.Getenv("TIDBCLOUD_PROJECTID")
	clusterId = os.Getenv("TIDBCLOUD_CLUSTERID")
)

func testAccPreCheck(t *testing.T) {
	var username, password, projectId string
	username = os.Getenv("TIDBCLOUD_USERNAME")
	password = os.Getenv("TIDBCLOUD_PASSWORD")
	projectId = os.Getenv("TIDBCLOUD_PROJECTID")
	if username == "" {
		t.Fatal("TIDBCLOUD_USERNAME must be set for acceptance tests")
	}
	if password == "" {
		t.Fatal("TIDBCLOUD_PASSWORD must be set for acceptance tests")
	}
	if projectId == "" {
		t.Fatal("TIDBCLOUD_PROJECTID must be set for acceptance tests")
	}
}
