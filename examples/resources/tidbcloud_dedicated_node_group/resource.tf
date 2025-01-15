variable "cluster_id" {
  type     = string
  nullable = false
}

variable "display_name" {
  type     = string
  nullable = false
}

resource "tidbcloud_dedicated_node_group" "example_group" {
    cluster_id = var.cluster_id
    node_count = 1
    display_name = var.display_name
}