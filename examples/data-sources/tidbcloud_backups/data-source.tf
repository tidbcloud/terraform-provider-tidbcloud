terraform {
  required_providers {
    tidbcloud = {
      source = "tidbcloud/tidbcloud"
    }
  }
}

# Set TIDBCLOUD_PUBLIC_KEY and TIDBCLOUD_PRIVATE_KEY in the environment.
provider "tidbcloud" {}

data "tidbcloud_backups" "example" {
  page       = 1
  page_size  = 10
  project_id = "fake_id"
  cluster_id = "fake_id"
}

output "output" {
  value = data.tidbcloud_backups.example
}
