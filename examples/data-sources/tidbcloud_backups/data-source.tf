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

data "tidbcloud_backups" "example" {
  page       = 1
  page_size  = 10
  project_id = "fake_id"
  cluster_id = "fake_id"
}

output "output" {
  value = data.tidbcloud_backup.example
}