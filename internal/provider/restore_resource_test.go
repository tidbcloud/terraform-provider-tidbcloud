package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

func TestRestoreResourceSchemaMasksRootPassword(t *testing.T) {
	server := providerserver.NewProtocol6(New("test")())()
	resp, err := server.GetProviderSchema(context.Background(), &tfprotov6.GetProviderSchemaRequest{})
	if err != nil {
		t.Fatal(err)
	}
	for _, diagnostic := range resp.Diagnostics {
		if diagnostic.Severity == tfprotov6.DiagnosticSeverityError {
			t.Fatalf("unable to read provider schema: %s", diagnostic.Summary)
		}
	}

	for _, attribute := range resp.ResourceSchemas["tidbcloud_restore"].Block.Attributes {
		if attribute.Name != "config" {
			continue
		}
		for _, nested := range attribute.NestedType.Attributes {
			if nested.Name == "root_password" {
				if !nested.Sensitive {
					t.Fatal("restore root password must be sensitive in the Terraform protocol schema")
				}
				return
			}
		}
	}
	t.Fatal("restore root password missing from the Terraform protocol schema")
}
