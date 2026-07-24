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
  monitoring_session_id = "monitoringSession::vmware::2e100e70-cf05-43e7-9278-d061eb004ab0"
}



########################################
# AFI Traffic Map for MS1
########################################
resource "gigamon_traffic_map" "tm_afi" {
  name                  = "tm-uctv-afi"
  monitoring_session_id = local.monitoring_session_id
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
      app_rules = {
        pass_rules = [
          {
        
            app_profile_config = {
              applications = [
               {name= "360-safeguard"}, 
               {name= "eset"}
              ]
              type = "filter"
            }
          }
        ]
        drop_rules = [
          {
    
            app_profile_config = {
              applications = [
               {name = "lookout-ms"}, 
               {name = "sophos-update"}
              ]
              type = "filter"
            }
          }
        ]
      }
    }
  ]
}



