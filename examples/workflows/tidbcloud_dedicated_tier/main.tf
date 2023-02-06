variable "cluster_name" {
  type     = string
  nullable = false
}

variable "cloud_provider" {
  type     = string
  nullable = false
  default  = "AWS"
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

variable "tidb" {
  type = object({
    node_size     = string
    node_quantity = number
  })
  nullable = false
  default = {
    node_size     = "8C16G"
    node_quantity = 2
  }
}

variable "tikv" {
  type = object({
    node_size        = string
    node_quantity    = number
    storage_size_gib = number
  })
  nullable = false
  default = {
    node_size        = "8C32G"
    node_quantity    = 3
    storage_size_gib = 500
  }
}

variable "tiflash" {
  type = object({
    node_size        = string
    node_quantity    = number
    storage_size_gib = number
  })
  nullable = false
  default = {
    node_size        = "8C64G"
    node_quantity    = 1
    storage_size_gib = 500
  }
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
}

data "tidbcloud_projects" "projects" {
}

resource "tidbcloud_cluster" "example" {
  project_id     = element(data.tidbcloud_projects.projects.items, 0).id
  name           = var.cluster_name
  cluster_type   = "DEDICATED"
  cloud_provider = var.cloud_provider
  region         = var.cloud_provider_region
  config = {
    root_password = var.password
    components = {
      tidb    = var.tidb
      tikv    = var.tikv
      tiflash = var.tiflash
    }
  }
}

output "connection_strings" {
  value = lookup(tidbcloud_cluster.example.status, "connection_strings")
}