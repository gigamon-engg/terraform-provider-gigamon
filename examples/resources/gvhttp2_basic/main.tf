# Minimal typed GVHTTP2 example.
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
# -----------------------------------------------------------------------------
# GVHTTP2
# -----------------------------------------------------------------------------
resource "gigamon_app_gvhttp2" "nokia_tls" {
  monitoring_session_id     = local.monitoring_session_id
  alias                     = "gvhttp2_nokia_tls"
  http2_listening_ipaddress = "192.168.20.11"
  http2_listening_port      = 100
  mode                      = "nokia"
  tls                       = "enable"
  location_certificate      = "/usr/lib/vseries-web/api/crypto/private/gvhttp2/gvhttp2.crt"
  location_private_key      = "/usr/lib/vseries-web/api/crypto/private/gvhttp2/pvt_key"
  max_concurrent_stream     = 50
  worker_thread             = 8
  csv_enable                = true
  pcap_enable               = true
  log_folder_loc            = "/var/log"
  log_level                 = ["info", "detail", "fullparse"]

  tx_tunnel = [
    {
      tx_src_ipaddress = "192.168.210.54"
      tx_src_port      = 555
      tx_dst_ipaddress = "192.168.220.5"
      tx_dst_port      = 1
      tx_type          = "vxlan"
      tx_thread        = 8
      tx_vni_id        = 1
    }
  ]
}