variable "cluster_id" {
  type     = string
  nullable = false
}

variable "user_name" {
  type     = string
  nullable = false
}

variable "password" {
  type     = string
  nullable = false
}

variable "builtin_role" {
  type     = string
  nullable = false
}

variable "custom_roles" {
  type     = list(string)
  nullable = false
}

resource "tidbcloud_serverless_sql_user" "example" {
  cluster_id   = var.cluster_id
  user_name    = var.user_name
  password     = var.password
  builtin_role = var.builtin_role
  custom_roles = var.custom_roles
}