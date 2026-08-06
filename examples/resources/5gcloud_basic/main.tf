# Copyright (c) HashiCorp, Inc.

# Minimal typed 5G Cloud example.
# This example uses an existing monitoring session that is imported.

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

# Store your existing monitoring session ID locally
# Format: monitoringSession::<platform>::<uuid>
locals {
  monitoring_session_id = var.monitoring_session_id
}

resource "gigamon_app_5gcloud" "minimal" {
  monitoring_session_id = local.monitoring_session_id

  mode = "ericssonSCPOutbound"

  rx_tunnel = [
    {
      rx_type          = "vxlan"
      listen_ipaddress = var.rx_listen_ip_address
      listen_port      = 2
      from_port        = 3
      rx_vni_id        = 4
      rx_thread        = 5
    }
  ]

  tx_tunnel = {
    tx_type             = "l2gre"
    tx_remote_ipaddress = var.tx_remote_ip_address
    tx_src_ipaddress    = var.tx_source_ip_address
    tx_src_port         = 7
    tx_dst_port         = 6
    l2gre_key           = 8
  }

  scp_config = {
    num_tcp_flows                       = 1024
    num_transaction_flows               = 2048
    tcp_flow_timeout                    = 900
    scp_transaction_timeout             = 11
    header_index                        = false
    header_compression_code             = false
    nrf_discovery_enabled               = true
    add_gigamon_header                  = false
    nf_instance_alias                   = "5gc-fn"
    fqdn_alias                          = "5gc-fqdn"
    ua_alias                            = "5gc-ua"
    min_tcp_flow_client_port            = 32765
    max_tcp_flow_client_port            = 36863
    packet_capture_log_level            = "none"
    csv_logging_log_level               = "none"
    num_scp_processing_threads          = 8
    num_tcp_flow_client_port_per_thread = 1000
    tcp_server_ports                    = 80

    http2_monitored_flows = {
      num_monitored_stream_flows = 1024
      http2_request_timeout      = 15
      http2_response_timeout     = 30
    }

    tcp_monitored_flows = {
      num_monitored_tcp_flows   = 2048
      tcp_flow_timeout          = 60
      tcp_flow_reassembly_timeout = 500
    }
  }

  tool_mtu         = 8800
  log_folder_loc   = "/var/log"
  tunnel_log_level = 2
  alias            = "cloud5g_11"



}


