terraform {
  required_providers {
    tidbcloud = {
      source = "tidbcloud/tidbcloud"
    }
  }
}

provider "tidbcloud" {
  public_key  = "fake_public_key"
  private_key = "fake_private_key"
}

data "tidbcloud_projects" "example" {
  page      = 1
  page_size = 10
}

output "output" {
  value = data.tidbcloud_project.example
}