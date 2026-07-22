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




resource "gigamon_app_ami" "ami_main" {
  alias                 = "tf-ami-main"
  monitoring_session_id = local.monitoring_session_id
  description           = "AMI app minimal validation with one exporter"

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
        aep_id = 5
        name   = "ami-exporter-1"

        exporter_config = {
          type         = "netflow"
          max_pkt_size = 1500

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

          netflow = {
            active_timeout   = 1
            inactive_timeout = 1
            record_type      = "cohesive"
            version = "ipfix"
            templateRefresh=60
          }
        }
      }
    ]
  }
}



