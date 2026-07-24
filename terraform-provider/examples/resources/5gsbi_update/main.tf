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


resource "gigamon_tunnel_in" "udpgre_in" {
  monitoring_session_id = local.monitoring_session_id
  alias                 = "udpgre-in"
  description           = "Ingress UDPGRE tunnel for SBI"
  ip_version            = "IPV4"
  remote_ip             = "19.1.1.99"

  udpgre {
    key              = 0
    source_port      = 10
    destination_port = 4754
  }
}

resource "gigamon_app_pcapng" "pcapng_app" {
  monitoring_session_id = local.monitoring_session_id
  alias                 = "pcapng-alias-01"
  app_mode              = "secondary"
}


resource "gigamon_app_5gsbi" "sbi_basic" {
  monitoring_session_id = local.monitoring_session_id

  alias            = "sbi5g_1"
  type             = "ericssonVTap"
  ip_mapping_alias = "sbi-nfinstance"

  http2_synthesize_tool_mtu_packet_size = 0
  http2_synthesize_indexed_headers      = false
  http2_synthesize_compressed_headers   = false
  transaction_log                       = false
  transaction_log_file_interval         = 60
  log_folder_size                       = 100
  stats_log                             = true
  log_folder_loc                        = "/var/log"

  ericsson_vtap_config = {
    mode                   = "L7json"
    eevtap_version         = "2"
    num_tcp_flows          = 256
    tcp_flow_timeout       = 1800
    num_streams_per_flow   = 8192
    http2_request_timeout  = 10
    http2_response_timeout = 5
    fqdn_mapping_alias     = "SCP"
  }
}



resource "gigamon_tunnel_out" "vxlan_out" {
  monitoring_session_id = local.monitoring_session_id
  alias                 = "vxlan-out"
  description           = "Egress VXLAN tunnel to remote tool"
  remote_ip             = "100.100.100.3"

  vxlan {
    vni              = 100
    destination_port = 4789
  }
}

resource "gigamon_link" "udpgre_in_to_pcapng" {
  monitoring_session_id = local.monitoring_session_id
  source_id             = gigamon_tunnel_in.udpgre_in.id
  dest_id               = gigamon_app_pcapng.pcapng_app.id
}

resource "gigamon_link" "pcapng_to_5gsbi" {
  monitoring_session_id = local.monitoring_session_id
  source_id             = gigamon_app_pcapng.pcapng_app.id
  dest_id               = gigamon_app_5gsbi.sbi_basic.id
}

resource "gigamon_link" "app_5gsbi_to_vxlan_out" {
  monitoring_session_id = local.monitoring_session_id
  source_id             = gigamon_app_5gsbi.sbi_basic.id
  dest_id               = gigamon_tunnel_out.vxlan_out.id
}

