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

# Get an API token at https://console.pomerium.app/app/management/api-tokens
variable "pomerium_zero_api_token" {
  sensitive   = true
  description = "Pomerium Zero API token"
  type        = string
}
