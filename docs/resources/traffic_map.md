---
page_title: "Traffic Map"
subcategory: "Maps"
description: "Manage traffic maps in Gigamon FM."
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

# gigamon_traffic_map

## Resource: `gigamon_traffic_map`

A **traffic map** classifies and forwards monitored traffic inside a Monitoring Session.
Each traffic map contains one or more **rule sets** (`rule_sets`) with **pass** and/or **drop** rules.

Each rule set is bound to an **AEP ID** (`aep_id`), which is the output endpoint consumed by `gigamon_link`
to connect the map to applications, tunnels, or other maps.

**Related map types**
> - `gigamon_inclusion_map` – ATS inclusion map (only `pass_rules` are allowed)
> - `gigamon_exclusion_map` – ATS exclusion map (only `drop_rules` are allowed)

---

## Example Usage

### Simple traffic map with one rule set

```hcl
resource "gigamon_traffic_map" "web_traffic" {
  monitoring_session_id = gigamon_monitoring_session.ms.id
  name                  = "web-traffic-map"
  description           = "Select HTTP/HTTPS traffic from the frontend tier"

  rule_sets = [
    {
      rule_set_id = "1"
      priority    = 1   # lower number = higher priority
      aep_id      = 10  # output AEP ID consumed by gigamon_link.source_aep_id

      pass_rules = [
        {
          rule_id = 1

          ipv4_source = {
            address   = "10.0.0.0"
            cidr_mask = "24"
          }

          ipv4_protocol = {
            protocol_min    = 6
            protocol_max    = 6
            protocol_subset = "all"
          }
        }
      ]
    }
  ]
}
```

### Linking a traffic map to an application via `source_aep_id`

```hcl
resource "gigamon_link" "web_to_app" {
  monitoring_session_id = gigamon_monitoring_session.ms.id

  # Source is the traffic map; source_aep_id must match rule_sets.aep_id
  source_id     = gigamon_traffic_map.web_traffic.id
  source_aep_id = 10

  dest_id = gigamon_application.app_ats.id
}
```

- `rule_sets[*].aep_id` selects which logical output inside the map the matched traffic hits.
- `gigamon_link.source_aep_id` tells FM which AEP of the source map to connect to the destination.
- `source_aep_id` is **required** when `source_id` refers to any map or load-balancing app, and **invalid** otherwise.

---

## AFI usage in `gigamon_traffic_map`

AFI behavior on a traffic map is configured through the top-level `asf` block.
The exact path is:

- `asf.asf_profile_config.session_fields`
- `asf.asf_profile_config.timeout`
- `asf.asf_profile_config.packet_count`
- `asf.asf_profile_config.bidi`
- `asf.asf_profile_config.buffering.enabled`
- `asf.asf_profile_config.buffering.protocol`
- `asf.asf_profile_config.buffering.buffer_count_before_match`

`app_rules` under each rule set are ASF/AFI-aware and are supported only when
`asf.asf_profile_config` is configured on the same `gigamon_traffic_map`.

### Placement and structure

```hcl
resource "gigamon_traffic_map" "afi_map" {
  monitoring_session_id = gigamon_monitoring_session.ms.id
  name                  = "afi-map"

  asf = {
    asf_profile_config = {
      session_fields = [
        { pos = 2, type = "fiveTuple" },
        { pos = 2, type = "vlanId" }
      ]
      timeout      = 15
      packet_count = 30
      bidi         = true

      buffering = {
        enabled                   = true
        protocol                  = "tcpUdp"
        buffer_count_before_match = 20
      }
    }
  }

  rule_sets = [
    {
      rule_set_id = "1"
      priority    = 1
      aep_id      = 10
      pass_rules  = [{ rule_id = 1, ip_version = { ip_version = "v4" } }]
    }
  ]
}
```

### Example: AFI map linked by `source_aep_id`

```hcl
resource "gigamon_traffic_map" "afi_to_tool" {
  monitoring_session_id = gigamon_monitoring_session.ms.id
  name                  = "afi-to-tool"

  asf = {
    asf_profile_config = {
      session_fields = [{ pos = 2, type = "fiveTuple" }]
      timeout        = 15
      packet_count   = 30
      bidi           = true
      buffering = {
        enabled                   = true
        protocol                  = "tcpUdp"
        buffer_count_before_match = 20
      }
    }
  }

  rule_sets = [
    {
      rule_set_id = "1"
      priority    = 1
      aep_id      = 20
      pass_rules = [
        {
          rule_id = 1
          ipv4_protocol = {
            protocol_min = 6
          }
        }
      ]
    }
  ]
}

resource "gigamon_link" "afi_map_to_app" {
  monitoring_session_id = gigamon_monitoring_session.ms.id

  source_id     = gigamon_traffic_map.afi_to_tool.id
  source_aep_id = 20

  dest_id = gigamon_application.app_ats.id
}
```

### Example: `app_rules` with ASF enabled

```hcl
resource "gigamon_traffic_map" "afi_with_app_rules" {
  monitoring_session_id = gigamon_monitoring_session.ms.id
  name                  = "afi-with-app-rules"

  asf = {
    asf_profile_config = {
      session_fields = [{ pos = 2, type = "fiveTuple" }]
      timeout        = 15
      packet_count   = 30
      bidi           = true
      buffering = {
        enabled                   = true
        protocol                  = "tcpUdp"
        buffer_count_before_match = 20
      }
    }
  }

  rule_sets = [
    {
      rule_set_id = "1"
      priority    = 1
      aep_id      = 30

      pass_rules = [{ rule_id = 1, ip_version = { ip_version = "v4" } }]

      app_rules = {
        pass_rules = [
          {
            app_profile_config = {
              type = "filter"
              applications = [
                { name = "ssl" },
                { name = "http" }
              ]
            }
          }
        ]
      }
    }
  ]
}
```

### Ordering and evaluation behavior

- Rule set priority controls rule set evaluation order (`priority` 1 is highest).
- `source_aep_id` on `gigamon_link` selects which rule set output (`rule_sets[*].aep_id`) is wired to the destination.
- ASF options are map-level controls and are not configured per individual rule.
- `app_rules` are validated as ASF-dependent by the provider; configure ASF first when using application-level rules.

### AFI/ASF validation notes (provider-validated)

- `asf` is optional, but if set it must include `asf_profile_config`.
- ASF requires a monitoring session that already has `scale_unit` configured.
- `session_fields`:
  - Must contain 1 to 2 entries.
  - Must contain exactly one `type = "fiveTuple"` entry.
  - May contain at most one additional `type = "vlanId"` entry.
  - `pos` must be `2`.
- `timeout` default is `15`, allowed range is `10` to `20`.
- `packet_count` default is `30`, allowed range is `2` to `100`.
- `buffering.buffer_count_before_match` default is `20`, allowed range is `3` to `20`.
- `packet_count` must be greater than or equal to `buffering.buffer_count_before_match`.
- `bidi` default is `true`.
- `buffering.enabled` default is `true`.
- `buffering.protocol` default is `tcpUdp`. The provider applies a default but does not currently enforce an enum validator at plan time.
- `rule_sets.app_rules` are supported only when `asf.asf_profile_config` is present.

---

### Multiple rule sets and rules from variables

Use a `for` expression to build `rule_sets` and the nested `pass_rules` / `drop_rules` lists
from structured variables, eliminating repetition when managing multiple application tiers or
traffic classes.

```hcl
# ── Variables ────────────────────────────────────────────────────────────────

variable "traffic_tiers" {
  description = "One rule set per application tier; each tier captures its own subnet ranges"
  type = list(object({
    name       = string          # used to label rules, not sent to FM
    priority   = number          # 1–5, lower = higher priority
    aep_id     = number          # 2–63, output AEP for gigamon_link
    src_cidrs  = list(string)    # one pass_rule per CIDR, e.g. ["10.0.1.0/24", "10.0.2.0/24"]
    dst_cidrs  = list(string)    # one pass_rule per CIDR (or empty list to skip)
    protocols  = list(number)    # one pass_rule per protocol number (or empty list to skip)
  }))
}

# ── Locals ────────────────────────────────────────────────────────────────────

locals {
  rule_sets = [
    for i, tier in var.traffic_tiers : {
      rule_set_id = tostring(i + 1)
      priority    = tier.priority
      aep_id      = tier.aep_id

      # One pass_rule per source CIDR — rules within a rule set are OR-combined
      pass_rules = concat(
        [
          for j, cidr in tier.src_cidrs : {
            rule_id = j + 1
            ipv4_source = {
              address   = cidrhost(cidr, 0)
              cidr_mask = tostring(split("/", cidr)[1])
            }
          }
        ],
        [
          for j, cidr in tier.dst_cidrs : {
            rule_id = length(tier.src_cidrs) + j + 1
            ipv4_destination = {
              address   = cidrhost(cidr, 0)
              cidr_mask = tostring(split("/", cidr)[1])
            }
          }
        ],
        [
          for j, proto in tier.protocols : {
            rule_id = length(tier.src_cidrs) + length(tier.dst_cidrs) + j + 1
            ipv4_protocol = {
              protocol_min = proto
            }
          }
        ],
      )

      # No drop_rules in this example; add a drop_rules list here if needed
      drop_rules = []
    }
  ]
}

# ── Traffic map ───────────────────────────────────────────────────────────────

resource "gigamon_traffic_map" "tiered" {
  monitoring_session_id = gigamon_monitoring_session.ms.id
  name                  = "tiered-traffic-map"

  rule_sets = local.rule_sets
}
```

Example `terraform.tfvars`:

```hcl
traffic_tiers = [
  {
    name      = "frontend"
    priority  = 1
    aep_id    = 10
    src_cidrs = ["10.0.1.0/24", "10.0.2.0/24"]
    dst_cidrs = []
    protocols = [6, 17]   # TCP and UDP
  },
  {
    name      = "backend"
    priority  = 2
    aep_id    = 11
    src_cidrs = ["10.0.3.0/24"]
    dst_cidrs = ["10.0.4.0/24"]
    protocols = []
  },
]
```

### Rule sets with both pass and drop rules per tier

```hcl
variable "mixed_tiers" {
  type = list(object({
    priority       = number
    aep_id         = number
    allowed_cidrs  = list(string)   # → pass_rules
    blocked_cidrs  = list(string)   # → drop_rules
  }))
}

resource "gigamon_traffic_map" "mixed" {
  monitoring_session_id = gigamon_monitoring_session.ms.id
  name                  = "mixed-pass-drop-map"

  rule_sets = [
    for i, tier in var.mixed_tiers : {
      rule_set_id = tostring(i + 1)
      priority    = tier.priority
      aep_id      = tier.aep_id

      pass_rules = [
        for j, cidr in tier.allowed_cidrs : {
          rule_id = j + 1
          ipv4_source = {
            address   = cidrhost(cidr, 0)
            cidr_mask = tostring(split("/", cidr)[1])
          }
        }
      ]

      drop_rules = [
        for j, cidr in tier.blocked_cidrs : {
          rule_id = j + 1
          ipv4_source = {
            address   = cidrhost(cidr, 0)
            cidr_mask = tostring(split("/", cidr)[1])
          }
        }
      ]
    }
  ]
}
```

---

## Argument Reference

### Top-level arguments

* `monitoring_session_id` (String, **Required**) – ID of the Monitoring Session that owns this map. Typically set from `gigamon_monitoring_session.<name>.id`. Changing this forces a new resource.
* `name` (String, **Required**) – Name of the traffic map, unique within the Monitoring Session.
* `description` (String, Optional) – Free-form description for this traffic map.
* `asf` (Object, Optional) – ASF profile used for AFI behavior on this traffic map. Must be provided as `asf.asf_profile_config` when `asf` is present.
* `rule_sets` (List of Objects, **Required**) – One or more rule sets that define how traffic is matched and forwarded. At least **1** and at most **5** rule sets per map.

---

## `rule_sets`

```hcl
rule_sets = [
  {
    rule_set_id = "1"
    priority    = 1
    aep_id      = 10

    pass_rules = [{ ... }]
    drop_rules = [{ ... }]
  }
]
```

* `rule_set_id` (String, **Required**) – Identifier of this rule set within the map. Must be a string `"1"`–`"5"`.
* `priority` (Number, **Required**) – Priority of this rule set. Range: **1–5**.
  Lower value = higher priority. When multiple rule sets match, the one with the lowest value is evaluated first.
* `aep_id` (Number, **Required**) – Output AEP endpoint ID for this rule set. Range: **2–63**.
  This value must be referenced by `gigamon_link.source_aep_id` to connect map output to a destination.
* `pass_rules` (List of Objects, Optional) – Rules for traffic to **forward** to `aep_id`. At least one rule is required when this block is specified. At least one of `pass_rules` or `drop_rules` must be present per rule set.
* `drop_rules` (List of Objects, Optional) – Rules for traffic to **discard**. At least one rule is required when this block is specified. At least one of `pass_rules` or `drop_rules` must be present per rule set.
* `app_rules` (Object, Optional) – Application rules (`pass_rules` and/or `drop_rules`) for ASF-enabled traffic maps. Supported only when `asf.asf_profile_config` is configured.

> **Traffic map**: both `pass_rules` and `drop_rules` are allowed in the same rule set.

> **Inclusion map** (`gigamon_inclusion_map`): only `pass_rules` are permitted.

> **Exclusion map** (`gigamon_exclusion_map`): only `drop_rules` are permitted.

---

## `pass_rules` / `drop_rules`

Each item represents one rule. All rule elements inside a single rule are combined with **AND**.
Multiple rules within `pass_rules` or `drop_rules` are combined with **OR**.

```hcl
pass_rules = [
  {
    rule_id = 1

    ether_type = { ... }
    ipv4_source = { ... }
    ipv4_protocol = { ... }
    # ... other rule element blocks ...
  }
]
```

* `rule_id` (Number, **Required**) – Identifier of this rule within the rule set. Recommended range **1–5**.

Each rule may include zero or more of the following match condition blocks. At least one must be present.

- `ether_type` – Match on EtherType / TPID
- `l2_src_mac` – Match on source MAC address
- `l2_dst_mac` – Match on destination MAC address
- `ip_version` – Match on IP version (v4 or v6)
- `ipv4_source` – Match on IPv4 source address/range
- `ipv4_destination` – Match on IPv4 destination address/range
- `ipv6_source` – Match on IPv6 source address/range
- `ipv6_destination` – Match on IPv6 destination address/range
- `vm_name_source` – Match on source VM name prefix
- `vm_name_destination` – Match on destination VM name prefix
- `vm_tag_source` – Match on source VM tag key/value
- `vm_tag_destination` – Match on destination VM tag key/value
- `ipv4_dscp` – Match on IPv4 DSCP code point
- `ipv6_dscp` – Match on IPv6 DSCP code point
- `ipv4_fragmentation` – Match on IPv4 fragmentation mode
- `ipv4_protocol` – Match on IPv4 protocol number
- `erspan_id` – Match on ERSPAN ID
- `ipv4_ttl` – Match on IPv4 TTL
- `ipv4_tos` – Match on IPv4 TOS byte
- `gre_key` – Match on GRE key
- `gtp_teid` – Match on GTP-U Tunnel Endpoint ID
- `host_name` – Match on source host name prefix
- `ipv6_flow_label` – Match on IPv6 Flow Label
- `ipv6_next_header` – Match on IPv6 Next Header protocol
- `mpls_label` – Match on MPLS label value
- `port_destination` – Match on destination port
- `port_source` – Match on source transport-layer port
- `tcp_control` – Match on TCP control/flag bits
- `vlan` – Match on VLAN ID
- `vntag_dst_vif_id` – Match on VN-Tag destination VIF ID
- `vntag_src_vif_id` – Match on VN-Tag source VIF ID
- `vntag_vif_list_id` – Match on VN-Tag VIF List ID
- `vxlan_id` – Match on VXLAN ID / VNI

---

## Rule Element Blocks

### `ether_type`

```hcl
ether_type = {
  nested_level_count = 0
  ether_type         = "0x0800"
  # or use a range:
  ether_type_start   = "0x0800"
  ether_type_end     = "0x86DD"
}
```

* `nested_level_count` (Number, Optional, default `0`) – VLAN nesting level; `0` = any level.
* `ether_type` (String, Optional) – Single EtherType hex value with `0x` prefix (e.g. `"0x0800"`, `"0x86DD"`). Mutually exclusive with `ether_type_start`. Exactly one of `ether_type` or `ether_type_start` must be provided.
* `ether_type_start`, `ether_type_end` (String, Optional) – EtherType range; both must be set together. Mutually exclusive with `ether_type`.

---

### `l2_src_mac` / `l2_dst_mac`

```hcl
l2_src_mac = {
  nested_level_count = 0
  mac_address        = "00:11:22:33:44:55"
  mac_address_mask   = "FF:FF:FF:FF:FF:FF"
}
# or as a range:
l2_dst_mac = {
  mac_address_start = "00:11:22:33:44:00"
  mac_address_end   = "00:11:22:33:44:FF"
}
```

* `nested_level_count` (Number, Optional, default `0`) – MAC layer to inspect for MAC-in-MAC; `0` = any.
* `mac_address` (String, Optional) – Single MAC address to match (e.g. `"00:1A:2B:3C:4D:5E"`). Mutually exclusive with `mac_address_start`. Exactly one of `mac_address` or `mac_address_start` must be provided.
* `mac_address_mask` (String, Optional, default `FF:FF:FF:FF:FF:FF`) – Bitmask applied to `mac_address` to define a range. Requires `mac_address`.
* `mac_address_start`, `mac_address_end` (String, Optional) – MAC address range; both must be set together. Mutually exclusive with `mac_address`.

---

### `ip_version`

```hcl
ip_version = {
  nested_level_count = 0
  ip_version         = "v4"  # or "v6"
}
```

* `nested_level_count` (Number, Optional, default `0`)
* `ip_version` (String, **Required**) – `"v4"` or `"v6"`.

---

### `ipv4_source` / `ipv4_destination`

```hcl
ipv4_source = {
  nested_level_count = 0
  address            = "10.0.0.0"
  cidr_mask          = "24"
  # alternatives: address_max or netmask
}
```

* `nested_level_count` (Number, Optional, default `0`, range 0–3) – IPv4 header depth for tunneled traffic.
* `address` (String, **Required**) – IPv4 address (start of range or network address).
* `address_max` (String, Optional) – Range end; mutually exclusive with `cidr_mask` and `netmask`.
* `cidr_mask` (String, Optional) – CIDR prefix length `"1"`–`"32"`; mutually exclusive with `address_max` and `netmask`.
* `netmask` (String, Optional) – Dotted-decimal netmask (must be contiguous); mutually exclusive with `address_max` and `cidr_mask`.

---

### `ipv6_source` / `ipv6_destination`

```hcl
ipv6_source = {
  nested_level_count = 0
  address            = "2001:db8::"
  cidr_mask          = "64"
}
```

* `nested_level_count` (Number, Optional, default `0`, range 0–3)
* `address` (String, **Required**) – IPv6 address or start of range.
* `address_max` (String, Optional) – IPv6 range end; mutually exclusive with `cidr_mask` and `netmask`.
* `cidr_mask` (String, Optional) – `"1"`–`"128"`; mutually exclusive with `address_max` and `netmask`.
* `netmask` (String, Optional) – IPv6 netmask as 8 uppercase hextets.

---

### `vm_name_source` / `vm_name_destination`

```hcl
vm_name_source = {
  vm_name_prefix = "frontend-"
}
```

* `vm_name_prefix` (String, **Required**) – Prefix of the VM name to match (exact prefix, no wildcards).
  For vSphere this is the VM name; for clouds it is the VM name as shown in GigaVUE-FM.

---

### `vm_tag_source` / `vm_tag_destination`

```hcl
vm_tag_source = {
  tag_name  = "environment"
  tag_value = "prod"
}
```

* `tag_name` (String, **Required**) – Tag key (vSphere tag name or cloud tag key).
* `tag_value` (String, **Required**) – Tag value (or vSphere tag category).

---

### `ipv4_dscp` / `ipv6_dscp`

```hcl
ipv4_dscp = {
  nested_level_count = 0
  dscp               = "af11"
}
```

* `nested_level_count` (Number, Optional, default `0`, range `0–3`) – Which IP header to inspect in tunneled/stacked traffic: `0`=any, `1`=outermost, `2`=second, `3`=third.
* `dscp` (String, **Required**) – DSCP code point. Valid values: `af11`, `af12`, `af13`, `af21`, `af22`, `af23`, `af31`, `af32`, `af33`, `af41`, `af42`, `af43`, `ef`.

---

### `ipv4_fragmentation`

```hcl
ipv4_fragmentation = {
  nested_level_count = 0
  mode               = "unfragmented_only"
}
```

* `nested_level_count` (Number, Optional, default `0`, range 0–3)
* `mode` (String, **Required**) – One of:
  * `unfragmented_only`
  * `any_fragment`
  * `non_first_fragments`
  * `first_fragment_only`
  * `first_or_unfragmented`

---

### `ipv4_protocol`

```hcl
ipv4_protocol = {
  nested_level_count = 0
  protocol_min       = 6
  protocol_max       = 6
  protocol_subset    = "all"
}
```

* `nested_level_count` (Number, Optional, default `0`, range 0–3)
* `protocol_min` (Number, **Required**) – Lower bound (inclusive), 0–255.
* `protocol_max` (Number, Optional) – Upper bound (inclusive), 0–255. Must be greater than `protocol_min` if set.
* `protocol_subset` (String, Optional, default `"all"`) – `"all"`, `"even"`, or `"odd"` (requires `protocol_max` for even/odd).

---

### `erspan_id`

```hcl
erspan_id = {
  erspan_id_min    = 1
  erspan_id_max    = 10
  erspan_id_subset = "all"
}
```

* `erspan_id_min` (Number, **Required**) – Lower bound (inclusive), 1–1024.
* `erspan_id_max` (Number, Optional) – Upper bound (inclusive), 1–1024; must be greater than `erspan_id_min`.
* `erspan_id_subset` (String, Optional, default `"all"`) – `"all"`, `"even"`, or `"odd"` (requires `erspan_id_max` for even/odd).

---

### `ipv4_ttl`

```hcl
ipv4_ttl = {
  nested_level_count = 0
  ttl_min            = 64
  ttl_max            = 128
  ttl_subset         = "all"
}
```

* `nested_level_count` (Number, Optional, default `0`, range 0–3)
* `ttl_min` (Number, **Required**) – 0–255.
* `ttl_max` (Number, Optional) – 0–255; must be greater than `ttl_min` if set.
* `ttl_subset` (String, Optional, default `"all"`) – `"all"`, `"even"`, or `"odd"` (requires `ttl_max` for even/odd).

---

### `ipv4_tos`

```hcl
ipv4_tos = {
  nested_level_count = 0
  tos_min            = "0A"
  tos_max            = "1F"
  tos_subset         = "all"
}
```

* `nested_level_count` (Number, Optional, default `0`, range 0–3)
* `tos_min` (String, **Required**) – 1-byte hex value (2 hex digits, e.g. `"0A"`).
* `tos_max` (String, Optional) – 1-byte hex value; must be greater than `tos_min` when set.
* `tos_subset` (String, Optional, default `"all"`) – `"all"`, `"even"`, or `"odd"` (requires `tos_max` for even/odd).

---

### `gre_key`

```hcl
gre_key = {
  gre_key_min    = "0000000A"
  gre_key_max    = "000000FF"
  gre_key_subset = "all"
}
```

* `gre_key_min` (String, **Required**) – Lower bound as 4-byte hex (8 hex digits).
* `gre_key_max` (String, Optional) – Upper bound as 4-byte hex; must be greater than `gre_key_min` if set.
* `gre_key_subset` (String, Optional, default `"all"`) – `"all"`, `"even"`, or `"odd"` (requires `gre_key_max` for even/odd).

---

### `gtp_teid`

Matches the GTP-U Tunnel Endpoint Identifier (TEID) field in GTP-U user-plane traffic.
The TEID is a 32-bit value represented as a 4-byte hexadecimal string.

```hcl
gtp_teid = {
  teid_min           = "00000001"
  nested_level_count = 0
  subnet             = "none"
}
```

Or with a range:

```hcl
gtp_teid = {
  teid_min           = "00001000"
  teid_max           = "00001FFF"
  nested_level_count = 0
  subnet             = "none"
}
```

* `teid_min` (String, **Required**) – Lower bound as 4-byte hex (exactly 8 hex characters, e.g. `"00000001"`). Range: `"00000000"` to `"FFFFFFFF"`.
* `teid_max` (String, Optional) – Upper bound as 4-byte hex; must be greater than `teid_min` when set.
* `nested_level_count` (Number, Optional, default `0`, range 0–3) – GTP-U header depth for tunneled traffic. `0` = any level.
* `subnet` (String, Optional, default `"none"`) – Restrict matches within [teid_min, teid_max] to `"none"`, `"even"`, or `"odd"`. Requires `teid_max` when set to `"even"` or `"odd"`.

---

### `host_name`

Matches traffic based on source host name prefix.
This condition applies to the source host name field and matches by prefix (not wildcard).

```hcl
host_name = {
  src_host_prefix = "api.example.com"
}
```

* `src_host_prefix` (String, Optional) – Host name prefix to match. The match is prefix-based: a prefix `"api"` will match hostnames starting with `"api"` (e.g. `"api.example.com"`, `"api-server"`). Exact match is performed when the full hostname is specified.

---

### `ipv6_flow_label`

Matches the 20-bit IPv6 Flow Label field.
The flow label is a 20-bit value, represented as a numeric value in the range 0 through 1,048,575 (0x000000–0x0FFFFF).

```hcl
ipv6_flow_label = {
  label_min = 291      # decimal; 0x00123 in hex
  pos       = 0
  subnet    = "none"
}
```

Or with a range:

```hcl
ipv6_flow_label = {
  label_min = 256      # decimal; 0x00100 in hex
  label_max = 511      # decimal; 0x001FF in hex
  pos       = 0
  subnet    = "none"
}
```

* `label_min` (Number, **Required**) – Lower bound (inclusive), 0–1,048,575 (20-bit range).
* `label_max` (Number, Optional) – Upper bound (inclusive), 0–1,048,575; must be greater than `label_min` when set.
* `pos` (Number, Optional, default `0`, range 0–3) – IPv6 header position in stacked/tunneled traffic. `0` = any level, `1` = outermost, `2` = second, `3` = third.
* `subnet` (String, Optional, default `"none"`) – Restrict matches within [label_min, label_max] to `"none"`, `"even"`, or `"odd"`. Requires `label_max` when set to `"even"` or `"odd"`.

---

### `ipv6_next_header`

Matches the IPv6 Next Header protocol field (0–255).
This condition applies to the Next Header field in the IPv6 header and is independent of IPv4 protocol matching.

```hcl
ipv6_next_header = {
  header_min = 6       # TCP
  pos        = 0
  subnet     = "none"
}
```

Or with a range:

```hcl
ipv6_next_header = {
  header_min = 6       # TCP
  header_max = 17      # UDP
  pos        = 0
  subnet     = "none"
}
```

* `header_min` (Number, **Required**) – Lower bound (inclusive), 0–255. Common values: `6` (TCP), `17` (UDP), `58` (ICMPv6).
* `header_max` (Number, Optional) – Upper bound (inclusive), 0–255; must be greater than `header_min` when set.
* `pos` (Number, Optional, default `0`, range 0–3) – IPv6 header position in stacked/tunneled traffic. `0` = any level.
* `subnet` (String, Optional, default `"none"`) – Restrict matches within [header_min, header_max] to `"none"`, `"even"`, or `"odd"`. Requires `header_max` when set to `"even"` or `"odd"`.

---

### `mpls_label`

Matches an MPLS label value in the MPLS label stack.

```hcl
mpls_label = {
  value_min = 100
  pos       = 0
  subnet    = "none"
}
```

Or with a range:

```hcl
mpls_label = {
  value_min = 100
  value_max = 200
  pos       = 0
  subnet    = "none"
}
```

* `value_min` (Number, **Required**) – Lower bound (inclusive), 1–1,048,576. Standard MPLS label range is typically 1–1,048,575 for user labels.
* `value_max` (Number, Optional) – Upper bound (inclusive), 1–1,048,576; must be greater than `value_min` when set.
* `pos` (Number, Optional, default `0`, range 0–4) – MPLS label position in the label stack. `0` = outermost label, `1` = second label, etc.
* `subnet` (String, Optional, default `"none"`) – Restrict matches within [value_min, value_max] to `"none"`, `"even"`, or `"odd"`. Requires `value_max` when set to `"even"` or `"odd"`.

---

### `port_destination`

Matches the Layer-4 destination port (TCP or UDP).
The port range is 0–65,535. Note that protocol matching (TCP/UDP vs other protocols) is typically configured using a separate `ipv4_protocol` or `ipv6_next_header` condition.

```hcl
port_destination = {
  port_min = 443       # HTTPS
  pos      = 0
  subnet   = "none"
}
```

Or with a range:

```hcl
port_destination = {
  port_min = 8000
  port_max = 8100
  pos      = 0
  subnet   = "none"
}
```

* `port_min` (Number, **Required**) – Lower bound (inclusive), 0–65,535.
* `port_max` (Number, Optional) – Upper bound (inclusive), 0–65,535; must be greater than `port_min` when set.
* `pos` (Number, Optional, default `0`, range 0–3) – Transport header position in stacked/tunneled traffic. `0` = any level, `1` = outermost, etc.
* `subnet` (String, Optional, default `"none"`) – Restrict matches within [port_min, port_max] to `"none"`, `"even"`, or `"odd"`. Requires `port_max` when set to `"even"` or `"odd"`.

---

### `port_source`

Matches the Layer-4 source port. Use the Terraform block name `port_source`; the FM internal type is `portSrc`.

This block is valid anywhere a rule element is valid, so it can be used in `pass_rules` and `drop_rules` on `gigamon_traffic_map`. On `gigamon_inclusion_map` only `pass_rules` are allowed, and on `gigamon_exclusion_map` only `drop_rules` are allowed. Match conditions in one rule are AND-combined. Separate rules in the same `pass_rules` or `drop_rules` list are OR-combined.

Pair source-port matching with a protocol match such as `ipv4_protocol = { protocol_min = 6 }` for TCP or `ipv4_protocol = { protocol_min = 17 }` for UDP. The provider schema does not force that pairing, but without it FM may match the same numeric field in different transport contexts.

Single-port example:

```hcl
port_source = {
  port_min           = 443
  nested_level_count = 0
}
```

Range example:

```hcl
port_source = {
  port_min           = 1024
  port_max           = 49151
  nested_level_count = 0
  subnet             = "none"
}
```

Complete rule example:

```hcl
resource "gigamon_traffic_map" "port_source" {
  monitoring_session_id = gigamon_monitoring_session.ms.id
  name                  = "port-source-map"

  rule_sets = [{
    rule_set_id = "1"
    priority    = 1
    aep_id      = 10
    pass_rules = [{
      rule_id = 1
      port_source = {
        port_min           = 1024
        port_max           = 49151
        nested_level_count = 0
        subnet             = "none"
      }
      ipv4_protocol = {
        protocol_min = 6
      }
    }]
  }]
}
```

* `port_min` (Number, **Required**) – Lower bound (inclusive), `0` to `65535`.
* `port_max` (Number, Optional) – Upper bound (inclusive), `0` to `65535`. The provider rejects values below `port_min`, but accepts `port_max == port_min`.
* `nested_level_count` (Number, Optional, default `0`, range `0–3`) – Which transport header to inspect in stacked or tunneled traffic. `0` means any matching level, `1` outermost, `2` second, `3` third.
* `subnet` (String, Optional, default `"none"`) – `"none"`, `"even"`, or `"odd"`. `"even"` and `"odd"` require `port_max`.

Validation notes:

* Missing `port_min` fails schema validation.
* `port_min` and `port_max` outside `0..65535` fail validation.
* `port_max < port_min` fails with an `Invalid port source range` error.
* `subnet = "even"` or `"odd"` without `port_max` fails with an `Invalid port source subnet` error.
* `nested_level_count` outside `0..3` fails validation.

---

### `tcp_control`

Matches TCP control bits. Use the Terraform block name `tcp_control`; the FM internal type is `tcpCtl`.

This condition should be paired with a TCP protocol match such as `ipv4_protocol = { protocol_min = 6 }` or `ipv6_next_header = { header_min = 6 }`. The provider exposes `tcp_control` as a standalone rule element, but the field only has protocol meaning for TCP traffic.

```hcl
resource "gigamon_traffic_map" "tcp_control" {
  monitoring_session_id = gigamon_monitoring_session.ms.id
  name                  = "tcp-control-map"

  rule_sets = [{
    rule_set_id = "1"
    priority    = 1
    aep_id      = 11
    pass_rules = [{
      rule_id = 1
      ipv4_protocol = {
        protocol_min = 6
      }
      tcp_control = {
        value              = "02"
        mask               = "3F"
        nested_level_count = 0
      }
    }]
  }]
}
```

* `value` (String, **Required**) – One-byte hexadecimal value, exactly two hex characters, no `0x` prefix. Example: `"02"` for SYN.
* `mask` (String, Optional) – One-byte hexadecimal mask, exactly two hex characters, no `0x` prefix. Example: `"3F"` to mask the low six TCP flag bits.
* `nested_level_count` (Number, Optional, default `0`, range `0–3`) – Which TCP header to inspect. `0` means any matching level, `1` outermost, `2` second, `3` third.

Common values:

* `"01"` FIN
* `"02"` SYN
* `"04"` RST
* `"08"` PSH
* `"10"` ACK
* `"20"` URG

Validation notes:

* `value` is required.
* `value` and `mask` must match exactly two hexadecimal characters. `"0x02"` and `"2"` are both invalid.
* `nested_level_count` outside `0..3` fails validation.
* Use a protocol condition for TCP so the rule intent is unambiguous and FM evaluates the field in the expected context.

---

### `vlan`

Matches a VLAN ID. Use the Terraform block name `vlan`; the FM internal type is also `vlan`.

The provider-facing field names are `vlan_min`, `vlan_max`, `nested_level_count`, and `subnet`.

Exact VLAN example:

```hcl
vlan = {
  vlan_min           = 100
  nested_level_count = 0
}
```

Range example:

```hcl
vlan = {
  vlan_min           = 100
  vlan_max           = 200
  nested_level_count = 0
  subnet             = "none"
}
```

Complete rule example:

```hcl
resource "gigamon_traffic_map" "vlan" {
  monitoring_session_id = gigamon_monitoring_session.ms.id
  name                  = "vlan-map"

  rule_sets = [{
    rule_set_id = "1"
    priority    = 1
    aep_id      = 12
    pass_rules = [{
      rule_id = 1
      vlan = {
        vlan_min           = 100
        vlan_max           = 200
        nested_level_count = 0
        subnet             = "none"
      }
      ip_version = {
        ip_version = "v4"
      }
    }]
  }]
}
```

* `vlan_min` (Number, **Required**) – Lower bound (inclusive), `1` to `4094`.
* `vlan_max` (Number, Optional) – Upper bound (inclusive), `1` to `4094`. The provider rejects values below `vlan_min`, but accepts `vlan_max == vlan_min`.
* `nested_level_count` (Number, Optional, default `0`, range `0–4`) – Which VLAN header to inspect. `0` means any matching VLAN header, `1` outermost, `2` second, `3` third, `4` fourth.
* `subnet` (String, Optional, default `"none"`) – `"none"`, `"even"`, or `"odd"`. `"even"` and `"odd"` require `vlan_max`.

Validation notes:

* VLAN ID `0` is not accepted by this provider schema.
* Values outside `1..4094` fail validation.
* `vlan_max < vlan_min` fails with an `Invalid VLAN range` error.
* `subnet = "even"` or `"odd"` without `vlan_max` fails with an `Invalid VLAN subnet` error.

---

### `vntag_dst_vif_id`

Matches the VN-Tag destination VIF ID. Use the Terraform block name `vntag_dst_vif_id`; the FM internal type is `vntagDvifId`.

Use this condition only for traffic carrying VN-Tag metadata. The provider schema exposes the block on traffic-map rules, but FM-side acceptance still depends on receiving compatible traffic and platform support.

Exact-value example:

```hcl
vntag_dst_vif_id = {
  vif_min             = 100
  nested_level_count  = 0
}
```

Range example:

```hcl
vntag_dst_vif_id = {
  vif_min             = 100
  vif_max             = 120
  nested_level_count  = 0
  subnet              = "none"
}
```

Complete rule example:

```hcl
resource "gigamon_traffic_map" "vntag_destination_vif" {
  monitoring_session_id = gigamon_monitoring_session.ms.id
  name                  = "vntag-destination-vif-map"

  rule_sets = [{
    rule_set_id = "1"
    priority    = 1
    aep_id      = 13
    pass_rules = [{
      rule_id = 1
      vntag_dst_vif_id = {
        vif_min            = 100
        vif_max            = 120
        nested_level_count = 0
        subnet             = "none"
      }
      ip_version = {
        ip_version = "v4"
      }
    }]
  }]
}
```

* `vif_min` (Number, **Required**) – Lower bound (inclusive), `0` to `16384`.
* `vif_max` (Number, Optional) – Upper bound (inclusive), `0` to `16384`. This provider requires `vif_max > vif_min` when a range is used.
* `nested_level_count` (Number, Optional, default `0`, range `0–3`) – Which VN-Tag header to inspect. `0` means any matching level.
* `subnet` (String, Optional, default `"none"`) – `"none"`, `"even"`, or `"odd"`. `"even"` and `"odd"` require `vif_max`.

Validation notes:

* Missing `vif_min` fails schema validation.
* Values outside `0..16384` fail validation.
* `vif_max <= vif_min` fails with an `Invalid VN-Tag destination VIF range` error.
* `subnet = "even"` or `"odd"` without `vif_max` fails with an `Invalid VN-Tag destination VIF subnet` error.

---

### `vntag_src_vif_id`

Matches the VN-Tag source VIF ID. Use the Terraform block name `vntag_src_vif_id`; the FM internal type is `vntagSvifId`.

Use this condition only for traffic carrying VN-Tag metadata. The provider schema does not apply an additional map-type or platform validator at plan time.

Exact-value example:

```hcl
vntag_src_vif_id = {
  vif_min             = 200
  nested_level_count  = 0
}
```

Range example:

```hcl
vntag_src_vif_id = {
  vif_min             = 200
  vif_max             = 240
  nested_level_count  = 0
  subnet              = "none"
}
```

Complete rule example:

```hcl
resource "gigamon_traffic_map" "vntag_source_vif" {
  monitoring_session_id = gigamon_monitoring_session.ms.id
  name                  = "vntag-source-vif-map"

  rule_sets = [{
    rule_set_id = "1"
    priority    = 1
    aep_id      = 14
    pass_rules = [{
      rule_id = 1
      vntag_src_vif_id = {
        vif_min            = 200
        nested_level_count = 0
      }
      ip_version = {
        ip_version = "v4"
      }
    }]
  }]
}
```

* `vif_min` (Number, **Required**) – Lower bound (inclusive), `0` to `4096`.
* `vif_max` (Number, Optional) – Upper bound (inclusive), `0` to `4096`. This provider requires `vif_max > vif_min` when a range is used.
* `nested_level_count` (Number, Optional, default `0`, range `0–3`) – Which VN-Tag header to inspect. `0` means any matching level.
* `subnet` (String, Optional, default `"none"`) – `"none"`, `"even"`, or `"odd"`. `"even"` and `"odd"` require `vif_max`.

Validation notes:

* Values outside `0..4096` fail validation.
* `vif_max <= vif_min` fails with an `Invalid VN-Tag source VIF range` error.
* `subnet = "even"` or `"odd"` without `vif_max` fails with an `Invalid VN-Tag source VIF subnet` error.

---

### `vntag_vif_list_id`

Matches the VN-Tag VIF List ID. Use the Terraform block name `vntag_vif_list_id`; the FM internal type is `vntagVifListId`.

Use this condition only for traffic carrying VN-Tag metadata. This provider release accepts VIF List ID values from `0` through `16384`.

Exact-value example:

```hcl
vntag_vif_list_id = {
  vif_min             = 4096
  nested_level_count  = 0
}
```

Range example:

```hcl
vntag_vif_list_id = {
  vif_min             = 4096
  vif_max             = 4200
  nested_level_count  = 0
  subnet              = "none"
}
```

Complete rule example:

```hcl
resource "gigamon_traffic_map" "vntag_vif_list" {
  monitoring_session_id = gigamon_monitoring_session.ms.id
  name                  = "vntag-vif-list-map"

  rule_sets = [{
    rule_set_id = "1"
    priority    = 1
    aep_id      = 15
    pass_rules = [{
      rule_id = 1
      vntag_vif_list_id = {
        vif_min            = 4096
        nested_level_count = 0
      }
      ip_version = {
        ip_version = "v4"
      }
    }]
  }]
}
```

* `vif_min` (Number, **Required**) – Lower bound (inclusive), `0` to `16384`.
* `vif_max` (Number, Optional) – Upper bound (inclusive), `0` to `16384`. This provider requires `vif_max > vif_min` when a range is used.
* `nested_level_count` (Number, Optional, default `0`, range `0–3`) – Which VN-Tag header to inspect. `0` means any matching level.
* `subnet` (String, Optional, default `"none"`) – `"none"`, `"even"`, or `"odd"`. `"even"` and `"odd"` require `vif_max`.

Validation notes:

* Values outside `0..16384` fail validation.
* `vif_max <= vif_min` fails with an `Invalid VN-Tag VIF list range` error.
* `subnet = "even"` or `"odd"` without `vif_max` fails with an `Invalid VN-Tag VIF list subnet` error.

---

### `vxlan_id`

Matches the VXLAN Network Identifier (VNI). Use the Terraform block name `vxlan_id`; the FM internal type is `vxlanId`.

This provider release does not expose a `pos` or `nested_level_count` field for VXLAN matching. VXLAN matching is therefore configured only by value range and subset. Use this condition only where the packet stream can carry VXLAN encapsulation and the target FM platform supports VXLAN-aware matching.

Exact-value example:

```hcl
vxlan_id = {
  vxlan_min = 5000
}
```

Range example:

```hcl
vxlan_id = {
  vxlan_min = 5000
  vxlan_max = 5500
  subnet    = "none"
}
```

Complete rule example:

```hcl
resource "gigamon_traffic_map" "vxlan_id" {
  monitoring_session_id = gigamon_monitoring_session.ms.id
  name                  = "vxlan-id-map"

  rule_sets = [{
    rule_set_id = "1"
    priority    = 1
    aep_id      = 16
    pass_rules = [{
      rule_id = 1
      vxlan_id = {
        vxlan_min = 5000
        subnet    = "none"
      }
      ip_version = {
        ip_version = "v4"
      }
    }]
  }]
}
```

* `vxlan_min` (Number, **Required**) – Lower bound (inclusive), `0` to `16777215`.
* `vxlan_max` (Number, Optional) – Upper bound (inclusive), `0` to `16777215`. This provider requires `vxlan_max > vxlan_min` when a range is used.
* `subnet` (String, Optional, default `"none"`) – `"none"`, `"even"`, or `"odd"`. `"even"` and `"odd"` require `vxlan_max`.

Validation notes:

* VXLAN ID `0` is accepted by the provider schema.
* Values above `16777215` fail validation.
* `vxlan_max <= vxlan_min` fails with an `Invalid VXLAN ID range` error.
* `subnet = "even"` or `"odd"` without `vxlan_max` fails with an `Invalid VXLAN ID subset` error.

---

## Additional Condition Notes

### Terraform Schema Names vs. FM/UI Type Names

The Terraform provider uses schema names that differ from the FM/UI type identifiers:

| Terraform Block Name | FM Type Name | Description |
|---|---|---|
| `gtp_teid` | `gtputeId` | GTP-U Tunnel Endpoint ID |
| `host_name` | `srcHostPrefix` | Source host name prefix |
| `ipv6_flow_label` | `ip6Flow` | IPv6 Flow Label |
| `ipv6_next_header` | `ipv6NextHeader` | IPv6 Next Header protocol |
| `mpls_label` | `mplsLabel` | MPLS label value |
| `port_destination` | `portDst` | Layer-4 destination port |
| `port_source` | `portSrc` | Layer-4 source port |
| `tcp_control` | `tcpCtl` | TCP control / flag bits |
| `vlan` | `vlan` | VLAN ID |
| `vntag_dst_vif_id` | `vntagDvifId` | VN-Tag destination VIF ID |
| `vntag_src_vif_id` | `vntagSvifId` | VN-Tag source VIF ID |
| `vntag_vif_list_id` | `vntagVifListId` | VN-Tag VIF List ID |
| `vxlan_id` | `vxlanId` | VXLAN ID / VNI |

Users should always use the Terraform block names (left column) in their configurations. The FM type names (right column) are internal representations and should never be used in Terraform code.

### Range and Subset Behavior

Most of these conditions expose `*_min` and `*_max` fields plus a `subnet` selector:

* Single-value form: specify only the required minimum field such as `port_min`, `vlan_min`, `vif_min`, or `vxlan_min`.
* Range form: specify both min and max fields. All max fields are inclusive.
* `subnet` values in this provider release are `"none"`, `"even"`, or `"odd"`.
* `"none"` means no parity filter. The provider does not expose an `"all"` subset keyword for these blocks.
* `"even"` and `"odd"` require the corresponding max field.
* `tcp_control` is the exception: it does not expose min/max range fields or `subnet`.

### Position and Header Depth

This provider uses two position field names:

* `nested_level_count` for `gtp_teid`, `port_source`, `tcp_control`, `vlan`, `vntag_dst_vif_id`, `vntag_src_vif_id`, and `vntag_vif_list_id`.
* `pos` for `ipv6_flow_label`, `ipv6_next_header`, `mpls_label`, and `port_destination`.

They serve the same purpose: selecting which header instance to inspect in stacked or tunneled traffic. `0` means any matching level. Higher values select deeper headers. VLAN allows `0..4`; most other blocks allow `0..3`. `vxlan_id` does not expose a position field in this provider release.

### Provider Validation Rules

The Terraform provider enforces the following validation constraints:

**GTP-U TEID (`gtp_teid`):**
- `teid_min` must be a valid 4-byte hex string (8 characters).
- If `teid_max` is set, it must be greater than `teid_min`.
- `subnet` requires `teid_max` when set to `"even"` or `"odd"`.

**Host Name (`host_name`):**
- `src_host_prefix` must be a non-empty string.
- Matching is prefix-based; wildcards are not supported.

**IPv6 Flow Label (`ipv6_flow_label`):**
- `label_min` must be between 0 and 1,048,575.
- If `label_max` is set, it must be between 0 and 1,048,575 and greater than `label_min`.
- `subnet` requires `label_max` when set to `"even"` or `"odd"`.

**IPv6 Next Header (`ipv6_next_header`):**
- `header_min` must be between 0 and 255.
- If `header_max` is set, it must be between 0 and 255 and greater than `header_min`.
- `subnet` requires `header_max` when set to `"even"` or `"odd"`.
- Common protocol values: `6` (TCP), `17` (UDP), `58` (ICMPv6).

**MPLS Label (`mpls_label`):**
- `value_min` must be between 1 and 1,048,576.
- If `value_max` is set, it must be between 1 and 1,048,576 and greater than `value_min`.
- `subnet` requires `value_max` when set to `"even"` or `"odd"`.
- Standard user-label range is 16 to 1,048,575 (0–15 are reserved).

**Port Destination (`port_destination`):**
- `port_min` must be between 0 and 65,535.
- If `port_max` is set, it must be between 0 and 65,535 and greater than `port_min`.
- `subnet` requires `port_max` when set to `"even"` or `"odd"`.
- This condition typically matches TCP and UDP ports; pair with `ipv4_protocol` or `ipv6_next_header` to restrict to specific protocols.

**Port Source (`port_source`):**
- `port_min` must be between 0 and 65,535.
- If `port_max` is set, it must be between 0 and 65,535 and cannot be less than `port_min`.
- `subnet` requires `port_max` when set to `"even"` or `"odd"`.
- Pair this condition with a protocol match so the rule intent is explicit.

**TCP Control (`tcp_control`):**
- `value` is required and must be exactly two hexadecimal characters.
- `mask`, when set, must be exactly two hexadecimal characters.
- `0x` prefixes are rejected by the provider validator.
- Pair this condition with a TCP protocol match.

**VLAN (`vlan`):**
- `vlan_min` must be between 1 and 4094.
- If `vlan_max` is set, it must be between 1 and 4094 and cannot be less than `vlan_min`.
- `subnet` requires `vlan_max` when set to `"even"` or `"odd"`.
- VLAN `0` is not accepted by the provider schema.

**VN-Tag Destination VIF ID (`vntag_dst_vif_id`):**
- `vif_min` must be between 0 and 16,384.
- If `vif_max` is set, it must be between 0 and 16,384 and must be greater than `vif_min`.
- `subnet` requires `vif_max` when set to `"even"` or `"odd"`.

**VN-Tag Source VIF ID (`vntag_src_vif_id`):**
- `vif_min` must be between 0 and 4,096.
- If `vif_max` is set, it must be between 0 and 4,096 and must be greater than `vif_min`.
- `subnet` requires `vif_max` when set to `"even"` or `"odd"`.

**VN-Tag VIF List ID (`vntag_vif_list_id`):**
- `vif_min` must be between 0 and 16,384.
- If `vif_max` is set, it must be between 0 and 16,384 and must be greater than `vif_min`.
- `subnet` requires `vif_max` when set to `"even"` or `"odd"`.

**VXLAN ID (`vxlan_id`):**
- `vxlan_min` must be between 0 and 16,777,215.
- If `vxlan_max` is set, it must be between 0 and 16,777,215 and must be greater than `vxlan_min`.
- `subnet` requires `vxlan_max` when set to `"even"` or `"odd"`.
- No position field is exposed in this provider release.

### Common Configuration Errors

- **Omitting range values:** If `subnet` is set to `"even"` or `"odd"`, the provider will return an error if `*_max` is not also specified.
- **Invalid hex formats:** GTP-U TEID values must be exactly 8 hex characters. Leading zeros are required (e.g., `"00000001"`, not `"1"`).
- **Value bounds:** Out-of-range values (e.g., label > 1,048,575 or port > 65,535) will cause validation errors during `terraform plan`.
- **TCP control formatting:** `tcp_control.value = "0x02"` and `tcp_control.value = "2"` both fail provider validation because the schema requires exactly two hexadecimal characters with no prefix.
- **Range semantics differ by block:** `port_source` and `vlan` allow `max == min`, while the VN-Tag and VXLAN blocks require `max > min`.
- **Mutually exclusive fields:** Do not mix single-value fields (like `teid_min` alone) with range fields in the same block unless explicitly documented as supporting both patterns.

### Provider and FM Applicability Notes

These blocks are documented because this provider release exposes them in the `gigamon_traffic_map` schema and conversion logic. The exact minimum FM version is not encoded in the provider schema, so release-specific FM compatibility still needs to be validated against your target FM build before publication. The provider also does not add plan-time guards for traffic-content prerequisites such as TCP, VXLAN, or VN-Tag presence; those constraints are enforced by the packet stream and by FM at apply time.

---

## Combined Examples

### Example: IPv6 TCP traffic matching multiple conditions

```hcl
resource "gigamon_traffic_map" "ipv6_tcp_traffic" {
  monitoring_session_id = gigamon_monitoring_session.ms.id
  name                  = "ipv6-tcp-traffic-map"
  description           = "Match IPv6 TCP traffic on port 443 (HTTPS) with specific flow label"

  rule_sets = [
    {
      rule_set_id = "1"
      priority    = 1
      aep_id      = 10

      pass_rules = [
        {
          rule_id = 1

          # Match IPv6 header
          ip_version = {
            ip_version = "v6"
          }

          # Match TCP (protocol 6)
          ipv6_next_header = {
            header_min = 6
            pos        = 0
          }

          # Match HTTPS destination port
          port_destination = {
            port_min = 443
            pos      = 0
          }

          # Match specific flow label range
          ipv6_flow_label = {
            label_min = 256
            label_max = 512
            pos       = 0
            subnet    = "none"
          }
        }
      ]
    }
  ]
}
```

### Example: GTP-U traffic filtering

```hcl
resource "gigamon_traffic_map" "gtp_u_filter" {
  monitoring_session_id = gigamon_monitoring_session.ms.id
  name                  = "gtp-u-filter-map"
  description           = "Filter GTP-U traffic by TEID and forward to tool"

  rule_sets = [
    {
      rule_set_id = "1"
      priority    = 1
      aep_id      = 15

      pass_rules = [
        {
          rule_id = 1

          # Match GTP-U TEID in range
          gtp_teid = {
            teid_min           = "00000100"
            teid_max           = "000001FF"
            nested_level_count = 0
            subnet             = "none"
          }
        }
      ]
    }
  ]
}

resource "gigamon_link" "gtp_u_to_tool" {
  monitoring_session_id = gigamon_monitoring_session.ms.id

  source_id     = gigamon_traffic_map.gtp_u_filter.id
  source_aep_id = 15

  dest_id = gigamon_application.analysis_tool.id
}
```

### Example: MPLS traffic with drop rules

```hcl
resource "gigamon_traffic_map" "mpls_drop_rules" {
  monitoring_session_id = gigamon_monitoring_session.ms.id
  name                  = "mpls-traffic-map"
  description           = "Route MPLS traffic; drop reserved labels"

  rule_sets = [
    {
      rule_set_id = "1"
      priority    = 1
      aep_id      = 20

      # Pass all MPLS traffic in normal range
      pass_rules = [
        {
          rule_id = 1
          mpls_label = {
            value_min = 16            # First usable label
            value_max = 1048575       # Last standard label
            pos       = 0
            subnet    = "none"
          }
        }
      ]

      # Drop reserved labels (0–15)
      drop_rules = [
        {
          rule_id = 1
          mpls_label = {
            value_min = 0
            value_max = 15
            pos       = 0
            subnet    = "none"
          }
        }
      ]
    }
  ]
}
```

### Example: Host name matching for request routing

```hcl
resource "gigamon_traffic_map" "host_name_routing" {
  monitoring_session_id = gigamon_monitoring_session.ms.id
  name                  = "host-name-routing-map"
  description           = "Route traffic based on source host name"

  rule_sets = [
    {
      rule_set_id = "1"
      priority    = 1
      aep_id      = 25

      # Route API server traffic
      pass_rules = [
        {
          rule_id = 1
          host_name = {
            src_host_prefix = "api-server"
          }
        },
        # Route web server traffic (OR'd with above)
        {
          rule_id = 2
          host_name = {
            src_host_prefix = "web-srv"
          }
        }
      ]
    }
  ]
}
```

### Example: Multi-condition rule combining IPv6, MPLS, and port matching

```hcl
resource "gigamon_traffic_map" "multi_condition_map" {
  monitoring_session_id = gigamon_monitoring_session.ms.id
  name                  = "multi-condition-map"
  description           = "Advanced traffic map with multiple match conditions"

  rule_sets = [
    {
      rule_set_id = "1"
      priority    = 1
      aep_id      = 30

      pass_rules = [
        {
          rule_id = 1

          # AND conditions: Match IPv6 traffic
          ip_version = {
            ip_version = "v6"
          }

          # Match MPLS outer label
          mpls_label = {
            value_min = 100
            value_max = 200
            pos       = 0
            subnet    = "none"
          }

          # Match destination port range
          port_destination = {
            port_min = 5000
            port_max = 6000
            pos      = 0
            subnet   = "none"
          }
        },

        # Alternative pass rule: GTP-U traffic (OR'd with above)
        {
          rule_id = 2
          gtp_teid = {
            teid_min = "00000001"
            subnet   = "none"
          }
        }
      ]

      # Drop reserved MPLS labels in this traffic class
      drop_rules = [
        {
          rule_id = 1
          mpls_label = {
            value_min = 0
            value_max = 15
            pos       = 0
          }
        }
      ]
    }
  ]
}

---

## Attribute Reference

The following attributes are exported in addition to all arguments above:

* `id` (String) – Typed map ID used for linking and lifecycle operations.
  Format: `map::trafficMap::<uuid>`. Pass this as `source_id` in `gigamon_link`.

---

## ESXi VM Selection

On ESXi, you can restrict a traffic map to capture traffic only from specific VMs by their MAC addresses.
This is configured using the separate `gigamon_esxi_vm_selection` resource.

See `gigamon_esxi_vm_selection` for full documentation.

---

## Validation Notes

- `rule_sets` must contain between **1** and **5** items.
- `rule_set_id` must be a string between `"1"` and `"5"` (exactly one character).
- `priority` must be between **1** and **5**.
- `aep_id` must be between **2** and **63**.
- Each `rule_set` must contain at least one of `pass_rules` or `drop_rules`.
- When `pass_rules` is specified it must contain at least **1** rule.
- When `drop_rules` is specified it must contain at least **1** rule.
- `description`, if set, must be non-empty.

---

## Import

Import is **not supported** for `gigamon_traffic_map`.
