# Minimal typed AMI example.
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
  api_token   = "eyJhbGciOiJIUzI1NiJ9.eyJ0b2tlbklkIjoiNDYxMDgyNDM1NDEzOTY5NCIsInN1YiI6Imdtb2hhbiIsImlhdCI6MTc4MTUxMjMyMywiZXhwIjoxNzg0MTA0MzIzfQ.mlP_dTGCIB42Y3PjpwoH6iKdlxFjDPktDBmdl1WFDhU"
}

# Store your existing monitoring session ID locally
# Format: monitoringSession::<platform>::<uuid>
locals {
  monitoring_session_id = "monitoringSession::vmware::0ddfdd2d-2a27-4abc-ae39-3432601bcd53"
}

resource "gigamon_app_ami" "minimal" {
  monitoring_session_id = local.monitoring_session_id
  alias                 = "ami-min"

  app_metadata = {
    flow_behavior    = "bidir"
    multi_collect    = true
    aggregate_mode   = false
    observ_domain_id = 0

    timeout = {
      idle = 300
    }

    dpi_inject_limit = 30

    exporters = [
      {
        aep_id = 2
        name   = "ami-exporter-2"

        exporter_config = {
          type = "cef"

          app_profile_config = [
            {
              applications = []
              type = "export"
            }
          ]
        }
      }
    ]

  }
}
