variable "cluster_id" {
  type     = string
  nullable = false
}

variable "export_id" {
  type     = string
  nullable = false
}

data "tidbcloud_serverless_export" "example" {
  cluster_id = var.cluster_id
  export_id  = var.export_id
}

output "output" {
  value = data.tidbcloud_serverless_export.example
}