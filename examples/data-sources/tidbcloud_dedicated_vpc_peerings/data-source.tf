data "tidbcloud_dedicated_vpc_peerings" "example" {
}

output "output" {
  value = data.tidbcloud_dedicated_vpc_peerings.example
}
