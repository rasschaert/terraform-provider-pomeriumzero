data "pomeriumzero_policy" "allow_any_authenticated_user" {
  namespace_id = data.pomeriumzero_cluster.default.namespace_id
  name         = "Allow Any Authenticated User"
}
