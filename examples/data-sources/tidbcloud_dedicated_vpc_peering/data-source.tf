variable "vpc_peering_id" {
  type     = string
  nullable = false
}

data "tidbcloud_dedicated_vpc_peering" "example" {
  vpc_peering_id = var.vpc_peering_id
}

output "output" {
  value = data.tidbcloud_dedicated_vpc_peering.example
}
