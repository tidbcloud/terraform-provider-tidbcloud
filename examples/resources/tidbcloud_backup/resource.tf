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

resource "tidbcloud_backup" "example" {
  project_id  = "fake_id"
  cluster_id  = "fake_id"
  name        = "example"
  description = "create by terraform"
}