# Minimal typed 5G-EVP example.
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
# ---------------------------------------------------------------------------
# EVP5G (Ericsson vTAP / 5G Cloud) application
# ---------------------------------------------------------------------------
resource "gigamon_app_evp5g" "evp5g" {
  alias                 = "evp5g-app-1"
  monitoring_session_id = local.monitoring_session_id

  rx_tunnel {
    listen_ip_address = "10.10.10.101"
    listen_port        = 4754
    rx_thread           = 8 
    dtls                 = "disable"
    dtls_key_alias       = "evp5g-dtls-key"
  }

  tx_tunnel {
    tx_remote_ip_address = "20.20.20.201"
    tx_src_port           = 4754
    tx_dst_port            = 4754
    tx_thread              = 4
    tx_src_ip_address      = ["1.1.1.1"]
  }

  time_server_config {
    primary_server   = "192.0.2.10"
    secondary_server = "192.0.2.11"
  }

  packet_ordering_config {
    num_egress_flows               = 512
    egress_flow_timeout_value      = 660
    num_buckets                    = 50
    pkts_per_bucket                = 20000
    bucket_interval                = 1
    pkt_rx_outside_bucket_interval  = "forward"
  }

  diagnostic_options {
    pct_disable = [0, 3, 7,8]
    tx_disable  = false
  }

  logging {
    packet_capture_log_level = "none"
    csv_logging_level        = "disable"
    msg_log_level             = "info"
    log_folder_loc             = "/var/log"
  }
}
