variable "cluster_id" {
  type     = string
  nullable = false
}

resource "tidbcloud_serverless_export" "example" {
  cluster_id = var.cluster_id
}