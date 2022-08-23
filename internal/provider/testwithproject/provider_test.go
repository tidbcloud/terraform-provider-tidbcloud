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
	projectId = os.Getenv("TiDBCLOUD_PROJECTID")
	clusterId = os.Getenv("TiDBCLOUD_CLUSTERID")
)

func testAccPreCheck(t *testing.T) {
	var username, password, projectId string
	username = os.Getenv("TiDBCLOUD_USERNAME")
	password = os.Getenv("TiDBCLOUD_PASSWORD")
	projectId = os.Getenv("TiDBCLOUD_PROJECTID")
	if username == "" {
		t.Fatal("TiDBCLOUD_USERNAME must be set for acceptance tests")
	}
	if password == "" {
		t.Fatal("TiDBCLOUD_PASSWORD must be set for acceptance tests")
	}
	if projectId == "" {
		t.Fatal("TiDBCLOUD_PROJECTID must be set for acceptance tests")
	}
}
