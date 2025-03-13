variable "cluster_id" {
  type     = string
  nullable = false
}

variable "branch_id" {
  type     = string
  nullable = false
}

data "tidbcloud_serverless_branch" "example" {
  cluster_id = var.cluster_id
  branch_id  = var.branch_id
}

output "output" {
  value = data.tidbcloud_serverless_branch.example
}