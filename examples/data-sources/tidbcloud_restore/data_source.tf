terraform {
  required_providers {
    tidbcloud = {
      source = "hashicorp/tidbcloud"
    }
  }
}

provider "tidbcloud" {
  username = "fake_username"
  password = "fake_password"
}

data "tidbcloud_restore" "example" {
  project_id = "fake_id"
}

output "output" {
  value = data.tidbcloud_restore.example
}