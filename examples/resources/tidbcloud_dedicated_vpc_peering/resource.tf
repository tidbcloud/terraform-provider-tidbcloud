variable "tidb_cloud_region_id" {
  type     = string
  nullable = true
}

variable "customer_region_id" {
  type     = string
  nullable = false
}

variable "customer_account_id" {
  type     = string
  nullable = false
}

variable "customer_vpc_id" {
  type     = string
  nullable = false
}

variable "customer_vpc_cidr" {
  type     = string
  nullable = false
}

resource "tidbcloud_dedicated_vpc_peering" "example" {
  tidb_cloud_region_id = var.tidb_cloud_region_id
  customer_region_id   = var.customer_region_id
  customer_account_id  = var.customer_account_id
  customer_vpc_id      = var.customer_vpc_id
  customer_vpc_cidr    = var.customer_vpc_cidr
}