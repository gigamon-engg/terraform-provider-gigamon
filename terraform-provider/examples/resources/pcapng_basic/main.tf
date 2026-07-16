# Copyright (c) HashiCorp, Inc.

# Minimal typed PCapNG example.
# This example uses an existing monitoring session that is imported.

terraform {
  required_providers {
    gigamon = {
      source = "local/gigamon/gigamon"
    }
  }
}

provider "gigamon" {
  fm_address  = "10.114.83.81"
  skip_verify = true
  api_token   = "eyJhbGciOiJIUzI1NiJ9.eyJ0b2tlbklkIjoiOTM1NjY5MDkzMzk1NjY4MCIsInN1YiI6IlRva2VuIiwiaWF0IjoxNzgyOTg0NTM4LCJleHAiOjE3ODU1NzY1Mzh9.iLDu2HXSVEzIJIrfTG6179Y8k8DBAf1dOSfYd_gbj7s"
}

locals {
  monitoring_session_id = "monitoringSession::vmware::f3781595-4ea8-45d8-a788-a8bd240a56b6"
}


resource "gigamon_app_pcapng" "minimal" {
  monitoring_session_id = local.monitoring_session_id

  alias = "pcapng_p1"
  name = "pcapng"

  app_mode = "secondary"
  domain_classification= true
  domain_table_alias ="pcap"
 flow_timeout = 1860
}
