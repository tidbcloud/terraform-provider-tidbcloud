variable "cluster_id" {
  type     = string
  nullable = false
}

variable "user_name" {
  type     = string
  nullable = false
}

data "tidbcloud_sql_user" "example" {
  cluster_id = var.cluster_id
  user_name  = var.user_name
}

output "output" {
  value = data.tidbcloud_sql_user.example
}