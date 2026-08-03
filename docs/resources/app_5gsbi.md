---
page_title: "5G-SBI Application"
subcategory: "Applications"
description: "Manage the 5G-SBI application in Gigamon FM."
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

## Resource: `gigamon_app_5gsbi`

The **5G-SBI application** processes 5G Service-Based Interface traffic for Ericsson vTap-based workflows.

Use this resource to:

- define 5G-SBI app identity and IP mapping alias,
- configure HTTP/2 synthesis behavior and logging options,
- tune Ericsson vTap flow/stream settings.

Each `gigamon_app_5gsbi` belongs to a single monitoring session.

---

## Example Usage

### Minimal 5G-SBI configuration

```hcl
resource "gigamon_app_5gsbi" "app" {
  monitoring_session_id = gigamon_monitoring_session.ms.id
  alias                 = "sbi5g-min"
  type                  = "ericssonVTap"
  ip_mapping_alias      = "nf-map-min"

  # Valid values: 0 or 1200..9200
  http2_synthesize_tool_mtu_packet_size = 0

  ericsson_vtap_config = {
    num_tcp_flows       = 1024
    tcp_flow_timeout    = 300
    num_streams_per_flow = 4096
  }
}
```

### Practical 5G-SBI configuration

```hcl
resource "gigamon_app_5gsbi" "app" {
  monitoring_session_id = gigamon_monitoring_session.ms.id
  alias                 = "sbi5g-prod"
  type                  = "ericssonVTap"
  ip_mapping_alias      = "nf-map-prod"

  http2_synthesize_tool_mtu_packet_size = 1500
  http2_synthesize_indexed_headers      = true
  http2_synthesize_compressed_headers   = true

  transaction_log               = true
  transaction_log_file_interval = 60
  log_folder_size               = 1024
  stats_log                     = true
  log_folder_loc                = "/var/log/gigamon/sbi5g"

  ericsson_vtap_config = {
    mode                   = "L7json"
    eevtap_version         = "2"
    num_tcp_flows          = 4096
    tcp_flow_timeout       = 600
    num_streams_per_flow   = 16384
    http2_request_timeout  = 20
    http2_response_timeout = 5
    destination_ip         = "SCP"
    fqdn_mapping_alias     = "fqdn-map-prod"
  }
}
```

### Topology integration example

```hcl
resource "gigamon_link" "map_to_sbi5g" {
  monitoring_session_id = gigamon_monitoring_session.ms.id
  source_id             = gigamon_traffic_map.web.id
  source_aep_id         = 2
  dest_id               = gigamon_app_5gsbi.app.id
}
```

Topology notes:

- `source_aep_id` must match the source traffic map rule set `aep_id`.
- `dest_id` should reference the typed ID exported by `gigamon_app_5gsbi`.
- Keep all linked resources within the same monitoring session.

> Compatibility note:
> Older examples that use `sbi_mode`, `protocol_handlers`, or an `authentication` block do not match the current provider schema for `gigamon_app_5gsbi`.
> Use `type`, `ip_mapping_alias`, and `ericsson_vtap_config` as documented below.

---

## Argument Reference

### Required

- **`monitoring_session_id`** (String)
  Monitoring Session where this application is configured.
  This ensures the SBI app is scoped to one monitoring workflow.

- **`alias`** (String)
  Alias for the 5G-SBI application.
  Must be non-empty and match: `^[A-Za-z0-9_-]+$`.

- **`type`** (String)
  Application type.
  Required value: `ericssonVTap`.

- **`ip_mapping_alias`** (String)
  Alias of the IP mapping configuration used for NF instance resolution.
  Must be non-empty.

- **`http2_synthesize_tool_mtu_packet_size`** (Number)
  MTU packet size for HTTP/2 synthesis.
  Valid values: `0` or range `1200..9200`.

- **`ericsson_vtap_config`** (Object)
  Ericsson vTap configuration block.
  Required.

### Optional

- **`http2_synthesize_indexed_headers`** (Boolean)
  Default: `false`.

- **`http2_synthesize_compressed_headers`** (Boolean)
  Default: `false`.

- **`transaction_log`** (Boolean)
  Default: `false`.

- **`transaction_log_file_interval`** (Number)
  Default: `60`.
  Allowed values: `5`, `60`.

- **`log_folder_size`** (Number)
  Default: `0` (unlimited).
  Allowed values: `0` or range `50..40960`.

- **`stats_log`** (Boolean)
  Default: `false`.

- **`log_folder_loc`** (String)
  Optional log folder path.

### `ericsson_vtap_config` block

Required fields:

- **`num_tcp_flows`** (Number)
  Range: `128..16384`.

- **`tcp_flow_timeout`** (Number)
  Range: `0..7200`.

- **`num_streams_per_flow`** (Number)
  Range: `1..122880`.

Optional fields:

- **`mode`** (String)
  Default: `L7json`.
  Allowed value: `L7json`.

- **`eevtap_version`** (String)
  Default: `2`.
  Allowed values: `1`, `2`.

- **`http2_request_timeout`** (Number)
  Default: `10`.
  Range: `1..300`.

- **`http2_response_timeout`** (Number)
  Default: `2`.
  Range: `1..300`.

- **`destination_ip`** (String)
  Default: `SCP`.
  Allowed values: `SCP`, `destinationNF`.

- **`fqdn_mapping_alias`** (String)
  Optional FQDN mapping alias.

---

## Validation and Behavior Notes

- `monitoring_session_id` is required and changing it recreates the resource.
- `name` is computed by provider and set internally to `sbi5g`.
- `type` is required and currently supports only `ericssonVTap`.
- Alias format is validated by regex: `^[A-Za-z0-9_-]+$`.
- The resource exports a typed `id` that is used as `dest_id` in `gigamon_link`.

---

## Mode Behavior + Dependency Matrix

```text
Condition                                      | Required / Allowed                      | Not Allowed / Enforced Behavior
---------------------------------------------- | --------------------------------------- | -----------------------------------------------
Always                                         | type = ericssonVTap                     | Any other type value is rejected
Always                                         | ericsson_vtap_config required           | Missing block is rejected
ericsson_vtap_config.mode omitted              | Defaults to L7json                      | Values other than L7json are rejected
ericsson_vtap_config.destination_ip omitted    | Defaults to SCP                         | Values other than SCP or destinationNF rejected
http2_synthesize_tool_mtu_packet_size set      | 0 or 1200..9200                         | Other values are rejected
transaction_log_file_interval set              | 5 or 60                                 | Other values are rejected
log_folder_size set                            | 0 or 50..40960                          | Other values are rejected
```

---

## Attributes Reference

In addition to the arguments above, this resource exports:

- **`id`** (String)
  Typed application identifier in the form `app::5GSBI::<uuid>`.

- **`name`** (String)
  Computed internal app name. Always `sbi5g`.

---

## Import

Import using the monitoring session ID and the raw application UUID:

```shell
terraform import gigamon_app_5gsbi.example "<monitoring_session_id>::<raw_uuid>"
```

Import and ID usage notes:

- Import expects the monitoring session ID and raw app UUID joined by `::`.
- After import, Terraform state stores the typed ID format `app::5GSBI::<uuid>` in `id`.
- Use exported `id` as `dest_id` when linking traffic to this app.
