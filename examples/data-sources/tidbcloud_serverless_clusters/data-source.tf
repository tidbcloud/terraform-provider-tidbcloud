data "tidbcloud_serverless_clusters" "example" {
}

output "output" {
  value = data.tidbcloud_serverless_clusters.example
}