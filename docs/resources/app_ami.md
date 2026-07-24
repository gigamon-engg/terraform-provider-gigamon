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

Each AMI instance belongs to one monitoring session and can be linked to downstream outputs such as tunnels and maps.

---

## Example Usage

### Minimal AMI with CEF exporter

```hcl
resource "gigamon_app_ami" "minimal" {
  monitoring_session_id = gigamon_monitoring_session.ms.id
  alias                 = "ami-min"

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
