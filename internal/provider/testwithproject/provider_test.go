package testwithproject

import (
	"os"
	"testing"

	"github.com/tidbcloud/terraform-provider-tidbcloud/internal/provider"

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
	projectId  = os.Getenv("TIDBCLOUD_PROJECT_ID")
	enableCost = os.Getenv("TIDBCLOUD_ENABLE_COST") == "true"
)

func testAccPreCheck(t *testing.T) {
	var publicKey, privateKey, projectId string
	publicKey = os.Getenv("TIDBCLOUD_PUBLIC_KEY")
	privateKey = os.Getenv("TIDBCLOUD_PRIVATE_KEY")
	projectId = os.Getenv("TIDBCLOUD_PROJECT_ID")
	if publicKey == "" {
		t.Fatal("TIDBCLOUD_PUBLIC_KEY must be set for acceptance tests")
	}
	if privateKey == "" {
		t.Fatal("TIDBCLOUD_PRIVATE_KEY must be set for acceptance tests")
	}
	if projectId == "" {
		t.Fatal("TIDBCLOUD_PROJECT_ID must be set for acceptance tests")
	}
}
