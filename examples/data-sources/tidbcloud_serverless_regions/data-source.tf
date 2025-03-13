data "tidbcloud_serverless_regions" "example" {
}

output "output" {
  value = data.tidbcloud_serverless_regions.example
}