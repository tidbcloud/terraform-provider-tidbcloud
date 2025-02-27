variable "region_id" {
  type     = string
  nullable = false
}

data "tidbcloud_dedicated_region" "example" {
  region_id = var.region_id
}

output "output" {
  value = data.tidbcloud_dedicated_region.example
}