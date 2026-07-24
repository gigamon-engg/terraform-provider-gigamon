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
  monitoring_domain_id = "monitoringDomain::vmware::ddcd0b1a-c5fc-448b-ab58-4d1874287a18"
  connection_id = "connection::vmware::6df491ae-757b-4d51-91e4-79ce35277db5"
}

########################################
# Monitoring Session
########################################

resource "gigamon_monitoring_session" "ms" {
  monitoring_domain_id = local.monitoring_domain_id
  connection_id        = local.connection_id
  alias                = "ms_pcap1"
  description          = "test"
  tapping_method       = "customerOrchestratedSource"
}



########################################
# TUNNEL IN
########################################

resource "gigamon_tunnel_in" "udpgre" {
  monitoring_session_id = gigamon_monitoring_session.ms.id
  alias                 = "udpgre-in"

  description = "UDPGRE ingress tunnel"
  remote_ip   = var.tunnel_remote_ip

  udpgre {
    key              = 10
    destination_port = 4754
  }
}

########################################
# PCAPNG
########################################

resource "gigamon_app_pcapng" "pcapNG" {
  monitoring_session_id = gigamon_monitoring_session.ms.id
  alias                 = var.pcapng_alias
  app_mode              = var.pcapng_app_mode
  domain_classification = var.pcapng_domain_classification
  domain_table_alias    = var.pcapng_domain_table_alias
  flow_timeout          = var.pcapng_flow_timeout
  depends_on = [ gigamon_monitoring_session.ms ]
}



########################################
# Traffic Map
########################################

# resource "gigamon_traffic_map" "flow_map" {
#   name                  = var.flow_map_name
#   monitoring_session_id = gigamon_monitoring_session.ms.id

#   rule_sets = [
#     {
#       rule_set_id = var.flow_map_rule_set_id
#       priority    = var.flow_map_priority
#       aep_id      = var.flow_map_aep_id

#       pass_rules = [
#         {
#           rule_id = var.flow_map_rule_id
#           ip_version = {
#             ip_version         = var.flow_map_ip_version
#             nested_level_count = 0
#           },
#           vm_name_source = {
#             vm_name_prefix = "ubuntu-target-201-43"
#           }
#         }
#       ]
#     }
#   ]
# }
########################################
# AFI Traffic Map for MS1
########################################
# resource "gigamon_traffic_map" "flow_map" {
#   name                  = "tm-esxi-afi"
#   monitoring_session_id = gigamon_monitoring_session.ms.id
#   description           = "AFI validation traffic map"

#   asf = {
#     asf_profile_config = {
#       session_fields = [
#         {
#           pos  = 2
#           type = "fiveTuple"
#         }
#       ]

#       timeout      = 15
#       packet_count = 40
#       bidi         = true

#       buffering = {
#         enabled                   = true
#         protocol                  = "tcpUdpSctp"
#         buffer_count_before_match = 20
#       }
#     }
#   }

#   rule_sets = [ 
#     {
#       rule_set_id = var.flow_map_rule_set_id
#       priority    = var.flow_map_priority
#       aep_id      = var.flow_map_aep_id

#       pass_rules = [
#         {
#           rule_id = var.flow_map_rule_id
#           ip_version = {
#             ip_version         = var.flow_map_ip_version
#             nested_level_count = 0
#           },
#           vm_name_source = {
#             vm_name_prefix = "ubuntu-target-201-43"
#           }
#         }
#       ]

#       drop_rules = [
#           {
#             rule_id = 2
#             ip_version = {
#               ip_version = "v6"
#             }
#           }
#         ]
#     }
#   ]
# }

# resource "gigamon_app_viz" "minimal" {
#   monitoring_session_id = gigamon_monitoring_session.ms.id

#   alias         = "app_viz1"
#   description   = ""
#   action        = true
#   mgmt_interface = "internal"

#   exporter_config = {
#     monitor = {
#       timeout = 300
#     }
#   }
# }

# resource "gigamon_app_ami" "ami_main" {
#   alias                 = "tf-ami-main"
#   monitoring_session_id = gigamon_monitoring_session.ms.id
#   description           = "AMI app minimal validation with one exporter"

#   app_metadata = {
#     flow_behavior    = "bidir"
#     multi_collect    = false
#     aggregate_mode   = true
#     observ_domain_id = 0
#     dpi_inject_limit = 20

#     timeout = {
#       idle = 300
#     }

#     exporters = [
#       {
#         aep_id = 3
#         name   = "ami-exporter-1"

#         exporter_config = {
#           type         = var.ami_exporter_type
#           max_pkt_size = 0

#           app_profile_config = [
#             {
#               application_id = true
#               family_id      = false
#               tag_id         = true
#               type           = "export"

#               applications = [
#                 {
#                   name = "app-1"
#                   attributes = [
#                     {
#                       name  = "attr1"
#                       value = "value1"
#                     }
#                   ]
#                 }
#               ]

#               counter = {
#                 bytes           = false
#                 bytes_long      = true
#                 packets         = false
#                 packets_long    = true
#                 inner_byte      = false
#                 inner_byte_long = true
#               }
#             }
#           ]

#           cef = var.ami_exporter_type == "cef" ? {
#             active_timeout   = var.ami_cef_active_timeout
#             inactive_timeout = var.ami_cef_inactive_timeout
#             record_type      = var.ami_record_type
#           } : null

#           netflow = var.ami_exporter_type == "netflow" ? {
#             active_timeout   = var.ami_netflow_active_timeout
#             inactive_timeout = var.ami_netflow_inactive_timeout
#             record_type      = var.ami_record_type
#             version          = var.ami_netflow_version
#             template_refresh = var.ami_netflow_template_refresh
#           } : null
#         }
#       }
#     ]
#   }
# }

########################################
# Slicing App
########################################


# resource "gigamon_app_slicing" "slicing" {
#   monitoring_session_id = gigamon_monitoring_session.ms.id
#   alias                 = var.slicing_app_alias
#   protocol              = var.slicing_protocol
#   offset                = var.slicing_offset
# }

# resource "gigamon_app_dedup" "dedup" {
#   alias                 = "dedup-main"
#   monitoring_session_id = gigamon_monitoring_session.ms.id
#   description           = "Deduplication app for production monitoring flow"
# }
########################################
# Tunnel Out - L2GRE
########################################

# resource "gigamon_tunnel_out" "l2gre_out" {
#   monitoring_session_id = gigamon_monitoring_session.ms.id
#   alias                 = var.tunnel_alias
#   remote_ip             = var.tunnel_remote_ip

#   l2gre {
#     key = var.tunnel_l2gre_key
#   }
# }

resource "gigamon_tunnel_out" "vxlan_out" {
  monitoring_session_id = gigamon_monitoring_session.ms.id
  alias                 = "vxlan-out"

  description = "VXLAN egress tunnel"
  remote_ip   = var.tunnel_remote_ip

  vxlan {
    vni              = 10   # 1–16777215
    destination_port = 4789   # standard VXLAN UDP port
  }
}
########################################
# Links
########################################

# resource "gigamon_link" "afi_to_ami" {
#   monitoring_session_id = gigamon_monitoring_session.ms.id

#   source_id     = gigamon_traffic_map.flow_map.id
#   source_aep_id = 2
#   dest_id       = gigamon_app_ami.ami_main.id

#   depends_on = [
#     gigamon_traffic_map.flow_map,
#     gigamon_app_ami.ami_main,
#   ]
# }

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

# resource "gigamon_link" "afi_to_appviz" {
#   monitoring_session_id = gigamon_monitoring_session.ms.id

#   source_id     = gigamon_traffic_map.flow_map.id
#   source_aep_id = 2
#   dest_id       = gigamon_app_viz.minimal.id

#   depends_on = [
#     gigamon_traffic_map.flow_map,
#     gigamon_app_viz.minimal,
#   ]
# }

# resource "gigamon_tunnel_out" "udp_out" {
#   monitoring_session_id = gigamon_monitoring_session.ms.id
#   alias                 = "ami-udp-out"

#   description = "UDP egress tunnel for AMI"
#   remote_ip   = var.tunnel_remote_ip

#   udp {
#     source_port      = 50000
#     destination_port = 50001
#   }
# }

########################################
# Tunnel Out - L2GRE
########################################

# resource "gigamon_tunnel_out" "l2gre_out" {
#   monitoring_session_id = gigamon_monitoring_session.ms.id
#   alias                 = var.tunnel_alias
#   remote_ip             = var.tunnel_remote_ip

#   l2gre {
#     key = var.tunnel_l2gre_key
#   }
# }

# resource "gigamon_link" "ami_to_udp" {
#   monitoring_session_id = gigamon_monitoring_session.ms.id
#   source_id = gigamon_app_ami.ami_main.id
#   source_aep_id = 3
#   dest_id   = gigamon_tunnel_out.udp_out.id
# }

# resource "gigamon_link" "ami_to_l2gre" {
#   monitoring_session_id = gigamon_monitoring_session.ms.id
#   source_id             = gigamon_app_ami.ami_main.id
#   dest_id               = gigamon_tunnel_out.l2gre_out.id
# }

resource "gigamon_endpoint_iface_mapping" "iface_mapping" {
    monitoring_session_id = gigamon_monitoring_session.ms.id
    vseries_node_ids = values(gigamon_esxi_fabric.fabric.host_vm_spec)[*].vseries_node_id
    mapping {
        endpoint_id = gigamon_tunnel_in.udpgre.id
        iface   = "ens192"
    }
    mapping {
        endpoint_id = gigamon_tunnel_out.vxlan_out.id
        iface   = "ens192"
    }
}