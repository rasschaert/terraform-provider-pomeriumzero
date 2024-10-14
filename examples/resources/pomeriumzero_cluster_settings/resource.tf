resource "pomeriumzero_cluster_settings" "default" {
  address                         = ":443"
  auto_apply_changesets           = true
  cookie_expire                   = "14h0m0s"
  cookie_http_only                = true
  cookie_name                     = "_pomerium"
  default_upstream_timeout        = "30s"
  dns_lookup_family               = "V4_PREFERRED"
  authenticate_service_url        = "https://authenticate.${pomeriumzero_cluster.default.fqdn}"
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

variable "pomerium_zero_identity_provider" {
  sensitive   = false
  description = "Pomerium Zero Identity Provider"
  type        = string
}

variable "pomerium_zero_identity_provider_url" {
  sensitive   = false
  description = "Pomerium Zero Identity Provider URL"
  type        = string
}

variable "pomerium_zero_identity_provider_client_id" {
  sensitive   = false
  description = "Pomerium Zero Identity Provider Client ID"
  type        = string
}

variable "pomerium_zero_identity_provider_client_secret" {
  sensitive   = true
  description = "Pomerium Zero Identity Provider Client Secret"
  type        = string
}
