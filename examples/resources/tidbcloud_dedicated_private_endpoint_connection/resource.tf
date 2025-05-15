variable "cluster_id" {
  type     = string
  nullable = false
}

variable "node_group_id" {
  type     = string
  nullable = false
}

variable "endpoint_id" {
  type     = string
  nullable = false
}

resource "tidbcloud_dedicated_private_endpoint_connection" "example" {
  cluster_id    = var.cluster_id
  node_group_id = var.node_group_id
  endpoint_id   = var.endpoint_id
}