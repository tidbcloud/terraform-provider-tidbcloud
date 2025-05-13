variable "cluster_id" {
  type     = string
  nullable = false
}

variable "node_group_id" {
  type     = string
  nullable = false
}

variable "private_endpoint_connection_id" {
  type     = string
  nullable = false
}

data "tidbcloud_dedicated_private_endpoint_connection" "example" {
  cluster_id                     = var.cluster_id
  node_group_id                  = var.node_group_id
  private_endpoint_connection_id = var.private_endpoint_connection_id
}

output "output" {
  value = data.tidbcloud_dedicated_private_endpoint_connection.example
}