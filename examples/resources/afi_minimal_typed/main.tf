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
  fm_address  = "10.114.83.72"
  skip_verify = true
  api_token   = "eyJhbGciOiJIUzI1NiJ9.eyJ0b2tlbklkIjoiMzE3MzIwMDQwNDI4NzQyMyIsInN1YiI6IlRva2VuMSIsImlhdCI6MTc4NDAwNzc5NCwiZXhwIjoxNzg2NTk5Nzk0fQ.Z2hHcfSdCYmQGW5ZjoF6lU9ms7-aehyHLFao3JyOJow"
}



locals {
  monitoring_domain_id = "monitoringDomin::vmware::83ef84c6-4889-4594-b8f8-6824cf90dea7"
  connection_id = "connection::vmware::e8ae7cf3-1563-46f6-a490-1a6362e4a3a9"
  monitoring_session_id = "monitoringSession::vmware::ddcd0b1a-c5fc-448b-ab58-4d1874287a18"
}


resource "gigamon_raw_endpoint" "rep_main" {
  monitoring_session_id = local.monitoring_session_id

  alias       = "rep-main"
  description = "Primary raw endpoint for this session"
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
          rule_id = 4
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
      rule_id = 5
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


resource "gigamon_link" "rep_afi" {
  monitoring_session_id = local.monitoring_session_id

  # Source: traffic map in this monitoring session
  source_id     = gigamon_raw_endpoint.rep_main.id


  # Destination: tunnel out in this monitoring session
  dest_id = gigamon_traffic_map.tm_afi.id
}




