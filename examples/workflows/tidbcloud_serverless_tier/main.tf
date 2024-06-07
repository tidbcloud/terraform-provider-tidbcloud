variable "cluster_name" {
  type     = string
  nullable = false
}

variable "cloud_provider_region" {
  type     = string
  nullable = false
  default  = "us-east-1"
}

variable "password" {
  type      = string
  nullable  = false
  sensitive = true
}

terraform {
  required_providers {
    tidbcloud = {
      source = "tidbcloud/tidbcloud"
    }
  }
}

provider "tidbcloud" {
  # export TIDBCLOUD_PUBLIC_KEY and TIDBCLOUD_PRIVATE_KEY with the TiDB Cloud API Key
  sync = true
}

data "tidbcloud_projects" "projects" {
}

resource "tidbcloud_cluster" "example" {
  project_id     = element(data.tidbcloud_projects.projects.items, 0).id
  name           = var.cluster_name
  cluster_type   = "DEVELOPER"
  cloud_provider = "AWS"
  region         = var.cloud_provider_region
  config = {
    root_password = var.password
  }
}

output "connection_strings" {
  value = lookup(tidbcloud_cluster.example.status, "connection_strings")
}
