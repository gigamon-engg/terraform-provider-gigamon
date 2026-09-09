---
page_title: "App Viz Application"
subcategory: "Applications"
description: "Manage the App Viz application in Gigamon FM."
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

## Resource: `gigamon_app_viz`

The **App Viz application** provides real-time visibility by sampling traffic in a monitoring session and exporting telemetry to monitor endpoints.

In most deployments, `gigamon_app_viz` is used as a **destination application** in monitoring session topology:

- traffic is classified by one or more maps,
- maps forward matched traffic through an AEP,
- `gigamon_link` connects that AEP output to App Viz.

Use this resource to:

- define the App Viz alias and control action state,
- choose internal or external management interface,
- tune exporter monitor timeout behavior.

Each `gigamon_app_viz` belongs to a single monitoring session.

---

## Example Usage

### Minimal App Viz configuration

```hcl
resource "gigamon_app_viz" "app" {
  monitoring_session_id = gigamon_monitoring_session.ms.id
  alias                 = "appviz-main"
}
```

### Practical App Viz configuration

```hcl
resource "gigamon_app_viz" "app" {
  monitoring_session_id = gigamon_monitoring_session.ms.id
  alias                 = "appviz-prod"
  description           = "App visibility for production traffic"
  action                = true
  mgmt_interface        = "internal"

  exporter_config = {
    monitor = {
      timeout = 300
    }
  }
}
```

### Topology linkage example (`gigamon_traffic_map` -> `gigamon_app_viz`)

```hcl
resource "gigamon_link" "map_to_appviz" {
  monitoring_session_id = gigamon_monitoring_session.ms.id
  source_id             = gigamon_traffic_map.web.id
  source_aep_id         = 2
  dest_id               = gigamon_app_viz.app.id
}
```

Topology notes:

- `source_aep_id` must match the `aep_id` of the source map rule set that should feed App Viz.
- `dest_id` should be the typed App Viz ID (`app::appviz::<uuid>`) exported by this resource.
- Keep `gigamon_link.monitoring_session_id` aligned with the session used by both source and destination resources.

---

## Argument Reference

### Required

- **`monitoring_session_id`** (String)
  Monitoring Session where this application is configured.
  This ties App Viz behavior to one monitoring session lifecycle.

- **`alias`** (String)
  Alias for the App Viz application.
  Use a stable, descriptive name for easier operations and topology readability.

### Optional

- **`description`** (String)
  Description for the App Viz application.
  Useful for recording purpose, environment, or owning team.

- **`action`** (Boolean)
  Enable or disable App Viz action behavior.
  Default is `true`.
  Set to `false` to preserve config in Terraform while pausing active behavior.

- **`mgmt_interface`** (String)
  Management interface.
  Valid values: `internal`, `external`.
  Default is `internal`.
  Choose the interface path that matches your routing/security model for telemetry export.

- **`exporter_config`** (Block)
  Exporter configuration for App Viz.
  This block is **required** by the provider.
  Use this block for monitor/export timing behavior.

### `exporter_config` block

- **`monitor`** (Block)
  Monitor-specific export behavior.
  Encapsulates monitor-side runtime controls for telemetry emission.

### `exporter_config.monitor` block

- **`timeout`** (Number)
  Monitor timeout in seconds.
  For App Viz this is fixed at `300` seconds by schema validation.
  The provider default is `300` and other values are rejected.

---

## Behavior and Validation Notes

- `monitoring_session_id` is required and changing it forces recreation of `gigamon_app_viz`.
- `alias` is required and must be non-empty.
- `description` defaults to an empty string when omitted.
- `action` defaults to `true` when omitted.
- `mgmt_interface` defaults to `internal` and only accepts `internal` or `external`.
- `exporter_config` is required and cannot be null.
- `exporter_config.monitor.timeout` defaults to `300` and is validated to exactly `300`.
- The exported `id` is a typed ID in the form `app::appviz::<uuid>`, which is used as `dest_id` in `gigamon_link`.

---

## Mode Behavior + Dependency Matrix

```text
Condition                               | Required / Allowed                    | Not Allowed / Enforced Behavior
--------------------------------------- | ------------------------------------- | -----------------------------------------------
Always                                  | exporter_config must be present       | Null or omitted exporter_config is rejected
exporter_config.monitor omitted         | Provider supplies default monitor     | N/A
exporter_config.monitor.timeout set     | Effective value must be 300           | Values other than 300 are rejected
mgmt_interface omitted                  | Defaults to internal                  | N/A
mgmt_interface set                      | internal or external                  | Any other value is rejected
```

---

## Attributes Reference

In addition to the arguments above, this resource exports:

- **`id`** (String)
  Typed application identifier in the form `app::appviz::<uuid>`.

---

## Import

Import using the monitoring session ID and the raw application UUID:

```shell
terraform import gigamon_app_viz.example "<monitoring_session_id>::<raw_uuid>"
```
