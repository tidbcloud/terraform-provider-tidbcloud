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

resource "tidbcloud_restore" "example" {
  project_id = "fake_id"
  backup_id  = "fake_id"
  name       = "example"
  config = {
    root_password = "Fake_root_password1"
    port          = 4000
    components = {
      tidb = {
        node_size : "8C16G"
        node_quantity : 1
      }
      tikv = {
        node_size : "8C32G"
        storage_size_gib : 500
        node_quantity : 3
      }
      tiflash = {
        node_size : "8C64G"
        storage_size_gib : 2048
        node_quantity : 2
      }
    }
  }
}