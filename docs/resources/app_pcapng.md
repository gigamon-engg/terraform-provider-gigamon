---
page_title: "PCapNG Application"
subcategory: "Applications"
description: "Manage the PCapNG application in Gigamon FM."
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

## Resource: `gigamon_app_pcapng`

The **PCapNG application** captures packets from a monitoring session and writes rolling capture files.

Use this resource to:

- define packet filters (BPF, IPs, VLANs),
- configure output file rotation and compression,
- tune runtime performance parameters.

Each `gigamon_app_pcapng` belongs to a single monitoring session.

---

## Example Usage

### Minimal PCapNG configuration

```hcl
resource "gigamon_app_pcapng" "app" {
  monitoring_session_id = gigamon_monitoring_session.ms.id
  capture_mode          = "continuous"

  packet_filter = {
    bpf_syntax  = "tcp and port 443"
    source_ip   = "192.0.2.10"
    dest_ip     = "198.51.100.10"
    vlan_filter = [100, 200]
  }

  output_config = {
    file_path     = "/var/log/gigamon/pcapng/capture.pcapng"
    max_file_size = 500
    rotation      = true
    compression   = "gzip"
  }

  performance = {
    buffer_size    = 64
    thread_count   = 4
    packet_snaplen = 65535
  }
}
```

---

## Argument Reference

### Required

- **`monitoring_session_id`** (String)
  Monitoring Session where this application is configured.
  Use the monitoring session ID so capture behavior is bound to the intended traffic pipeline.

- **`capture_mode`** (String)
  Capture mode.
  Valid values: `continuous`, `on-demand`, `triggered`.
  Choose `continuous` for always-on capture, `on-demand` for operator-driven sessions, and `triggered` for event-based captures.

### Optional

- **`packet_filter`** (Block)
  Packet filtering configuration.
  Use this to constrain captured traffic and avoid oversized capture files.

- **`output_config`** (Block)
  Capture output configuration.
  Use this to define storage path, rollover behavior, and compression policy.

- **`performance`** (Block)
  Performance tuning configuration.
  Use this to tune runtime resource consumption for expected packet rates.

### `packet_filter` block

- **`bpf_syntax`** (String)
  BPF filter expression.
  Prefer explicit filters (for example, protocol+port) to reduce storage and improve troubleshooting focus.

- **`source_ip`** (String)
  Source IP filter.
  Use when captures must be narrowed to specific sources.

- **`dest_ip`** (String)
  Destination IP filter.
  Use when captures must be narrowed to specific destinations.

- **`vlan_filter`** (List of Number)
  VLAN IDs to include.
  Helpful when multiple tenants/workloads share the same tap path.

### `output_config` block

- **`file_path`** (String)
  Output capture file path.
  Ensure the path is writable by the runtime and follows your retention conventions.

- **`max_file_size`** (Number)
  Maximum file size in MB.
  Range: `1-10000`.
  Set this with rotation to keep disk usage predictable.

- **`rotation`** (Boolean)
  Enable file rotation.
  Recommended for long-running capture modes.

- **`compression`** (String)
  Compression mode.
  Valid values: `none`, `gzip`, `xz`.
  Choose based on storage efficiency vs compression overhead requirements.

### `performance` block

- **`buffer_size`** (Number)
  Buffer size in MB.
  Range: `4-1024`.
  Increase for high packet-rate bursts to reduce drop risk.

- **`thread_count`** (Number)
  Thread count.
  Range: `1-16`.
  Scale up carefully based on available CPU and observed throughput.

- **`packet_snaplen`** (Number)
  Snapshot length per packet.
  Range: `64-65535`.
  Lower values reduce storage footprint; higher values preserve more packet payload context.

---

## Attributes Reference

In addition to the arguments above, this resource exports:

- **`id`** (String)
  Typed application identifier in the form `app::<type>::<uuid>`.

---

## Import

Import using the monitoring session ID and the raw application UUID:

```shell
terraform import gigamon_app_pcapng.example "<monitoring_session_id>::<raw_uuid>"
```
