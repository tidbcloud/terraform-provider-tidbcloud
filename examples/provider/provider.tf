# We provide a full example in Readme https://github.com/tidbcloud/terraform-provider-tidbcloud

terraform {
  required_providers {
    tidbcloud = {
      source  = "tidbcloud/tidbcloud"
      version = "~> 0.0.1"
    }
  }
  required_version = ">= 1.0.0"
}

provider "tidbcloud" {
  username = "fake_username"
  password = "fake_password"
}

