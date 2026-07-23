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

The **5G-SBI application** monitors Service-Based Interface traffic in 5G environments.

Use this resource to:

- set the SBI role mode,
- declare protocol handlers,
- optionally configure authentication material.

Each `gigamon_app_5gsbi` belongs to a single monitoring session.

---

## Example Usage

### Minimal 5G-SBI configuration

```hcl
resource "gigamon_app_5gsbi" "app" {
  monitoring_session_id = gigamon_monitoring_session.ms.id
  sbi_mode              = "nrf"

  protocol_handlers = ["http", "https"]

  authentication = {
    enabled   = false
    cert_path = "/etc/gigamon/certs/sbi.crt"
    key_path  = "/etc/gigamon/certs/sbi.key"
  }
}
```

---

## Argument Reference

### Required

- **`monitoring_session_id`** (String)
  Monitoring Session where this application is configured.
  This ensures the SBI app is scoped to one monitoring workflow.

- **`sbi_mode`** (String)
  SBI role mode.
  Valid values: `nrf`, `udm`, `amf`.
  Choose the value that matches the network function role you want to model and inspect.

### Optional

- **`protocol_handlers`** (List of String)
  Protocol handlers.
  Valid values include `http`, `https`, `grpc`.
  Use the minimal required set for your deployment to keep parsing focused and predictable.

- **`authentication`** (Block)
  Authentication configuration.
  Use this block to describe certificate/key material handling for secured SBI exchanges.

### `authentication` block

- **`enabled`** (Boolean)
  Enable authentication.
  Keep `false` for non-authenticated testing environments; set `true` when certificate-based handling is required.

- **`cert_path`** (String)
  Certificate file path.
  Provide the path expected by the application runtime in your deployment context.

- **`key_path`** (String)
  Private key file path.
  Keep key and cert paths aligned to avoid runtime auth handshake failures.

---

## Attributes Reference

In addition to the arguments above, this resource exports:

- **`id`** (String)
  Typed application identifier in the form `app::<type>::<uuid>`.

---

## Import

Import using the monitoring session ID and the raw application UUID:

```shell
terraform import gigamon_app_5gsbi.example "<monitoring_session_id>::<raw_uuid>"
```
