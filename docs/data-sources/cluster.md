---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "pomeriumzero_cluster Data Source - terraform-provider-pomeriumzero"
subcategory: ""
description: |-
  Pomerium Zero Cluster data source
---

# pomeriumzero_cluster (Data Source)

Pomerium Zero Cluster data source

## Example Usage

```terraform
data "pomeriumzero_cluster" "default" {
  name = "gifted-nightingale-1337"
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `name` (String) Cluster name

### Read-Only

- `auto_detect_ip_address` (String) Auto-detected IP address
- `created_at` (String) Creation timestamp
- `domain` (String) Cluster domain
- `fqdn` (String) Cluster FQDN
- `id` (String) Cluster identifier
- `namespace_id` (String) Cluster namespace ID
- `updated_at` (String) Last update timestamp
