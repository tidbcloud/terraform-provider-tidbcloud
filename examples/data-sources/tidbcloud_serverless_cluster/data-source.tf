variable "cluster_id" {
  type     = string
  nullable = false
}

data "tidbcloud_serverless_cluster" "example" {
  cluster_id = var.cluster_id
}

output "output" {
  value = data.tidbcloud_serverless_cluster.example
}