---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "tidbcloud_dedicated_node_group Resource - terraform-provider-tidbcloud"
subcategory: ""
description: |-
  dedicated node group resource
---

# tidbcloud_dedicated_node_group (Resource)

dedicated node group resource

## Example Usage

```terraform
variable "cluster_id" {
  type     = string
  nullable = false
}

variable "display_name" {
  type     = string
  nullable = false
}

resource "tidbcloud_dedicated_node_group" "example_group" {
  cluster_id   = var.cluster_id
  node_count   = 1
  display_name = var.display_name
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `cluster_id` (String) The ID of the cluster.
- `display_name` (String) The display name of the node group.
- `node_count` (Number) The count of the nodes in the group.

### Optional

- `public_endpoint_setting` (Attributes) Settings for public endpoint. (see [below for nested schema](#nestedatt--public_endpoint_setting))
- `tiproxy_setting` (Attributes) Settings for TiProxy nodes. (see [below for nested schema](#nestedatt--tiproxy_setting))

### Read-Only

- `endpoints` (List of Object) The endpoints of the node group. (see [below for nested schema](#nestedatt--endpoints))
- `is_default_group` (Boolean) Whether the node group is the default group.
- `node_group_id` (String) The ID of the node group.
- `node_spec_display_name` (String) The display name of the node spec.
- `node_spec_key` (String) The key of the node spec.
- `state` (String) The state of the node group.

<a id="nestedatt--public_endpoint_setting"></a>
### Nested Schema for `public_endpoint_setting`

Optional:

- `enabled` (Boolean) Whether public endpoint is enabled.
- `ip_access_list` (List of Object) IP access list for the public endpoint. (see [below for nested schema](#nestedatt--public_endpoint_setting--ip_access_list))

<a id="nestedatt--public_endpoint_setting--ip_access_list"></a>
### Nested Schema for `public_endpoint_setting.ip_access_list`

Optional:

- `cidr_notation` (String)
- `description` (String)



<a id="nestedatt--tiproxy_setting"></a>
### Nested Schema for `tiproxy_setting`

Optional:

- `node_count` (Number) The number of TiProxy nodes.
- `type` (String) The type of TiProxy nodes.- SMALL: Low performance instance with 2 vCPUs and 4 GiB memory. Max QPS: 30, Max Data Traffic: 90 MiB/s.- LARGE: High performance instance with 8 vCPUs and 16 GiB memory. Max QPS: 100, Max Data Traffic: 300 MiB/s.


<a id="nestedatt--endpoints"></a>
### Nested Schema for `endpoints`

Read-Only:

- `connection_type` (String)
- `host` (String)
- `port` (Number)
