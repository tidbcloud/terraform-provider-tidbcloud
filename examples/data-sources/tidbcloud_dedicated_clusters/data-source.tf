data "tidbcloud_dedicated_clusters" "example" {
}

output "output" {
  value = data.tidbcloud_dedicated_clusters.example
}