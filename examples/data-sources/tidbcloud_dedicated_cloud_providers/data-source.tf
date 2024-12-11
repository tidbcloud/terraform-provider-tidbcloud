variable "project_id" {
  type     = string
  nullable = true
}

data "tidbcloud_dedicated_cloud_providers" "example" {
  project_id = var.project_id
}

output "output" {
  value = data.tidbcloud_dedicated_cloud_providers.example
}