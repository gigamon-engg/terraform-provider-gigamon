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
  fm_address  = "10.114.50.20"
  skip_verify = true
  api_token   = "eyJhbGciOiJIUzI1NiJ9.eyJ0b2tlbklkIjoiMzA5Mjc1OTIxMTk3MzMxMiIsInN1YiI6Ik11c3RhcSIsImlhdCI6MTc4NDEwNjQyNSwiZXhwIjoxNzg2Njk4NDI1fQ.zR7zqinmMeyysIYWEN_Q5pR3wEXZiZTvrp5U_zC_6Xw"
}

locals {
  monitoring_session_id = "monitoringSession::vmware::cc05cec7-2933-411a-a846-1566b65d7c98"
}

resource "gigamon_tunnel_in" "udpgre_validation" {
  monitoring_session_id = local.monitoring_session_id
  alias                 = "udpgre-min"
  description           = "UDPGRE ingress tunnel validation example"
  ip_version            = "IPV6"
  remote_ip             = "2001:0db8:85a3:0000:0000:8a2e:0370:7334"

  udpgre {
    key              = 0
    source_port      = 5001
    destination_port = 4754
  }
}
