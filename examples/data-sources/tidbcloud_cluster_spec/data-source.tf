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

data "tidbcloud_cluster_spec" "example" {
}

output "output" {
  value = data.tidbcloud_cluster_spec.example
}