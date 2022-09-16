terraform {
  required_providers {
    tidbcloud = {
      source = "tidbcloud/tidbcloud"
    }
  }
}

provider "tidbcloud" {
  username = "fake_username"
  password = "fake_password"
}

data "tidbcloud_project" "example" {
  page      = 1
  page_size = 10
}

output "output" {
  value = data.tidbcloud_project.example
}