variable "project_name" {
  type     = string
  nullable = false
  default  = "default_project"
}

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
  page_size = "50"
}

locals {
  project_id = {
    value = element([for s in data.tidbcloud_projects.projects.items : s.id if s.name == var.project_name],0)
  }
}

resource "tidbcloud_cluster" "example" {
  project_id     = local.project_id.value
  name           = var.cluster_name
  cluster_type   = "DEVELOPER"
  cloud_provider = "AWS"
  region         = var.cloud_provider_region
  config = {
    root_password = var.password
  }
}
