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
  fm_address  = "10.114.83.72"
  skip_verify = true
  api_token   = "eyJhbGciOiJIUzI1NiJ9.eyJ0b2tlbklkIjoiMzE3MzIwMDQwNDI4NzQyMyIsInN1YiI6IlRva2VuMSIsImlhdCI6MTc4NDAwNzc5NCwiZXhwIjoxNzg2NTk5Nzk0fQ.Z2hHcfSdCYmQGW5ZjoF6lU9ms7-aehyHLFao3JyOJow"
}

# Store your existing monitoring session ID locally
# Format: monitoringSession::<platform>::<uuid>
locals {
  monitoring_session_id = "monitoringSession::vmware::ddcd0b1a-c5fc-448b-ab58-4d1874287a18"
}

resource "gigamon_app_5gcloud" "minimal" {
  monitoring_session_id = local.monitoring_session_id
  mode = "nokiaHEP3Inbound"

  rx_tunnel = [
    {
      rx_type          = "tcp"
      listen_ipaddress = "1.1.1.1"
      listen_port      = 1
      from_port        = 1
    }
  ]

  tx_tunnel = {
    tx_type             = "vxlan"
    tx_remote_ipaddress = "2.2.2.2"
    tx_src_ipaddress    = "3.3.3.3"
    tx_src_port         = 3
    tx_dst_port         = 2
    tx_vni_id           = 0
  }

  scp_config = {
    num_tcp_flows                       = 1024
    num_transaction_flows               = 2048
    tcp_flow_timeout                    = 900
    scp_transaction_timeout             = 10
    header_index                        = false
    header_compression_code             = false
    nrf_discovery_enabled               = true
    add_gigamon_header                  = false
    fqdn_alias                          = "5gc-fqdn"
    min_tcp_flow_client_port            = 32768
    max_tcp_flow_client_port            = 36863
    packet_capture_log_level            = "none"
    csv_logging_log_level               = "none"
    num_scp_processing_threads          = 8
    num_tcp_flow_client_port_per_thread = 1000
    tcp_server_ports                    = 1

  }

  hep3_config = {
    num_ingress_tcpconn        = 1024
    num_egress_tcp_flows       = 4096
    egress_tcp_flow_timeout    = 900
    num_receive_thread         = 8
    mtls                       = "disable"
    hep3_timestamp             = false
    recv_timestamp             = false
    mtls_key_alias             = ""
    num_egress_sctp_flows      = 1024
    egress_sctp_flow_timeout   = 900
    service_map_table_alias    = "5g-servicemap"
  }

  tool_mtu         = 8800
  log_folder_loc   = "/var/log"
  tunnel_log_level = 2
  alias            = "cloud5g"



}


