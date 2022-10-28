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

resource "tidbcloud_cluster" "dedicated_tier_cluster" {
  project_id     = "fake_id"
  name           = "example1"
  cluster_type   = "DEDICATED"
  cloud_provider = "AWS"
  region         = "us-east-1"
  config = {
    root_password = "Fake_root_password1"
    components = {
      tidb = {
        node_size : "8C16G"
        node_quantity : 1
      }
      tikv = {
        node_size : "8C32G"
        storage_size_gib : 500,
        node_quantity : 3
      }
      tiflash = {
        node_size : "8C64G"
        storage_size_gib : 5000,
        node_quantity : 2
      }
    }
  }
}

resource "tidbcloud_cluster" "serverless_tier_cluster" {
  project_id     = "fake_id"
  name           = "example2"
  cluster_type   = "DEVELOPER"
  cloud_provider = "AWS"
  region         = "us-east-1"
  config = {
    root_password = "Fake_root_password1"
  }
}
