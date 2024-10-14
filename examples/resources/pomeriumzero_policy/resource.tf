resource "pomeriumzero_policy" "allow_any_authenticated_user" {
  name         = "Allow Any Authenticated User"
  description  = "Any authenticated user is allowed."
  explanation  = "You are not authenticated."
  remediation  = ""
  enforced     = false
  namespace_id = pomeriumzero_cluster.default.namespace_id
  ppl = jsonencode(
    [
      {
        allow = {
          and = [
            {
              authenticated_user = 1
            },
          ]
        }
      },
    ]
  )
}

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
