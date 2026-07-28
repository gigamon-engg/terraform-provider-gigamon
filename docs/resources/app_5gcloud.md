---
page_title: "5G Cloud Application"
subcategory: "Applications"
description: "Manage the 5G Cloud application in Gigamon FM."
---

<!--
Copyright (c) 2017-2026 Gigamon, Inc. All rights reserved.

Author: Gigamon Terraform Team (gigamon-terraform-team@gigamon.com)

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, version 3 of the License.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>
-->

## Resource: `gigamon_app_5gcloud`

The **5G Cloud application** enables deep packet and flow visibility for 5G cloud traffic in a monitoring session.

Use this resource to:

- classify traffic by protocol and port range,
- enable or disable the app quickly,
- export flow records in NetFlow format.

Each `gigamon_app_5gcloud` belongs to a single monitoring session.

---

## Example Usage

### Minimal 5G Cloud configuration

```hcl
resource "gigamon_app_5gcloud" "app" {
  monitoring_session_id = gigamon_monitoring_session.ms.id

  enabled = true
  profile = "default"

  filter_config = {
    protocol_filter = "all"
    port_range = {
      min = 0
      max = 65535
    }
  }

  export_config = {
    format   = "netflow"
    interval = 60
  }
}
```

### Mode-based 5G Cloud configurations

The following examples show common deployment modes for `gigamon_app_5gcloud`.

### Casa Systems vTAP mode

```hcl
resource "gigamon_app_5gcloud" "casa_vtap" {
  monitoring_session_id = gigamon_monitoring_session.ms1.id
  alias                 = "casa_vtap_app"
  mode                  = "casaVtap"

  rx_tunnel = [{
    rx_type          = "vxlan"
    listen_ipaddress = "192.168.20.4"
    listen_port      = 6100
    from_port        = 49001
    rx_vni_id        = 110
    rx_thread        = 1
  }]

  tx_tunnel = {
    tx_type             = "udpgre"
    tx_remote_ipaddress = "192.168.10.5"
    tx_src_ipaddress    = "192.168.20.3"
    tx_src_port         = 6101
    tx_dst_port         = 4754
    tx_vni_id           = 50
  }

  log_folder_loc   = "/var/log"
  tunnel_log_level = 3
}
```

### Ericsson SCP mode

```hcl
resource "gigamon_app_5gcloud" "ericsson_scp_outbound" {
  monitoring_session_id = gigamon_monitoring_session.ms1.id
  alias                 = "ericsson_scp_outbound_app"
  mode                  = "ericssonSCPOutbound"

  rx_tunnel = [{
    rx_type          = "vxlan"
    listen_ipaddress = "192.168.20.4"
    listen_port      = 6100
    from_port        = 49001
    rx_vni_id        = 110
    rx_thread        = 1
  }]

  tx_tunnel = {
    tx_type             = "vxlan"
    tx_remote_ipaddress = "192.168.10.5"
    tx_src_ipaddress    = "192.168.20.3"
    tx_src_port         = 6101
    tx_dst_port         = 4754
    tx_vni_id           = 50
  }

  scp_config = {
    num_tcp_flows                       = 1024
    num_transaction_flows               = 2048
    tcp_flow_timeout                    = 900
    scp_transaction_timeout             = 10
    header_index                        = true
    header_compression_code             = false
    nrf_discovery_enabled               = true
    add_gigamon_header                  = true
    nf_instance_alias                   = "5g_cloud_network_function"
    fqdn_alias                          = "5g-apps"
    ua_alias                            = "user_agent_csv"
    min_tcp_flow_client_port            = 32768
    max_tcp_flow_client_port            = 36863
    packet_capture_log_level            = "receive"
    csv_logging_log_level               = "none"
    num_scp_processing_threads          = 16
    num_tcp_flow_client_port_per_thread = 1000
    tcp_server_ports                    = 443
  }

  log_folder_loc   = "/var/log"
  tunnel_log_level = 3
}
```

### Oracle SCP mode

```hcl
resource "gigamon_app_5gcloud" "oracle_scp" {
  monitoring_session_id = gigamon_monitoring_session.ms1.id
  alias                 = "oracle_scp_app"
  mode                  = "oracleSCP"

  rx_tunnel = [{
    rx_type          = "vxlan"
    listen_ipaddress = "192.168.20.4"
    listen_port      = 6100
    from_port        = 49001
    rx_vni_id        = 110
    rx_thread        = 1
  }]

  tx_tunnel = {
    tx_type             = "vxlan"
    tx_remote_ipaddress = "192.168.10.5"
    tx_src_ipaddress    = "192.168.20.3"
    tx_src_port         = 6101
    tx_dst_port         = 4754
    tx_vni_id           = 50
  }

  scp_config = {
    num_tcp_flows                       = 1024
    num_transaction_flows               = 2048
    tcp_flow_timeout                    = 900
    scp_transaction_timeout             = 10
    header_index                        = true
    header_compression_code             = false
    nrf_discovery_enabled               = true
    add_gigamon_header                  = true
    nf_instance_alias                   = "5g_cloud_network_function"
    fqdn_alias                          = "5g-apps"
    ua_alias                            = "user_agent_csv"
    min_tcp_flow_client_port            = 32768
    max_tcp_flow_client_port            = 36863
    packet_capture_log_level            = "receive"
    csv_logging_log_level               = "none"
    num_scp_processing_threads          = 16
    num_tcp_flow_client_port_per_thread = 1000
  }

  log_folder_loc   = "/var/log"
  tunnel_log_level = 3
}
```

### Nokia SCP mode

```hcl
resource "gigamon_app_5gcloud" "nokia_scp_inbound" {
  monitoring_session_id = gigamon_monitoring_session.ms1.id
  alias                 = "nokia_scp_inbound_app"
  mode                  = "nokiaSCPInbound"

  rx_tunnel = [{
    rx_type          = "vxlan"
    listen_ipaddress = "192.168.20.4"
    listen_port      = 6100
    from_port        = 49001
    rx_vni_id        = 110
    rx_thread        = 1
  }]

  tx_tunnel = {
    tx_type             = "vxlan"
    tx_remote_ipaddress = "192.168.10.5"
    tx_src_ipaddress    = "192.168.20.3"
    tx_src_port         = 6101
    tx_dst_port         = 4754
    tx_vni_id           = 50
  }

  scp_config = {
    num_tcp_flows                       = 1024
    num_transaction_flows               = 2048
    tcp_flow_timeout                    = 900
    scp_transaction_timeout             = 10
    header_index                        = true
    header_compression_code             = false
    nrf_discovery_enabled               = true
    add_gigamon_header                  = true
    nf_instance_alias                   = "5g_cloud_network_function"
    fqdn_alias                          = "5g-apps"
    ua_alias                            = "user_agent_csv"
    min_tcp_flow_client_port            = 32768
    max_tcp_flow_client_port            = 36863
    packet_capture_log_level            = "receive"
    csv_logging_log_level               = "none"
    num_scp_processing_threads          = 16
    num_tcp_flow_client_port_per_thread = 1000
    nokia_inbound_use_3gpp_target_api_root = true
    nokia_inbound_replace_authority        = true
  }

  log_folder_loc   = "/var/log"
  tunnel_log_level = 3
}
```

### Nokia CMM mode

```hcl
resource "gigamon_app_5gcloud" "nokia_hep3_inbound" {
  monitoring_session_id = gigamon_monitoring_session.ms1.id
  alias                 = "nokia_hep3_inbound_app"
  mode                  = "nokiaHEP3Inbound"

  rx_tunnel = [{
    rx_type          = "tcp"
    listen_ipaddress = "192.168.20.4"
    listen_port      = 6100
    rx_thread        = 1
  }]

  tx_tunnel = {
    tx_type             = "vxlan"
    tx_remote_ipaddress = "192.168.10.5"
    tx_src_ipaddress    = "192.168.20.3"
    tx_src_port         = 6101
    tx_dst_port         = 4754
    tx_vni_id           = 50
  }

  scp_config = {
    num_tcp_flows                       = 1024
    num_transaction_flows               = 2048
    tcp_flow_timeout                    = 900
    scp_transaction_timeout             = 10
    header_index                        = true
    header_compression_code             = false
    nrf_discovery_enabled               = true
    add_gigamon_header                  = true
    nf_instance_alias                   = "5g_cloud_network_function"
    fqdn_alias                          = "5g-apps"
    ua_alias                            = "user_agent_csv"
    min_tcp_flow_client_port            = 32768
    max_tcp_flow_client_port            = 36863
    packet_capture_log_level            = "receive"
    csv_logging_log_level               = "none"
    num_scp_processing_threads          = 16
    num_tcp_flow_client_port_per_thread = 1000
    tcp_server_ports                    = 443
  }

  hep3_config = {
    num_ingress_tcp_conn     = 1024
    num_egress_tcp_flows     = 2048
    ingress_tcp_timeout      = 60
    egress_tcp_flow_timeout  = 900
    num_receive_thread       = 12
    mtls                     = "disable"
    hep3_timestamp           = true
    recv_timestamp           = false
    private_key_path         = "/usr/lib/vseries-web/api/crypto/private/cloud5g/pvt_key"
    cert_file_path           = "/usr/lib/vseries-web/api/crypto/private/cloud5g/cloud5G.crt"
    num_egress_sctp_flows    = 1024
    egress_sctp_flow_timeout = 900
    service_map_table_alias  = "5g-servicemap"
  }

  log_folder_loc   = "/var/log"
  tunnel_log_level = 3
}
```

### Nokia IMS mode

```hcl
resource "gigamon_app_5gcloud" "nokia_hep3_ims" {
  monitoring_session_id = gigamon_monitoring_session.ms1.id
  alias                 = "nokia_hep3_ims_app"
  mode                  = "nokiaHEP3IMS"

  rx_tunnel = [{
    rx_type          = "tcp"
    listen_ipaddress = "192.168.20.4"
    listen_port      = 6100
    rx_thread        = 1
  }]

  tx_tunnel = {
    tx_type             = "vxlan"
    tx_remote_ipaddress = "192.168.10.5"
    tx_src_ipaddress    = "192.168.20.3"
    tx_src_port         = 6101
    tx_dst_port         = 4754
    tx_vni_id           = 50
  }

  scp_config = {
    num_tcp_flows                       = 1024
    num_transaction_flows               = 2048
    tcp_flow_timeout                    = 900
    scp_transaction_timeout             = 10
    header_index                        = true
    header_compression_code             = false
    nrf_discovery_enabled               = true
    add_gigamon_header                  = true
    nf_instance_alias                   = "5g_cloud_network_function"
    fqdn_alias                          = "5g-apps"
    ua_alias                            = "user_agent_csv"
    min_tcp_flow_client_port            = 32768
    max_tcp_flow_client_port            = 36863
    packet_capture_log_level            = "receive"
    csv_logging_log_level               = "none"
    num_scp_processing_threads          = 16
    num_tcp_flow_client_port_per_thread = 1000
    tcp_server_ports                    = 443
  }

  hep3_config = {
    num_ingress_tcp_conn     = 1024
    num_egress_tcp_flows     = 2048
    ingress_tcp_timeout      = 60
    egress_tcp_flow_timeout  = 900
    num_receive_thread       = 12
    mtls                     = "disable"
    hep3_timestamp           = true
    recv_timestamp           = false
    private_key_path         = "/usr/lib/vseries-web/api/crypto/private/cloud5g/pvt_key"
    cert_file_path           = "/usr/lib/vseries-web/api/crypto/private/cloud5g/cloud5G.crt"
    num_egress_sctp_flows    = 1024
    egress_sctp_flow_timeout = 900
    service_map_table_alias  = "5g-servicemap"
  }

  log_folder_loc   = "/var/log"
  tunnel_log_level = 3
}
```

---

## Argument Reference

### Required

- **`monitoring_session_id`** (String)
  Monitoring Session where this application is configured.
  Use the ID from your monitoring session resource (for example, `gigamon_monitoring_session.ms.id`).
  This ties the 5G Cloud app to exactly one session context.

### Optional

- **`enabled`** (Boolean)
  Enable or disable the 5G Cloud app.
  Default: `true`.
  Set this to `false` when you want to keep configuration in Terraform without actively processing traffic.

- **`profile`** (String)
  Profile selector.
  Valid values: `default`, `custom`.
  Default: `default`.
  Use `default` for baseline behavior; switch to `custom` when you need explicit filter/export tuning.

- **`filter_config`** (Block)
  Traffic filtering settings.
  Use this block to control which protocols and ports are considered by the application.

- **`export_config`** (Block)
  Export behavior settings.
  Use this block to control output format and emission cadence.

### `filter_config` block

- **`protocol_filter`** (String)
  Protocol filter type.
  Valid values: `all`, `udp`, `tcp`.
  Default: `all`.
  Choose `tcp` or `udp` to reduce noise when your use case is protocol-specific.

- **`port_range`** (Block)
  Port range sub-block.
  Use this when you need to constrain analysis/export to known service ports.

### `filter_config.port_range` block

- **`min`** (Number)
  Minimum port number.
  Range: `0-65535`.
  Default: `0`.
  This is the inclusive lower bound.

- **`max`** (Number)
  Maximum port number.
  Range: `0-65535`.
  Default: `65535`.
  This is the inclusive upper bound.

### `export_config` block

- **`format`** (String)
  Export format.
  Valid value: `netflow`.
  Default: `netflow`.
  Keep this as `netflow` unless the provider adds additional formats in a future release.

- **`interval`** (Number)
  Export interval in seconds.
  Range: `1-3600`.
  Default: `60`.
  Lower values increase export frequency; higher values reduce exporter churn.

---

## Attributes Reference

In addition to the arguments above, this resource exports:

- **`id`** (String)
  Typed application identifier in the form `app::<type>::<uuid>`.

---

## Import

Import using the monitoring session ID and the raw application UUID:

```shell
terraform import gigamon_app_5gcloud.example "<monitoring_session_id>::<raw_uuid>"
```
