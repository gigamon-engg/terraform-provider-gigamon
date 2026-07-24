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

  alias          = "app_viz1"
  description    = ""
  action         = true
  mgmt_interface = "internal"

  exporter_config = {
    monitor = {
      timeout = 300
    }
  }
}
```

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
  Set to `false` to preserve config in Terraform while pausing active behavior.

- **`mgmt_interface`** (String)
  Management interface.
  Valid values: `internal`, `external`.
  Choose the interface path that matches your routing/security model for telemetry export.

- **`exporter_config`** (Block)
  Exporter configuration for App Viz.
  Use this block for monitor/export timing behavior.

### `exporter_config` block

- **`monitor`** (Block)
  Monitor-specific export behavior.
  Encapsulates monitor-side runtime controls for telemetry emission.

### `exporter_config.monitor` block

- **`timeout`** (Number)
  Monitor timeout in seconds.
  Increase for slower downstream systems; decrease for faster failover behavior.

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
