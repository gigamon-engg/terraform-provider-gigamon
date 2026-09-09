# UDPGRE tunnel validation example.
# This example uses an existing monitoring session that is imported.

terraform {
  required_providers {
    gigamon = {
      source = "local/gigamon/gigamon"
    }
  }
}

provider "gigamon" {
  fm_address  = var.fm_ip_address
  skip_verify = true
  api_token   = var.api_token
}

locals {
  monitoring_session_id = var.monitoring_session_id
}

resource "gigamon_tunnel_in" "udpgre_validation" {
  monitoring_session_id = local.monitoring_session_id
  alias                 = "udpgre-min"
  description           = "UDPGRE ingress tunnel validation example"
  ip_version            = "IPV6"
  remote_ip             = var.remote_ip_address

  udpgre {
    key              = 0
    source_port      = 5001
    destination_port = 4754
  }
}
