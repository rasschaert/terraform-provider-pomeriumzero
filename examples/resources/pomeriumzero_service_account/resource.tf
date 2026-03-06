resource "pomeriumzero_service_account" "ci" {
  cluster_id  = pomeriumzero_cluster.default.id
  description = "CI/CD service account"
  user_id     = "ci@example.com"
}

# Access the token (sensitive) — use nonsensitive() only where appropriate
output "ci_service_account_token" {
  value     = pomeriumzero_service_account.ci.token
  sensitive = true
}

# Service account with an expiry date
resource "pomeriumzero_service_account" "ci_short_lived" {
  cluster_id  = pomeriumzero_cluster.default.id
  description = "Short-lived CI service account"
  user_id     = "ci@example.com"
  expires_at  = "2026-12-31T23:59:59Z"
}
