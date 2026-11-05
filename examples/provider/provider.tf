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
# Pass the keys through environment variables.
# In CI, inject these values from your CI secret store or a secrets manager.
# export TIDBCLOUD_PUBLIC_KEY="fake_public_key"
# export TIDBCLOUD_PRIVATE_KEY="fake_private_key"
provider "tidbcloud" {
  # Set sync = true for synchronous operations on tidbcloud_cluster,
  # tidbcloud_backup, tidbcloud_import, and tidbcloud_restore.
  # Other resources are always managed synchronously.
  # sync = true
}
