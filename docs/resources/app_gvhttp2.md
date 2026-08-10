---
page_title: "GVHTTP2 Application"
subcategory: "Applications"
description: "Manage the GVHTTP2 application in Gigamon FM."
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

## Resource: `gigamon_app_gvhttp2`

The **GVHTTP2 application** processes HTTP/2 traffic in a monitoring session and forwards decoded output to an egress tunnel.

Use this resource to:

- configure listener IP/port and TLS behavior,
- select the parser mode for Casa, Nokia, Oracle, or Nokia HEP3 workflows,
- tune stream/thread/logging behavior,
- define a single egress tunnel for decoded output.

Each `gigamon_app_gvhttp2` belongs to one monitoring session.

### Important behavior and constraints

- `mode` is immutable after create. Changing it requires replacing the resource.
- `tx_tunnel` must contain exactly one element.
- `tx_tunnel.tx_type` must match `mode`:
  - `casa`, `nokia`, `oracle` require `tx_type = "vxlan"`
  - `nokiaHEP3Stream`, `nokiaHEP3Transaction` require `tx_type = "tcp"`
- TLS coupling is strict:
  - when `tls = "enable"`, both `location_certificate` and `location_private_key` are required
  - when `tls = "disable"`, both certificate path attributes must be omitted
- `pcap_enable` is not supported in HEP3 modes.

## Example Usage

### GVHTTP2 in Nokia mode with TLS enabled

```hcl
resource "gigamon_app_gvhttp2" "nokia_tls" {
  monitoring_session_id     = gigamon_monitoring_session.ms.id
  alias                     = "gvhttp2_nokia_tls"
  http2_listening_ipaddress = "192.168.20.11"
  http2_listening_port      = 4754
  mode                      = "nokia"

  tls                  = "enable"
  location_certificate = "/usr/lib/vseries-web/api/crypto/private/gvhttp2/gvhttp2.crt"
  location_private_key = "/usr/lib/vseries-web/api/crypto/private/gvhttp2/pvt_key"

  max_concurrent_stream = 50
  worker_thread         = 8
  csv_enable            = true
  pcap_enable           = true
  log_folder_loc        = "/var/log"
  log_level             = ["info", "detail"]

  tx_tunnel = [
    {
      tx_src_ipaddress = "192.168.210.54"
      tx_src_port      = 555
      tx_dst_ipaddress = "192.168.220.5"
      tx_dst_port      = 4754
      tx_type          = "vxlan"
      tx_thread        = 8
      tx_vni_id        = 100
    }
  ]
}
```

### GVHTTP2 in Nokia HEP3 stream mode

```hcl
resource "gigamon_app_gvhttp2" "nokia_hep3_stream" {
  monitoring_session_id     = gigamon_monitoring_session.ms.id
  alias                     = "gvhttp2_nokia_hep3"
  http2_listening_ipaddress = "192.168.20.21"
  http2_listening_port      = 6100
  mode                      = "nokiaHEP3Stream"

  tls = "disable"

  tx_tunnel = [
    {
      tx_src_ipaddress = "192.168.40.10"
      tx_src_port      = 6101
      tx_dst_ipaddress = "192.168.40.11"
      tx_dst_port      = 6102
      tx_type          = "tcp"
    }
  ]

  csv_enable = true
  log_level  = ["info"]
}
```

## Argument Reference

### Required

- **`monitoring_session_id`** (String)
  Monitoring session ID where the GVHTTP2 app is created.
  Changing this value forces resource replacement.

- **`alias`** (String)
  User-defined app alias.
  Allowed characters are alphanumeric, hyphen (`-`), and underscore (`_`).

- **`http2_listening_ipaddress`** (String)
  Listener bind address.
  Must be a valid IPv4 or IPv6 address.

- **`http2_listening_port`** (Number)
  Listener port.
  Valid range: `1..65535`.

- **`mode`** (String)
  Parser/correlation mode.
  Allowed values:
  - `casa`
  - `nokia`
  - `oracle`
  - `nokiaHEP3Stream`
  - `nokiaHEP3Transaction`

  This value is immutable after create.

- **`tx_tunnel`** (List of Objects)
  Exactly one TX tunnel must be provided.
  The selected mode determines valid `tx_type` and whether VXLAN-only fields are usable.

### Optional

- **`tls`** (String)
  Enables or disables TLS on the HTTP/2 listener.
  Allowed values: `enable`, `disable`.
  Default: `disable`.

- **`location_certificate`** (String)
  Certificate path for TLS termination.
  Required when `tls = "enable"`.

- **`location_private_key`** (String)
  Private key path for TLS termination.
  Required when `tls = "enable"`.

- **`max_concurrent_stream`** (Number)
  Maximum concurrent HTTP/2 streams.
  Range: `1..100`.
  Default: `100`.

- **`worker_thread`** (Number)
  Number of worker threads.
  Range: `1..16`.
  Default: `4`.

- **`csv_enable`** (Boolean)
  Enables CSV output.
  Default: `false`.

- **`pcap_enable`** (Boolean)
  Enables packet capture.
  Default: `false`.
  Not supported with `nokiaHEP3Stream` and `nokiaHEP3Transaction`.

- **`log_folder_loc`** (String)
  Folder where GVHTTP2 logs are written.
  Default: `/var/log`.

- **`log_level`** (List of String)
  Log verbosity flags.
  Allowed values: `all`, `info`, `detail`, `fullparse`.
  Default: `["info"]`.

### `tx_tunnel` Block

Provide exactly one `tx_tunnel` entry.

- **`tx_src_ipaddress`** (String)
  Source IP address for egress traffic.
  Must be a valid IPv4 or IPv6 address.

- **`tx_src_port`** (Number)
  Source port.
  Range: `1..65535`.

- **`tx_dst_ipaddress`** (String)
  Destination IP address.
  Must be a valid IPv4 or IPv6 address.

- **`tx_dst_port`** (Number)
  Destination port.
  Range: `1..65535`.

- **`tx_type`** (String)
  Egress tunnel type.
  Allowed values: `vxlan`, `tcp`.
  Must match the selected `mode`.

- **`tx_thread`** (Number)
  VXLAN transmission threads.
  Range: `1..16`.
  Default: `4`.
  Applicable only when mode is `casa`, `nokia`, or `oracle`.

- **`tx_vni_id`** (Number)
  VXLAN VNI.
  Range: `0..16777215`.
  Default: `0`.
  Applicable only when mode is `casa`, `nokia`, or `oracle`.

## Attributes Reference

In addition to all arguments above, `gigamon_app_gvhttp2` exports:

- **`id`** (String)
  Typed application ID used in other resources such as `gigamon_link`.

## Import

GVHTTP2 applications can be imported with this format:

```shell
terraform import gigamon_app_gvhttp2.example "<monitoring_session_id>::<raw_uuid>"
```
