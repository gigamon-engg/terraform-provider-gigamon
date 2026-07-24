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
  api_token   = "eyJhbGciOiJIUzI1NiJ9.eyJ0b2tlbklkIjoiMzA5Mjc1OTIxMTk3MzMxMiIsInN1YiI6Ik11c3RhcSIsImlhdCI6MTc4NDEwNjQyNSwiZXhwIjoxNzg2Njk4NDI1fQ.zR7zqinmMeyysIYWEN_Q5pR3wEXZiZTvrp5U_zC_6Xw"
}


locals {
  monitoring_domain_id = "monitoringDomin::vmware::83ef84c6-4889-4594-b8f8-6824cf90dea7"
  connection_id = "connection::vmware::e8ae7cf3-1563-46f6-a490-1a6362e4a3a9"
  monitoring_session_id = "monitoringSession::vmware::3c1bde50-c3d5-403f-8576-d7f60f5c5657"
}




resource "gigamon_monitoring_session" "uctv_ms" {
  alias                = "tf-uctv-ms-1"
  description          = "UCTV MS for 3PO drift tests"
  monitoring_domain_id = local.monitoring_domain_id
  connection_id        = local.connection_id

  tapping_method = "uctv"

  fast_mode          = false
  scale_unit         = 1
  distribute_traffic = true

  traffic_acquisition = {
    mirroring = {
      secure_tunnels_enabled = false
    }
    precryption = {
      secure_tunnels_enabled = false
    }
  }
}

########################################
# AFI Traffic Map for MS1
########################################
resource "gigamon_traffic_map" "tm_afi" {
  name                  = "tm-uctv-afi"
  monitoring_session_id = gigamon_monitoring_session.uctv_ms.id
  description           = "AFI validation traffic map"

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

  rule_sets = [
    {
      rule_set_id = "3"
      priority    = 4
      aep_id      = 4

      pass_rules = [
        {
          rule_id = 1

          ip_version = {
            ip_version = "v4"
          }

          ipv4_source = {
            address   = "10.10.10.0"
            cidr_mask = "24"
          }

          ipv4_destination = {
            address = "192.168.10.0"
            netmask = "255.255.255.0"
          }

          ipv4_protocol = {
            protocol_min = 6
          }
        }
      ]

      drop_rules = [
        {
          rule_id = 2
          ip_version = {
            ip_version = "v6"
          }
        }
      ]
    }
  ]
}


resource "gigamon_app_ami" "ami_main" {
  alias                 = "tf-ami-main"
  monitoring_session_id = gigamon_monitoring_session.uctv_ms.id
  description           = "AMI app validation with nested ipv6 next_header bug repro"

  app_metadata = {
    flow_behavior    = "bidir"
    multi_collect    = true
    aggregate_mode   = false
    observ_domain_id = 0
    dpi_inject_limit = 30

    timeout = {
      idle = 299
    }

    match = {
      ipv4 = {
        destination = {
          prefix_min_mask = "32"
        }
       
        source = {
          prefix_min_mask = "24"
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





resource "gigamon_tunnel_out" "udp_out" {
  monitoring_session_id = gigamon_monitoring_session.uctv_ms.id
  alias                 = "ami-udp-out"

  description = "UDP egress tunnel for AMI"
  remote_ip   = "198.51.100.10"

  udp {
    source_port      = 50000
    destination_port = 50001
  }
}


########################################
# Links for AFI -> AMI -> TEP
########################################

resource "gigamon_link" "afi_to_ami" {
  monitoring_session_id = gigamon_monitoring_session.uctv_ms.id

  source_id     = gigamon_traffic_map.tm_afi.id
  source_aep_id = 4
  dest_id       = gigamon_app_ami.ami_main.id

  depends_on = [
    gigamon_traffic_map.tm_afi,
    gigamon_app_ami.ami_main,
  ]
}

resource "gigamon_link" "ami_to_udp" {
  monitoring_session_id = gigamon_monitoring_session.uctv_ms.id
  source_aep_id         = 5
  source_id             = gigamon_app_ami.ami_main.id
  dest_id               = gigamon_tunnel_out.udp_out.id
}




