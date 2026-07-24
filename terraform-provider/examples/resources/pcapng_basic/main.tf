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
  monitoring_domain_id = "monitoringDomin::vmware::b798affc-f440-410a-9796-f0dbfe9c5da8"
  connection_id = "connection::vmware::6df491ae-757b-4d51-91e4-79ce35277db5"
  monitoring_session_id = "monitoringSession::vmware::ddcd0b1a-c5fc-448b-ab58-4d1874287a18"
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
  remote_ip   = "9.9.9.9"
  udpgre {
    key              = 10
    destination_port = 4754
  }
  
}

resource "gigamon_tunnel_out" "vxlan_out" {
  monitoring_session_id = gigamon_monitoring_session.ms.id
  alias                 = "vxlan-out"

  description = "VXLAN egress tunnel"
  remote_ip   = "1.1.2.3"

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





