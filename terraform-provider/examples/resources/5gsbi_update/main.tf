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
  fm_address  = "10.114.83.72"
  skip_verify = true
  api_token   = "eyJhbGciOiJIUzI1NiJ9.eyJ0b2tlbklkIjoiMzE3MzIwMDQwNDI4NzQyMyIsInN1YiI6IlRva2VuMSIsImlhdCI6MTc4NDAwNzc5NCwiZXhwIjoxNzg2NTk5Nzk0fQ.Z2hHcfSdCYmQGW5ZjoF6lU9ms7-aehyHLFao3JyOJow"
}

locals {
  monitoring_session_id = "monitoringSession::vmware::ddcd0b1a-c5fc-448b-ab58-4d1874287a18"
}

resource "gigamon_app_5gsbi" "sbi5g" {
  monitoring_session_id = local.monitoring_session_id

  alias = "sbi5gAppTemplate"
  #name  = "sbi5g"
  type  = "ericssonVTap"

  ip_mapping_alias                      = "sbi-nfinstance"
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
