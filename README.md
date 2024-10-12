# Terraform Provider for Pomerium Zero

This Terraform provider allows you to manage resources in Pomerium Zero, a cloud-native identity-aware access proxy.

## Usage

```hcl
terraform {
  required_providers {
    pomeriumzero = {
      source  = "rasschaert/pomeriumzero"
      version = "1.0.0"
    }
  }
}

provider "pomeriumzero" {
  # Get an API token at https://console.pomerium.app/app/management/api-tokens
  api_token           = var.pomerium_zero_api_token
  organization_name = "personal"
}

```

## Data Sources

### pomeriumzero_cluster

Retrieves information about a Pomerium Zero cluster.

This can be used to reference the cluster ID for managing the cluster configuration in a `pomeriumzero_cluster_settings` resource.

It can also be used to reference the namespace ID for creating `pomeriumzero_route` and `pomeriumzero_policy` resources on this cluster.

```hcl
data "pomeriumzero_cluster" "default" {
  name = "gifted-nightingale-1337"
}

```

## Resources

### pomeriumzero_cluster_settings

Manages the settings for a Pomerium Zero cluster.

```hcl

import {
  id = data.pomeriumzero_cluster.default.id
  to = pomeriumzero_cluster_settings.default
}

resource "pomeriumzero_cluster_settings" "default" {
  address                         = ":443"
  authenticate_service_url        = "https://authenticate.gifted-nightingale-1337.pomerium.app"
  auto_apply_changesets           = true
  cookie_expire                   = "8h0m0s"
  cookie_http_only                = true
  cookie_name                     = "_pomerium"
  default_upstream_timeout        = "30s"
  dns_lookup_family               = "V4_PREFERRED"
  identity_provider               = var.pomerium_zero_identity_provider
  identity_provider_client_id     = var.pomerium_zero_identity_provider_client_id
  identity_provider_client_secret = var.pomerium_zero_identity_provider_client_secret
  identity_provider_url           = var.pomerium_zero_identity_provider_url
  log_level                       = "info"
  pass_identity_headers           = false
  proxy_log_level                 = "info"
  skip_xff_append                 = false
  timeout_idle                    = "5m0s"
  timeout_read                    = "30s"
  timeout_write                   = "0s"
  tracing_sample_rate             = 0.0001
}

```

### pomeriumzero_policy

Manages policies in Pomerium Zero.

If you want to apply a policy on one or more routes, you don't do it in the `pomeriumzero_policy` resource, but in the relevant `pomeriumzero_route` resources.

```hcl
resource "pomeriumzero_policy" "allow_foobar_group_members" {
  name         = "Allow Foobar group members"
  description  = "Member of the Foobar group are allowed."
  explanation  = "You are not a member of the Foobar group."
  remediation  = "Please contact the IT team if you think this is an error."
  enforced     = false
  namespace_id = data.pomeriumzero_cluster.default.namespace_id
  ppl = jsonencode({
    allow = {
      or = [
        {
          "claim/groups" = "foobar"
        }
      ]
    }
  })
}
```

### pomeriumzero_route

Manages routes in Pomerium Zero.

```hcl
resource "pomeriumzero_route" "foobar_tooling" {
  name           = "PoC devops"
  from           = "https://foobar-tool.example.com"
  to             = ["https://foobar-tool.examplecorp.lan/"]
  prefix         = "/home/"
  prefix_rewrite = "/home/"
  namespace_id   = data.pomeriumzero_cluster.default.namespace_id

  allow_websockets     = false
  preserve_host_header = false

  policy_ids = [
    pomeriumzero_policy.allow_foobar_group_members.id
  ]
}
```

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the [Mozilla Public License 2.0](LICENSE).
