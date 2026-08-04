# Complete end-to-end example showing AMI and traffic map resources.
# References an existing monitoring session by its TypedID (no management of the session itself).

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

# Create an AMI application using the monitoring session
resource "gigamon_app_ami" "example" {
  monitoring_session_id = local.monitoring_session_id
  alias                 = "ami-example"
  description           = "Example AMI application with typed configuration"

  app_metadata = {
    flow_behavior    = "bidir"
    multi_collect    = true
    aggregate_mode   = false
    observ_domain_id = 0
    dpi_inject_limit = 30

    timeout = {
      idle = 300
    }

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

resource "gigamon_tunnel_out" "ami_udp_out" {
  monitoring_session_id = local.monitoring_session_id
  alias                 = "ami-udp-out-example"
  description           = "UDP egress tunnel for AMI end-to-end example"
  remote_ip             = "198.51.100.10"

  udp = {
    source_port      = 50000
    destination_port = 50001
  }
}

resource "gigamon_link" "ami_to_udp" {
  monitoring_session_id = local.monitoring_session_id
  source_id             = gigamon_app_ami.example.id
  dest_id               = gigamon_tunnel_out.ami_udp_out.id

  depends_on = [
    gigamon_app_ami.example,
    gigamon_tunnel_out.ami_udp_out,
  ]
}

# Create a traffic map (AFI) using the monitoring session
resource "gigamon_traffic_map" "example" {
  monitoring_session_id = local.monitoring_session_id
  name                  = "afi-example"
  description           = "Example traffic map with typed ASF configuration"

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

# Outputs for verification
output "ami_id" {
  description = "The created AMI application typed ID"
  value       = gigamon_app_ami.example.id
}

output "traffic_map_id" {
  description = "The created traffic map typed ID"
  value       = gigamon_traffic_map.example.id
}
