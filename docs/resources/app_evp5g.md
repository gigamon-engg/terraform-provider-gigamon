---
page_title: "5G EVP Application"
subcategory: "Applications"
description: "Manage the 5G EVP application in Gigamon FM."
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

## Resource: `gigamon_app_evp5g`

The **5G EVP application** (Ericsson vTAP / 5G Cloud EVP) runs in a monitoring session and forwards processed 5G traffic through a configured transmit tunnel.

Use this resource to:

- define EVP5G identity and bind it to one monitoring session,
- configure receive and transmit tunnel behavior,
- tune packet ordering and diagnostics,
- control packet/csv/message logging behavior.

Each `gigamon_app_evp5g` belongs to a single monitoring session.

### Important behavior and constraints

- `monitoring_session_id` is immutable; changing it recreates the resource.
- `tx_tunnel.tx_dst_port` supports only `4754`.
- Certificate/private-key file paths for DTLS are provider-managed constants and are not exposed as Terraform arguments.
- Packet ordering is always enabled by the provider when `packet_ordering_config` is sent.

## Example Usage

### Minimal EVP5G application

```hcl
resource "gigamon_app_evp5g" "evp5g" {
	alias                 = "evp5g-app-1"
	monitoring_session_id = gigamon_monitoring_session.ms.id

	rx_tunnel {
		listen_ip_address = "10.10.10.101"
	}

	tx_tunnel {
		tx_remote_ip_address = "20.20.20.201"
		tx_dst_port          = 4754
		tx_src_ip_address    = ["1.1.1.1"]
	}
}
```

### EVP5G with packet ordering, diagnostics, and logging

```hcl
resource "gigamon_app_evp5g" "evp5g_tuned" {
	alias                 = "evp5g-prod"
	monitoring_session_id = gigamon_monitoring_session.ms.id

	rx_tunnel {
		listen_ip_address = "10.10.10.101"
		listen_port       = 4754
		rx_thread         = 8
		dtls              = "disable"
	}

	tx_tunnel {
		tx_remote_ip_address = "20.20.20.201"
		tx_src_port          = 4754
		tx_dst_port          = 4754
		tx_thread            = 4
		tx_src_ip_address    = ["1.1.1.1", "1.1.1.2"]
	}

	time_server_config {
		primary_server   = "192.0.2.10"
		secondary_server = "192.0.2.11"
	}

	packet_ordering_config {
		num_egress_flows              = 512
		egress_flow_timeout_value     = 660
		num_buckets                   = 50
		pkts_per_bucket               = 20000
		bucket_interval               = 1
		pkt_rx_outside_bucket_interval = "forward"
	}

	diagnostic_options {
		pct_disable = [0, 3, 7, 8]
		tx_disable  = false
	}

	logging {
		packet_capture_log_level = "none"
		csv_logging_level        = "disable"
		msg_log_level            = "info"
		log_folder_loc           = "/var/log"
	}
}
```

## Argument Reference

### Required

- **`monitoring_session_id`** (String)
	Monitoring session where the EVP5G application is deployed.
	Changing this value forces replacement.

- **`alias`** (String)
	Application alias.
	Allowed characters are alphanumeric, hyphen (`-`), and underscore (`_`).

### Optional

- **`rx_tunnel`** (Block)
	Receive tunnel settings for inbound EVP5G traffic.

- **`tx_tunnel`** (Block)
	Transmit tunnel settings for outbound EVP5G traffic.

- **`time_server_config`** (Block)
	Primary/secondary time server settings.

- **`packet_ordering_config`** (Block)
	Flow and bucket sizing for packet re-ordering behavior.

- **`diagnostic_options`** (Block)
	Per-feature diagnostic toggles.

- **`logging`** (Block)
	Packet/csv/message logging controls.

### `rx_tunnel` Block

- **`listen_ip_address`** (String)
	IPv4 or IPv6 address to bind for receiving traffic.

- **`listen_port`** (Number)
	Listener port.
	Range: `1..65535`.
	Default: `4754`.

- **`rx_thread`** (Number)
	RX thread count.
	Range: `1..16`.
	Default: `8`.

- **`dtls`** (String)
	Enables or disables DTLS.
	Allowed values: `enable`, `disable`.
	Default: `disable`.

- **`dtls_key_alias`** (String)
	Optional key alias used when DTLS is enabled.

### `tx_tunnel` Block

- **`tx_remote_ip_address`** (String)
	Remote destination IP address for transmitted traffic.

- **`tx_src_port`** (Number)
	Source port.
	Range: `1..65535`.
	Default: `4754`.

- **`tx_dst_port`** (Number)
	Destination port.
	Only supported value: `4754`.

- **`tx_thread`** (Number)
	TX thread count.
	Range: `1..16`.
	Default: `4`.

- **`tx_src_ip_address`** (List of String)
	One or more source IP addresses used by the transmit tunnel.
	At least one value is required when the block is present.

### `time_server_config` Block

- **`primary_server`** (String)
	Primary time server IP address.

- **`secondary_server`** (String)
	Optional secondary time server IP address.

### `packet_ordering_config` Block

- **`num_egress_flows`** (Number)
	Number of tracked egress flows.
	Range: `32..16384`.
	Default: `512`.

- **`egress_flow_timeout_value`** (Number)
	Egress flow timeout in seconds.
	Range: `360..1860`.
	Default: `660`.

- **`num_buckets`** (Number)
	Bucket count used by packet ordering.
	Range: `10..200`.
	Default: `50`.

- **`pkts_per_bucket`** (Number)
	Packets per bucket.
	Range: `10000..100000`.
	Default: `20000`.

- **`bucket_interval`** (Number)
	Bucket interval in seconds.
	Range: `1..5`.
	Default: `1`.

- **`pkt_rx_outside_bucket_interval`** (String)
	Action when packets arrive outside the bucket interval.
	Allowed values: `forward`, `discard`.
	Default: `forward`.

### `diagnostic_options` Block

- **`pct_disable`** (List of Number)
	List of PCT indexes to disable.
	Each value range: `0..12`.

- **`tx_disable`** (Boolean)
	Disables transmit behavior for diagnostics.
	Default: `false`.

### `logging` Block

- **`packet_capture_log_level`** (String)
	Packet capture log level.
	Allowed values: `all`, `receive`, `transmit`, `none`.

- **`csv_logging_level`** (String)
	CSV logging mode.
	Allowed values: `enable`, `disable`.

- **`msg_log_level`** (String)
	Message logging verbosity.
	Allowed values: `none`, `fatal`, `error`, `info`, `detail`, `full-parse`.

- **`log_folder_loc`** (String)
	Log directory path.
	Default: `/var/log`.

## Attributes Reference

In addition to all arguments above, `gigamon_app_evp5g` exports:

- **`id`** (String)
	Typed application ID used for references in dependent resources.
