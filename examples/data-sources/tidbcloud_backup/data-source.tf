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

data "tidbcloud_backup" "example" {
  page       = 1
  page_size  = 10
  project_id = "fake_id"
  cluster_id = "fake_id"
}

output "output" {
  value = data.tidbcloud_backup.example
}