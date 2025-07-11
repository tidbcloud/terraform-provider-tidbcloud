---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "tidbcloud_serverless_cluster Data Source - terraform-provider-tidbcloud"
subcategory: ""
description: |-
  serverless cluster data source
---

# tidbcloud_serverless_cluster (Data Source)

serverless cluster data source

## Example Usage

```terraform
variable "cluster_id" {
  type     = string
  nullable = false
}

data "tidbcloud_serverless_cluster" "example" {
  cluster_id = var.cluster_id
}

output "output" {
  value = data.tidbcloud_serverless_cluster.example
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `cluster_id` (String) The ID of the cluster.

### Read-Only

- `annotations` (Map of String) The annotations of the cluster.
- `auto_scaling` (Attributes) The auto scaling configuration of the cluster. (see [below for nested schema](#nestedatt--auto_scaling))
- `automated_backup_policy` (Attributes) The automated backup policy of the cluster. (see [below for nested schema](#nestedatt--automated_backup_policy))
- `create_time` (String) The time the cluster was created.
- `created_by` (String) The email of the creator of the cluster.
- `display_name` (String) The display name of the cluster.
- `encryption_config` (Attributes) The encryption settings for the cluster. (see [below for nested schema](#nestedatt--encryption_config))
- `endpoints` (Attributes) The endpoints for connecting to the cluster. (see [below for nested schema](#nestedatt--endpoints))
- `labels` (Map of String) The labels of the cluster.
- `region` (Attributes) The region of the cluster. (see [below for nested schema](#nestedatt--region))
- `spending_limit` (Attributes) The spending limit of the cluster. (see [below for nested schema](#nestedatt--spending_limit))
- `state` (String) The state of the cluster.
- `update_time` (String) The time the cluster was last updated.
- `user_prefix` (String) The unique prefix in SQL user name.
- `version` (String) The version of the cluster.

<a id="nestedatt--auto_scaling"></a>
### Nested Schema for `auto_scaling`

Read-Only:

- `max_rcu` (Number) The maximum RCU (Request Capacity Unit) of the cluster.
- `min_rcu` (Number) The minimum RCU (Request Capacity Unit) of the cluster.


<a id="nestedatt--automated_backup_policy"></a>
### Nested Schema for `automated_backup_policy`

Read-Only:

- `retention_days` (Number) The number of days to retain automated backups.
- `start_time` (String) The time of day when the automated backup will start.


<a id="nestedatt--encryption_config"></a>
### Nested Schema for `encryption_config`

Read-Only:

- `enhanced_encryption_enabled` (Boolean) Whether enhanced encryption is enabled.


<a id="nestedatt--endpoints"></a>
### Nested Schema for `endpoints`

Read-Only:

- `private` (Attributes) The private endpoint for connecting to the cluster. (see [below for nested schema](#nestedatt--endpoints--private))
- `public` (Attributes) The public endpoint for connecting to the cluster. (see [below for nested schema](#nestedatt--endpoints--public))

<a id="nestedatt--endpoints--private"></a>
### Nested Schema for `endpoints.private`

Read-Only:

- `aws` (Attributes) Message for AWS PrivateLink information. (see [below for nested schema](#nestedatt--endpoints--private--aws))
- `host` (String) The host of the private endpoint.
- `port` (Number) The port of the private endpoint.

<a id="nestedatt--endpoints--private--aws"></a>
### Nested Schema for `endpoints.private.aws`

Read-Only:

- `availability_zone` (List of String) The availability zones that the service is available in.
- `service_name` (String) The AWS service name for private access.



<a id="nestedatt--endpoints--public"></a>
### Nested Schema for `endpoints.public`

Read-Only:

- `disabled` (Boolean) Whether the public endpoint is disabled.
- `host` (String) The host of the public endpoint.
- `port` (Number) The port of the public endpoint.



<a id="nestedatt--region"></a>
### Nested Schema for `region`

Read-Only:

- `cloud_provider` (String) The cloud provider of the region.
- `display_name` (String) The display name of the region.
- `name` (String) The unique name of the region.
- `region_id` (String) The ID of the region.


<a id="nestedatt--spending_limit"></a>
### Nested Schema for `spending_limit`

Read-Only:

- `monthly` (Number) Maximum monthly spending limit in USD cents.
