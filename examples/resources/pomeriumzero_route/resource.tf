resource "pomeriumzero_route" "verify" {
  name         = "Verify"
  from         = "https://verify.${pomeriumzero_cluster.default.fqdn}"
  to           = ["https://verify.pomerium.com"]
  namespace_id = data.pomeriumzero_cluster.default.namespace_id

  allow_websockets     = false
  preserve_host_header = false

  policy_ids = [
    pomeriumzero_policy.allow_any_authenticated_user.id
  ]

  pass_identity_headers = true
}


resource "pomeriumzero_route" "foobar_tooling" {
  name = "Foobar Tooling"
  # The external URL that the Pomerium Zero cluster should listen on
  from = "https://foobar-tool.example.com"
  # A system that is only reachable by the Pomerium Zero cluster via a private network
  to = ["https://foobar-tool.examplecorp.lan/"]
  # Only match requests that begin with /home/
  prefix = "/home/"
  # Make sure to also include the prefix when forwarding the requests to the origin
  prefix_rewrite       = "/home/"
  namespace_id         = data.pomeriumzero_cluster.default.namespace_id
  allow_websockets     = false
  preserve_host_header = false
  policy_ids = [
    pomeriumzero_policy.allow_foobar_group_members.id
  ]
}

resource "pomeriumzero_route" "kubernetes_api" {
  name = "Kubernetes API"
  from = "https://k8s-api.${pomeriumzero_cluster.default.fqdn}"
  to = ["https://kubernetes.default.svc.cluster.local/"]
  namespace_id = data.pomeriumzero_cluster.default.namespace_id
  allow_websockets = false
  preserve_host_header = false
  policy_ids = [
    pomeriumzero_policy.allow_kubernetes_admins.id
  ]
  pass_identity_headers = true
  kubernetes_service_account_token = data.kubernetes_secret.k8s_api_service_account_token.data["token"]
}

# Rewrite an incoming custom header into the upstream Authorization header.
# Useful when the same Authorization slot is already needed for environment-level
# auth (e.g. a Pomerium service-account JWT) and you want to carry an application
# user token (e.g. Cognito) through to the upstream as Authorization.
resource "pomeriumzero_route" "app_backend" {
  name             = "app-backend"
  from             = "https://app-backend.example.com"
  to               = ["http://app-backend.app-backend.svc.cluster.local:3000"]
  namespace_id     = data.pomeriumzero_cluster.default.namespace_id
  allow_websockets = false
  policy_ids = [
    pomeriumzero_policy.allow_app_backend_service_account.id,
  ]
  set_request_headers = {
    Authorization = "$${pomerium.request.headers[\"X-Id-Token\"]}"
  }
}
