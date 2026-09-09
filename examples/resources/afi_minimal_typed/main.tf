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
  fm_address  = var.fm_ip_address
  skip_verify = true
  api_token   = var.api_token
}



locals {
  monitoring_domain_id = var.monitoring_domain_id
  connection_id = var.connection_id
  monitoring_session_id = var.monitoring_session_id
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




