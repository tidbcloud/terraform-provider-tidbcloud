terraform {
  required_providers {
    tidbcloud = {
      source = "tidbcloud/tidbcloud"
    }
  }
  required_version = ">= 1.0.0"
}

# Instructions for getting an API Key
# https://docs.pingcap.com/tidbcloud/api/v1beta#section/Authentication/API-Key-Management
# You can also pass the keys through environment variables:
# export TIDBCLOUD_PUBLIC_KEY = "fake_public_key"
# export TIDBCLOUD_PRIVATE_KEY = "fake_private_key"
provider "tidbcloud" {
  public_key  = "fake_public_key"
  private_key = "fake_private_key"
}

# If you want to create or update the cluster resource synchronously, set the sync to true
provider "tidbcloud" {
  public_key  = "fake_public_key"
  private_key = "fake_private_key"
  sync        = true
}
