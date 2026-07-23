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
