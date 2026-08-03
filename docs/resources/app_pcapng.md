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

The **PCapNG application** adds PCapNG processing to a monitoring session and can be used as a link destination in session topology.

Use this resource to:

- set app mode (`primary` or `secondary`),
- optionally enable domain classification in `primary` mode,
- configure domain-related controls when classification is enabled.

Each `gigamon_app_pcapng` belongs to a single monitoring session.

---

## Example Usage

### Minimal PCapNG configuration

```hcl
resource "gigamon_app_pcapng" "app" {
  monitoring_session_id = gigamon_monitoring_session.ms.id
  alias                 = "pcapng-secondary"
  app_mode              = "secondary"
}
```

### Practical PCapNG configuration

```hcl
resource "gigamon_app_pcapng" "app" {
  monitoring_session_id = gigamon_monitoring_session.ms.id
  alias                 = "pcapng-primary"
  app_mode              = "primary"
  domain_classification = true
  flow_timeout          = 600
  domain_table_alias    = "domain-map-1"
}
```

### Topology link usage example

```hcl
resource "gigamon_link" "map_to_pcapng" {
  monitoring_session_id = gigamon_monitoring_session.ms.id
  source_id             = gigamon_traffic_map.web.id
  source_aep_id         = 2
  dest_id               = gigamon_app_pcapng.app.id
}
```

Topology notes:

- `source_aep_id` must match the source map rule set `aep_id`.
- `dest_id` uses the typed ID exported by `gigamon_app_pcapng`.
- Keep all linked resources in the same monitoring session.

---

## Argument Reference

### Required

- **`monitoring_session_id`** (String)
  Monitoring Session where this application is configured.
  Use the monitoring session ID so capture behavior is bound to the intended traffic pipeline.

- **`alias`** (String)
  Alias for the PCapNG application.
  Must be non-empty and may contain only alphanumeric characters, `_`, and `-`.

### Optional

- **`app_mode`** (String)
  App mode for PCapNG.
  Allowed values: `primary`, `secondary`.
  Default: `secondary`.

- **`domain_classification`** (Boolean)
  Enable domain classification behavior.
  Default: `false`.
  Can be set to `true` only when `app_mode = "primary"`.

- **`domain_table_alias`** (String)
  Domain table alias.
  Configurable only when `app_mode = "primary"` and `domain_classification = true`.

- **`flow_timeout`** (Number)
  Flow timeout used with domain classification.
  Default: `660`.
  Configurable only when `app_mode = "primary"` and `domain_classification = true`.
  Valid range (when configurable): `360..1860`.

### Computed/Internal

- **`name`** (String, Computed)
  Internal FM app name. Always `pcapng`.

---

## Mode-Specific Behavior

### `secondary` mode (default)

- `domain_classification` must remain `false`.
- `domain_table_alias` must not be configured.
- `flow_timeout` is not configurable in this mode.

### `primary` mode

- `domain_classification` can be `false` or `true`.
- When `domain_classification = false`, `domain_table_alias` and custom `flow_timeout` are not allowed.
- When `domain_classification = true`, you may set `domain_table_alias` and `flow_timeout`.

---

## Mode Behavior + Dependency Matrix

| Condition | Required/Allowed | Not Allowed / Enforced Behavior |
|---|---|---|
| `app_mode = secondary` | `domain_classification` must stay `false` | `domain_classification = true` is rejected |
| `app_mode = secondary` | N/A | `domain_table_alias` and custom `flow_timeout` are rejected |
| `app_mode = primary` and `domain_classification = false` | `domain_table_alias` omitted; `flow_timeout` left at default | Setting `domain_table_alias` or custom `flow_timeout` is rejected |
| `app_mode = primary` and `domain_classification = true` | `domain_table_alias` and `flow_timeout` may be set | `flow_timeout` outside `360..1860` is rejected |
| `app_mode` omitted | Defaults to `secondary` | N/A |

---

## Validation and Dependency Notes

- `monitoring_session_id` is required and changing it recreates the resource.
- `alias` is required; regex: `^[A-Za-z0-9_-]+$`.
- `app_mode` defaults to `secondary`; only `primary` and `secondary` are allowed.
- `domain_classification` defaults to `false`.
- `domain_table_alias` is conditionally allowed only with `app_mode = "primary"` and `domain_classification = true`.
- `flow_timeout` defaults to `660` and is conditionally configurable only with `app_mode = "primary"` and `domain_classification = true`.
- When configurable, `flow_timeout` must be between `360` and `1860`.

- Provider payload behavior: domain fields are sent only in `primary` mode; `domain_table_alias` and `flow_timeout` are sent only when `domain_classification = true`.

---

## Attributes Reference

In addition to the arguments above, this resource exports:

- **`id`** (String)
  Typed application identifier in the form `app::PCapNG::<uuid>`.

---

## Import

Import using the monitoring session ID and the raw application UUID:

```shell
terraform import gigamon_app_pcapng.example "<monitoring_session_id>::<raw_uuid>"
```
