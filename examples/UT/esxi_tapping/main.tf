#  Copyright (c) 2017-2026 Gigamon, Inc. All rights reserved.
#
#  Author: Gigamon Terraform Team (gigamon-terraform-team@gigamon.com)
#
#  This program is free software: you can redistribute it and/or modify
#  it under the terms of the GNU General Public License as published by
#  the Free Software Foundation, version 3 of the License.
#
#  This program is distributed in the hope that it will be useful,
#  but WITHOUT ANY WARRANTY; without even the implied warranty of
#  MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
#  GNU General Public License for more details.
#
#  You should have received a copy of the GNU General Public License
#  along with this program. If not, see <https://www.gnu.org/licenses/>

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

resource "gigamon_esxi_image" "vseries-6-12" {
  file_name = var.image_file_path
  timeout = 240
}

resource "gigamon_esxi_monitoring_domain" "terraform-md" {
  alias                           = "terraform-md"
}

# vCenter connection associated with the Monitoring Domain.
resource "gigamon_esxi_connection" "terraform-conn" {
  alias                = "terraform-conn"
  monitoring_domain_id = gigamon_esxi_monitoring_domain.terraform-md.id
  vcenter_address = var.vcenter_address
  username             = var.vcenter_username
  password             = var.vcenter_password
}

data "gigamon_esxi_datacenter" "terraform-dc" {
  connection_id    = gigamon_esxi_connection.terraform-conn.id
  data_center_name = "fm-terraform-dev"
}

data "gigamon_esxi_cluster" "terraform-cluster" {
  connection_id     = gigamon_esxi_connection.terraform-conn.id
  data_center_moref = data.gigamon_esxi_datacenter.terraform-dc.data_center_moref
  cluster_name      = "ClusterUno"
}

data "gigamon_esxi_hosts" "terraform-hosts" {
  connection_id     = gigamon_esxi_connection.terraform-conn.id
  data_center_moref = data.gigamon_esxi_datacenter.terraform-dc.data_center_moref

  cluster_moref = [
    data.gigamon_esxi_cluster.terraform-cluster.cluster_moref,
  ]
  hostname = [
    var.esxi_host_ip
  ]

}

resource "gigamon_esxi_fabric" "terraform-fabric" {
  name = "terraform-fabric"
  connection_id = gigamon_esxi_connection.terraform-conn.id
  datacenter_moref = data.gigamon_esxi_datacenter.terraform-dc.data_center_moref
  image_id = gigamon_esxi_image.vseries-6-12.id
  dynamic "host_vm_spec" {
    for_each = data.gigamon_esxi_hosts.terraform-hosts.host_details
    content {
      host_moref = host_vm_spec.value.host_moref
      host_name = host_vm_spec.value.hostname
      datastore_moref = host_vm_spec.value.datastore_cluster_moref.fm_terraform_ds
      admin_password = var.vm_admin_password
      name = "Terraform-VSeries"
      management_interface = {
        network_moref = host_vm_spec.value.network_moref.VM-Network
      }
      tunnel_interface = {
        network_moref = host_vm_spec.value.network_moref.VM-Network
      }
    }
  }
}

resource "gigamon_monitoring_session" "terraform-ms" {
  alias                = "terraform-ms"
  connection_id        = gigamon_esxi_connection.terraform-conn.id
  monitoring_domain_id = gigamon_esxi_monitoring_domain.terraform-md.id
  description          = "Terraform MS"

  depends_on = [
    gigamon_esxi_fabric.terraform-fabric,
  ]
}

resource "gigamon_trafficmap" "terraform-map" {
  name                  = "terraform-map"
  monitoring_session_id = gigamon_monitoring_session.terraform-ms.id
  comment               = "Pass all IPv4 traffic"

  rule_sets = [
    {
      rule_set_id = "1"
      priority    = 1
      aep_id      = 2

      pass_rules = [
        {
          rule_id = 1

          ip_version = {
            ip_version = "v4"
          }
        }
      ]
    }
  ]
}

resource "gigamon_app_dedup" "terraform-dedup" {
  monitoring_session_id = gigamon_monitoring_session.terraform-ms.id
  alias                 = "terraform-dedup"
}

resource "gigamon_tunnel_out" "terraform_tun" {
  alias                 = "terraform-tunnel-1"
  monitoring_session_id = gigamon_monitoring_session.terraform-ms.id
  remote_ip             = var.remote_ip_address
  vxlan {
   vni = 1
   destination_port = 1
  }
}

resource "gigamon_link" "map_to_dedup" {
  monitoring_session_id = gigamon_monitoring_session.terraform-ms.id
  source_id    = gigamon_trafficmap.terraform-map.id
  source_aep_id = 2
  dest_id = gigamon_app_dedup.terraform-dedup.id
}

resource "gigamon_link" "dedup_to_tunnel" {
  monitoring_session_id = gigamon_monitoring_session.terraform-ms.id
  source_id = gigamon_app_dedup.terraform-dedup.id
  dest_id = gigamon_tunnel_out.terraform_tun.id
}