variable "email" {
  type     = string
  nullable = false
}

variable "org_role" {
  type     = string
  nullable = false
}

resource "tidbcloud_member" "example" {
  email    = var.email
  org_role = var.org_role

  # Optional project-level roles.
  # project_roles = [
  #   {
  #     rbac_role = "project:dev"
  #     scope_id  = "<project_id>"
  #   }
  # ]

  # Optional instance-level roles.
  # instance_roles = [
  #   {
  #     rbac_role = "cluster:viewer"
  #     scope_id  = "<instance_id>"
  #   }
  # ]
}
