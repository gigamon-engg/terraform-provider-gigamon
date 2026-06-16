# Minimal typed AFI example using traffic map ASF block.
# Replace IDs/credentials for your environment.

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
  api_token   = "eyJhbGciOiJIUzI1NiJ9.eyJ0b2tlbklkIjoiNDYxMDgyNDM1NDEzOTY5NCIsInN1YiI6Imdtb2hhbiIsImlhdCI6MTc4MTUxMjMyMywiZXhwIjoxNzg0MTA0MzIzfQ.mlP_dTGCIB42Y3PjpwoH6iKdlxFjDPktDBmdl1WFDhU"
}

# Store your existing monitoring session ID locally
# Format: monitoringSession::<platform>::<uuid>
locals {
  monitoring_session_id = "monitoringSession::vmware::68545039-6cf1-4462-a27b-904d2e9b6d27"
}

resource "gigamon_traffic_map" "afi_minimal" {
  monitoring_session_id = local.monitoring_session_id
  name                  = "map-afi-min"

  rule_sets = [
    {
      rule_set_id = "1"
      aep_id      = 2
      priority    = 1

      pass_rules = [
        {
          rule_id = 1
          ip_version = {
            ip_version = "v4"
          }
        }
      ]
    }
  ]

  asf = {
    asf_profile_config = {
      session_fields = [
        {
          pos  = 2
          type = "fiveTuple"
        }
      ]

      timeout      = 15
      packet_count = 30
      bidi         = true

      buffering = {
        enabled                   = true
        protocol                  = "tcpUdp"
        buffer_count_before_match = 20
      }
    }
  }
}
