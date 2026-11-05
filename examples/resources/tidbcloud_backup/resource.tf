terraform {
  required_providers {
    tidbcloud = {
      source = "tidbcloud/tidbcloud"
    }
  }
}

# Set TIDBCLOUD_PUBLIC_KEY and TIDBCLOUD_PRIVATE_KEY in the environment.
provider "tidbcloud" {}

resource "tidbcloud_backup" "example" {
  project_id  = "fake_id"
  cluster_id  = "fake_id"
  name        = "example"
  description = "create by terraform"
}
