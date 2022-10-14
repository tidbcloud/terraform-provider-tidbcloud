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

resource "tidbcloud_backup" "example" {
  project_id  = "fake_id"
  cluster_id  = "fake_id"
  name        = "example"
  description = "create by terraform"
}