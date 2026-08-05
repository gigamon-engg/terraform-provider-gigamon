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

########################################
# Monitoring Session
########################################

resource "gigamon_monitoring_session" "ms" {
  monitoring_domain_id = local.monitoring_domain_id
  connection_id        =  local.connection_id
  alias                = "pcap_ms"
  description          = "check Pcap desc"
  tapping_method       = "customerOrchestratedSource"

}




########################################
# PCAPNG
########################################

resource "gigamon_app_pcapng" "pcapNG" {
  monitoring_session_id = gigamon_monitoring_session.ms.id
  alias                 = "pcap_app1"
  app_mode              = "secondary"
  depends_on = [ gigamon_monitoring_session.ms ]
}



########################################
# TUNNEL IN
########################################

resource "gigamon_tunnel_in" "udpgre" {
  monitoring_session_id = gigamon_monitoring_session.ms.id
  alias                 = "udpgre-in"

  description = "UDPGRE ingress tunnel"
  remote_ip             = var.remote_ip_address
  udpgre {
    key              = 10
    destination_port = 4754
  }
  
}

resource "gigamon_tunnel_out" "vxlan_out" {
  monitoring_session_id = gigamon_monitoring_session.ms.id
  alias                 = "vxlan-out"

  description = "VXLAN egress tunnel"
  remote_ip             = var.remote_ip_address

  vxlan {
    vni              = 10   # 1–16777215
    destination_port = 4789   # standard VXLAN UDP port
  }
}

resource "gigamon_link" "udpgre_to_pcapng" {
  monitoring_session_id = gigamon_monitoring_session.ms.id

  source_id     = gigamon_tunnel_in.udpgre.id
  dest_id       = gigamon_app_pcapng.pcapNG.id
}

resource "gigamon_link" "pcapng_to_vxlan" {
  monitoring_session_id = gigamon_monitoring_session.ms.id

  source_id     = gigamon_app_pcapng.pcapNG.id
  dest_id       = gigamon_tunnel_out.vxlan_out.id
}





