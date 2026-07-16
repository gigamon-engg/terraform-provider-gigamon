# Copyright (c) HashiCorp, Inc.

# 5G-SBI update example.
# Use this in a separate directory after importing the existing app.

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

resource "gigamon_app_5gsbi" "sbi5g" {
  monitoring_session_id = local.monitoring_session_id

  alias = "sbi5gAppTemplate"
  name  = "sbi5g"
  type  = "ericssonVTap"

  ip_mapping_alias                      = "nfinstance"
  http2_synthesize_tool_mtu_packet_size = 0
  # http2_synthesize_indexed_headers      = true
  # http2_synthesize_compressed_headers   = true
  # transaction_log                       = true
  # transaction_log_file_interval         = 60
  log_folder_size                       = 40960
  # stats_log                             = true
  # log_folder_loc                        = "/var/log"

  ericsson_vtap_config = {
    mode                   = "L7json"
    # eevtap_version         = "1"
    num_tcp_flows          = 512
    tcp_flow_timeout       = 7200
    num_streams_per_flow   = 8192
    # http2_request_timeout  = 15
    # http2_response_timeout = 5
    # destination_ip         = "SCP"
    # fqdn_mapping_alias     = "5g-sbiFQDN"
  }
}
