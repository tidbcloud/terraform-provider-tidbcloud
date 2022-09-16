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

data "tidbcloud_cluster_spec" "example" {
}

output "output" {
  value = data.tidbcloud_cluster_spec.example
}