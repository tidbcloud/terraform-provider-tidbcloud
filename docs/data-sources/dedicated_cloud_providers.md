---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "tidbcloud_dedicated_cloud_providers Data Source - terraform-provider-tidbcloud"
subcategory: ""
description: |-
  dedicated cloud providers data source
---

# tidbcloud_dedicated_cloud_providers (Data Source)

dedicated cloud providers data source

## Example Usage

```terraform
variable "project_id" {
  type     = string
  nullable = true
}

data "tidbcloud_dedicated_cloud_providers" "example" {
  project_id = var.project_id
}

output "output" {
  value = data.tidbcloud_dedicated_cloud_providers.example
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Optional

- `project_id` (String) The ID of the project. If not set, it will return the cloud providers that can be selected under the default project.

### Read-Only

- `cloud_providers` (List of String) The cloud providers.