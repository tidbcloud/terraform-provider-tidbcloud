terraform {
  required_providers {
    tidbcloud = {
      source = "tidbcloud/tidbcloud"
    }
  }
}

# Set TIDBCLOUD_PUBLIC_KEY and TIDBCLOUD_PRIVATE_KEY in the environment.
provider "tidbcloud" {}

data "tidbcloud_restores" "example" {
  project_id = "fake_id"
}

output "output" {
  value = data.tidbcloud_restores.example
}
