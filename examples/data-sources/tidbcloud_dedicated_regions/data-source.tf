variable "project_id" {
  type     = string
  nullable = true
}

variable "cloud_provider" {
  type     = string
  nullable = true
}

data "tidbcloud_dedicated_regions" "example" {
  project_id     = var.project_id
  cloud_provider = var.cloud_provider
}

output "output" {
  value = data.tidbcloud_dedicated_regions.example
}