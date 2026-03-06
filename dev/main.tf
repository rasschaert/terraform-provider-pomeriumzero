terraform {
  required_providers {
    pomeriumzero = {
      source = "rasschaert/pomeriumzero"
    }
  }
}

provider "pomeriumzero" {
  api_token = var.pomerium_zero_api_token
}

variable "pomerium_zero_api_token" {
  sensitive   = true
  description = "Pomerium Zero API token. Load via: export TF_VAR_pomerium_zero_api_token=$(op read \"op://Employee/Personal Pomerium API token for terraform provider development/credential\")"
  type        = string
}

variable "cluster_name" {
  description = "Name of the Pomerium Zero cluster to use. Find it in the Pomerium Zero console."
  type        = string
}

# ---------------------------------------------------------------------------
# Data sources
# ---------------------------------------------------------------------------

data "pomeriumzero_cluster" "default" {
  name = var.cluster_name
}

# ---------------------------------------------------------------------------
# Policy
# ---------------------------------------------------------------------------

resource "pomeriumzero_policy" "allow_authenticated" {
  name         = "dev-tf-allow-authenticated"
  description  = "Allow any authenticated user (dev sandbox)"
  explanation  = "You must be authenticated."
  remediation  = ""
  enforced     = false
  namespace_id = data.pomeriumzero_cluster.default.namespace_id

  ppl = jsonencode({
    allow = {
      and = [{ authenticated_user = 1 }]
    }
  })
}

# ---------------------------------------------------------------------------
# Route
# ---------------------------------------------------------------------------

resource "pomeriumzero_route" "verify" {
  name         = "dev-tf-verify"
  from         = "https://dev-tf-verify.${data.pomeriumzero_cluster.default.fqdn}"
  to           = ["https://verify.pomerium.com"]
  namespace_id = data.pomeriumzero_cluster.default.namespace_id

  show_error_details    = true
  pass_identity_headers = true

  policy_ids = [pomeriumzero_policy.allow_authenticated.id]
}

# ---------------------------------------------------------------------------
# Service account
# ---------------------------------------------------------------------------

resource "pomeriumzero_service_account" "dev" {
  cluster_id  = data.pomeriumzero_cluster.default.id
  description = "dev-tf sandbox service account"
  user_id     = "dev-tf@example.com"
}

# ---------------------------------------------------------------------------
# Outputs
# ---------------------------------------------------------------------------

output "cluster_fqdn" {
  value = data.pomeriumzero_cluster.default.fqdn
}

output "policy_id" {
  value = pomeriumzero_policy.allow_authenticated.id
}

output "route_id" {
  value = pomeriumzero_route.verify.id
}

output "service_account_id" {
  value = pomeriumzero_service_account.dev.id
}

output "service_account_token" {
  value     = pomeriumzero_service_account.dev.token
  sensitive = true
}
