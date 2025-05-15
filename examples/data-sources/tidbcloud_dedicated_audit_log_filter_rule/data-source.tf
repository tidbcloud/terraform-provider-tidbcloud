variable "cluster_id" {
  type     = string
  nullable = false
}

variable "audit_log_filter_rule_id" {
  type     = string
  nullable = false
}

data "tidbcloud_dedicated_audit_log_filter_rule" "example" {
  cluster_id               = var.cluster_id
  audit_log_filter_rule_id = var.audit_log_filter_rule_id
}

output "output" {
  value = data.tidbcloud_dedicated_audit_log_filter_rule.example
}