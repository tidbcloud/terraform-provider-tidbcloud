variable "display_name" {
  type     = string
  nullable = false
}

variable "region_name" {
  type     = string
  nullable = false
}

variable "high_availability_type" {
  type     = string
  nullable = false
}

resource "tidbcloud_serverless_cluster" "example" {
  display_name = var.display_name
  region = {
    name = var.region_name
  }
  high_availability_type = var.high_availability_type
}