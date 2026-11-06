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
# To use TIDBCLOUD_PUBLIC_KEY and TIDBCLOUD_PRIVATE_KEY from the environment,
# omit the two key arguments below.
provider "tidbcloud" {
  public_key  = "fake_public_key"
  private_key = "fake_private_key"

  # Set sync = true for synchronous operations on tidbcloud_cluster,
  # tidbcloud_backup, tidbcloud_import, and tidbcloud_restore.
  # Other resources are always managed synchronously.
  # sync = true
}
