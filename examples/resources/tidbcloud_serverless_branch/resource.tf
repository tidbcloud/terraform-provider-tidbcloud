variable "cluster_id" {
  type     = string
  nullable = false
}

variable "display_name" {
  type     = string
  nullable = false
}

variable "parent_id" {
  type     = string
  nullable = false
}


resource "tidbcloud_serverless_cluster" "example" {
  cluster_id   = var.cluster_id
  display_name = var.display_name
  parent_id    = var.parent_id
}