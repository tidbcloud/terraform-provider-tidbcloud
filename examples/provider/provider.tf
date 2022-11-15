# We provide a full example in README https://github.com/tidbcloud/terraform-provider-tidbcloud#using-the-provider

terraform {
  required_providers {
    tidbcloud = {
      source  = "tidbcloud/tidbcloud"
      version = "~> 0.1.0"
    }
  }
  required_version = ">= 1.0.0"
}

provider "tidbcloud" {
  public_key  = "fake_public_key"
  private_key = "fake_private_key"
}