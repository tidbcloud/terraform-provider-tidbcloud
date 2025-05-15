variable "network_container_id" {
  type     = string
  nullable = false
}

data "tidbcloud_dedicated_network_container" "example" {
  network_container_id = var.network_container_id
}

output "output" {
  value = data.tidbcloud_dedicated_network_container.example
}
