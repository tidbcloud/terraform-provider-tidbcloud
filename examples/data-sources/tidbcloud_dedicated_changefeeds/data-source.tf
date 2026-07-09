data "tidbcloud_dedicated_changefeeds" "example" {
  cluster_id = "your_cluster_id"
}

output "output" {
  value = data.tidbcloud_dedicated_changefeeds.example
}
