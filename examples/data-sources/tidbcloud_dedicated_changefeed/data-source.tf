data "tidbcloud_dedicated_changefeed" "example" {
  changefeed_id = "your_changefeed_id"
}

output "output" {
  value = data.tidbcloud_dedicated_changefeed.example
}
