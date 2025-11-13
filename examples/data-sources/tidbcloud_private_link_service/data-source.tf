variable "cluster_id" {
  type     = string
  nullable = false
}

variable "node_group_id" {
  type     = string
  nullable = false
}

data "tidbcloud_dedicated_private_link_service" "example" {
  cluster_id    = var.cluster_id
  node_group_id = var.node_group_id
}

output "output" {
  value = data.tidbcloud_dedicated_private_link_service.example
}