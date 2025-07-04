variable "project_id" {
  type     = string
  nullable = true
}

variable "region_id" {
  type     = string
  nullable = false
}

variable "cidr_notation" {
  type     = string
  nullable = false
}

resource "tidbcloud_dedicated_network_container" "example" {
  project_id    = var.project_id
  region_id     = var.region_id
  cidr_notation = var.cidr_notation
}