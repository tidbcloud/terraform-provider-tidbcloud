terraform {
  required_providers {
    tidbcloud = {
      source = "tidbcloud/tidbcloud"
    }
  }
}

# Set TIDBCLOUD_PUBLIC_KEY and TIDBCLOUD_PRIVATE_KEY in the environment.
provider "tidbcloud" {}

data "tidbcloud_projects" "example" {
  page      = 1
  page_size = 10
}

output "output" {
  value = data.tidbcloud_projects.example
}
