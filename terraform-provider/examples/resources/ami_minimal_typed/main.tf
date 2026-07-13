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
  monitoring_session_id = "monitoringSession::vmware::80c3f63d-9715-4b37-acd2-a9a398f34b6e"
}

resource "gigamon_app_ami" "minimal" {
  monitoring_session_id = local.monitoring_session_id
  alias                 = "ami-min"
  
  app_metadata = {
    flow_behavior    = "bidir"
    multi_collect    = true
    aggregate_mode   = false
    observ_domain_id = 0
    dpi_inject_limit = 30

    timeout = {
      idle = 300
    }

    match = {
      ipv6 = {
        destination = {
          prefix_min_mask = "128"
        }
        next_header = true
        source = {
          prefix_min_mask = "128"
        }
      }
    }

    exporters = [
      {
        aep_id = 5
        name   = "ami-exporter-1"

        exporter_config = {
          type         = "cef"
          max_pkt_size = 0

          app_profile_config = [
            {
              application_id = true
              family_id      = false
              tag_id         = true
              type           = "export"

              applications = [
                {
                  name = "app-1"
                  attributes = [
                    {
                      name  = "attr1"
                      value = "value1"
                    }
                  ]
                }
              ]

              counter = {
                bytes           = false
                bytes_long      = true
                packets         = false
                packets_long    = true
                inner_byte      = false
                inner_byte_long = true
              }
            }
          ]

          cef = {
            active_timeout   = 60
            inactive_timeout = 15
            record_type      = "segregated"
          }
        }
      }
    ]
  }
}

resource "gigamon_tunnel_out" "ami_udp_out" {
  monitoring_session_id = local.monitoring_session_id
  alias                 = "ami-udp-out-min"
  description           = "UDP egress tunnel for AMI minimal example"
  remote_ip             = "198.51.100.10"

  udp {
    source_port      = 50000
    destination_port = 50001
  }
}

resource "gigamon_link" "ami_to_udp" {
  monitoring_session_id = local.monitoring_session_id
  source_id             = gigamon_app_ami.minimal.id
  source_aep_id         = 5
  dest_id               = gigamon_tunnel_out.ami_udp_out.id

  depends_on = [
    gigamon_app_ami.minimal,
    gigamon_tunnel_out.ami_udp_out,
  ]
}
