package provider

import (
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"os"
	"testing"
)

// testAccProtoV6ProviderFactories are used to instantiate a provider during
// acceptance testing. The factory function will be invoked for every Terraform
// CLI command executed to create a provider server to which the CLI can
// reattach.
var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"tidbcloud": providerserver.NewProtocol6WithError(New("test")()),
}

func testAccPreCheck(t *testing.T) {
	var publicKey, privateKey, projectId string
	publicKey = os.Getenv(TiDBCloudPublicKey)
	privateKey = os.Getenv(TiDBCloudPrivateKey)
	projectId = os.Getenv(TiDBCloudProjectID)
	if publicKey == "" {
		t.Fatalf("%s must be set for acceptance tests", TiDBCloudPublicKey)
	}
	if privateKey == "" {
		t.Fatalf("%s must be set for acceptance tests", TiDBCloudPrivateKey)
	}
	if projectId == "" {
		t.Fatalf("%s must be set for acceptance tests", TiDBCloudProjectID)
	}
}
