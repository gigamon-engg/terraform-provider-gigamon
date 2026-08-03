---
page_title: "AMI Application"
subcategory: "Applications"
description: "Manage the AMI application in Gigamon FM."
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

## Resource: `gigamon_app_ami`

The **AMI application** (Application Metadata Intelligence) performs deep packet inspection and exports structured metadata records from mirrored traffic.

Each AMI instance belongs to one monitoring session and is typically used in topology as:

- `gigamon_traffic_map` (or other source) -> `gigamon_app_ami`
- `gigamon_app_ami` -> downstream destination (for example, tunnel/tool)

AMI exporter entries are anchored to AEP IDs. Those AEP IDs are what you use when AMI acts as a link source.

---

## Example Usage

### Minimal AMI configuration

```hcl
resource "gigamon_app_ami" "app" {
  monitoring_session_id = gigamon_monitoring_session.ms.id
  alias                 = "ami-min"

  app_metadata = {
    flow_behavior    = "bidir"
    multi_collect    = true
    aggregate_mode   = false
    observ_domain_id = 0

    timeout = {
      idle = 300
    }
  }
}
```

### Practical AMI exporter configuration

```hcl
resource "gigamon_app_ami" "app" {
  monitoring_session_id = gigamon_monitoring_session.ms.id
  alias                 = "ami-export"

  app_metadata = {
    flow_behavior    = "bidir"
    multi_collect    = true
    aggregate_mode   = false
    observ_domain_id = 0
    dpi_inject_limit = 30

    timeout = {
      idle = 300
    }

    exporters = [
      {
        aep_id = 2
        name   = "ami-exporter-2"

        exporter_config = {
          type = "cef"
          max_pkt_size = 1500

          cef = {
            active_timeout   = 60
            inactive_timeout = 15
            record_type      = "segregated"
          }

          app_profile_config = [
            {
              applications = []
              type         = "export"
            }
          ]
        }
      }
    ]
  }
}
```

### Topology integration example

```hcl
# Upstream: send map output to AMI
resource "gigamon_link" "map_to_ami" {
  monitoring_session_id = gigamon_monitoring_session.ms.id
  source_id             = gigamon_traffic_map.web.id
  source_aep_id         = 2
  dest_id               = gigamon_app_ami.app.id
}

# Downstream: send AMI exporter output to a destination
resource "gigamon_link" "ami_to_tunnel" {
  monitoring_session_id = gigamon_monitoring_session.ms.id
  source_id             = gigamon_app_ami.app.id
  source_aep_id         = 2
  dest_id               = gigamon_tunnel.ami_sink.id
}
```

Topology notes:

- In the first link, `source_aep_id` belongs to the source map rule set.
- In the second link, `source_aep_id` must match `app_metadata.exporters[*].aep_id` on AMI.
- Keep all linked resources in the same monitoring session.

---

## Argument Reference

### Required

- **`monitoring_session_id`** (String)
  Monitoring Session where this application is configured.
  Use your session resource ID so this AMI app participates in the intended traffic pipeline.

- **`alias`** (String)
  Alias for the AMI application.
  Choose a stable and descriptive name because this value is typically used in UI/operator workflows.

### Optional

- **`description`** (String)
  Description for the AMI application.
  Use this to capture intent (for example, tool target, tenant, or environment).

- **`app_metadata`** (Block)
  Typed AMI metadata configuration.
  This is the primary block for AMI behavior tuning and exporter modeling.

---

## Nested Field Guidance

### `app_metadata` (Object)

Core fields:

- **`flow_behavior`** (String, Optional, default: `bidir`)
  Flow behavior mode for AMI processing. Common value is `bidir`.
  Important compatibility rule: when `flow_behavior = "bidir"`, netflow exporter version `v5` or `v9` is rejected.

- **`timeout`** (Object, Optional)
  - **`idle`** (Number, Optional, default: `300`, min: `1`)

- **`multi_collect`** (Boolean, Optional, default: `true`)

- **`aggregate_mode`** (Boolean, Optional, default: `false`)

- **`observ_domain_id`** (Number, Optional, default: `0`, min: `0`)

- **`dpi_inject_limit`** (Number, Optional, default: `30`)
  Valid values: `0` or range `20..50`.

- **`match`** (Object, Optional)
  Nested match selectors for `ipv4`, `ipv6`, `transport`, and `datalink`.
  Important constraint: `app_metadata.match.ipv4.next_header` is not supported.

- **`exporters`** (List of Objects, Optional)
  Export pipeline definitions for AMI output. Each exporter has an AEP and exporter profile.

- **`persist_profile_config`** (Object, Optional)
  Optional persist profile definition.
  - `alias` is required if this block is set.
  - `type` defaults to `persist`.

### `app_metadata.exporters[*]` (Object)

- **`aep_id`** (Number, Required)
  Valid range: `2..63`.

- **`name`** (String, Required)
  Must be non-empty.

- **`exporter_config`** (Object, Required)
  Contains exporter profile type and settings.

### `app_metadata.exporters[*].exporter_config` (Object)

- **`type`** (String, Optional, default: `cef`)
  Allowed values: `cef`, `netflow`.

- **`max_pkt_size`** (Number, Required)
  Valid values: `0` or range `1280..9001`.

- **`app_profile_config`** (List of Objects, Optional)
  Application profile selectors for metadata export.
  Common defaults inside each profile include:
  - `type` default: `export`
  - `application_id` default: `true`
  - `family_id` default: `false`
  - `tag_id` default: `true`

- **`cef`** (Object, Conditional)
  Required when `type = "cef"`. Must not be set when `type = "netflow"`.
  - `active_timeout` default `60`, range `1..604800`
  - `inactive_timeout` default `15`, range `1..604800`
  - `record_type` default `segregated`, allowed `segregated|cohesive`

- **`netflow`** (Object, Conditional)
  Required when `type = "netflow"`. Must not be set when `type = "cef"`.
  - `active_timeout` default `60`, range `1..604800`
  - `inactive_timeout` default `15`, range `1..604800`
  - `record_type` default `segregated`, allowed `segregated|cohesive`
  - `template_refresh` default `60`, range `1..216000`
  - `version` required, allowed `ipfix|v5|v9`
  - extra rule: `flow_behavior = "bidir"` cannot be combined with netflow version `v5` or `v9`

---

## Validation and Behavior Notes

- `monitoring_session_id` is required and changing it recreates this resource.
- `alias` is required and must be non-empty.
- `description` is optional; if set, it must be non-empty.
- AMI requires `scale_unit` to already be configured on the selected monitoring session.
- `app_metadata` is optional in schema, but real deployments should define it explicitly.
- Exporter type consistency is enforced:
  - `type = "cef"` requires `cef` and rejects `netflow`.
  - `type = "netflow"` requires `netflow` and rejects `cef`.
- `app_metadata.match.ipv4.next_header` and `app_profile_config[*].ipv4.next_header` are rejected.

---

## Mode Behavior + Dependency Matrix

| Condition | Required/Allowed | Not Allowed / Enforced Behavior |
|---|---|---|
| `app_metadata.exporters[*].exporter_config.type = "cef"` | `cef` block required | `netflow` block must not be set |
| `app_metadata.exporters[*].exporter_config.type = "netflow"` | `netflow` block required; `netflow.version` required (`ipfix`, `v5`, `v9`) | `cef` block must not be set |
| `flow_behavior = "bidir"` with netflow exporter | `netflow.version = ipfix` | `netflow.version = v5` or `v9` is rejected |
| Any exporter entry | `exporter_config` required; `max_pkt_size` required (`0` or `1280..9001`) | Missing `exporter_config` is rejected |
| `persist_profile_config` present | `persist_profile_config.alias` required | Missing alias is rejected |
| IPv4 next-header toggles | Use `ipv6.next_header` when needed | `app_metadata.match.ipv4.next_header` and `app_profile_config[*].ipv4.next_header` are rejected |

### `app_metadata` block overview

The `app_metadata` block contains AMI behavior and export modeling, including:

- flow behavior and aggregation controls,
- exporter definitions and exporter profiles,
- optional match and persistence profile sections,
- timeout settings.

For nested field details, use the provider schema output or generated schema sections maintained from `tfplugindocs`.

Practical guidance:

- Start with a single exporter profile and expand only after baseline validation succeeds.
- Keep timeout and flow behavior values conservative first, then tune from observed traffic characteristics.
- When using multiple exporters, keep naming and AEP mapping consistent to simplify troubleshooting.

---

## Attributes Reference

In addition to the arguments above, this resource exports:

- **`id`** (String)
  Typed application identifier in the form `app::ami::<uuid>`.

---

## Import

Import using the monitoring session ID and the raw application UUID:

```shell
terraform import gigamon_app_ami.example "<monitoring_session_id>::<raw_uuid>"
```

Example:

```shell
terraform import gigamon_app_ami.example "monitoringSession::vmware::aaa-bbb-ccc::11111111-2222-3333-4444-555555555555"
```
