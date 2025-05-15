variable "cluster_id" {
  type     = string
  nullable = false
}

variable "user_expr" {
  type     = string
  nullable = false
}

variable "db_expr" {
  type     = string
  nullable = false
}

variable "table_expr" {
  type     = string
  nullable = false
}

variable "access_type_list" {
  type     = list(string)
  nullable = false
}

resource "tidbcloud_dedicated_audit_log_filter_rule" "example" {
  cluster_id       = var.cluster_id
  user_expr        = var.user_expr
  db_expr          = var.db_expr
  table_expr       = var.table_expr
  access_type_list = var.access_type_list
}