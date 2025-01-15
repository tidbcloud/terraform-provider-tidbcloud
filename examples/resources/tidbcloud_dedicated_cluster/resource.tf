variable "name" {
  type     = string
  nullable = false
}

variable "region_id" {
  type     = string
  nullable = false
}

variable "root_password" {
  type     = string
  nullable = false
}

resource "tidbcloud_dedicated_cluster" "example" {
  name          = var.name
  region_id     = var.region_id
  port          = 4000
  root_password = var.root_password
  tidb_node_setting = {
    node_spec_key = "2C4G"
    node_count    = 1
  }
  tikv_node_setting = {
    node_spec_key   = "2C4G"
    node_count      = 3
    storage_size_gi = 10
    storage_type    = "BASIC"
  }
}