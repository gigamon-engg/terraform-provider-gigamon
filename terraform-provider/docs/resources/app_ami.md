---
layout: 'gigamon'
page_title: 'Gigamon: gigamon_app_ami'
subcategory: 'Applications'
description: |-
    Manages an AMI (Application Metadata Intelligence) instance on a GigaVUE-FM monitoring session.
---

# gigamon_app_ami (Resource)

**AMI (Application Metadata Intelligence)** performs deep-packet inspection on mirrored traffic and exports structured application metadata records to downstream collectors.

Each AMI instance is bound to a single monitoring session and exports records through one or more _exporters_ that each target a specific AEP output endpoint.

## Example Usage

### Minimal AMI with CEF exporter

```terraform
# Minimal typed AMI example.
# This example uses an existing monitoring session that is imported.

terraform {
  required_providers {
    gigamon = {
      source = "local/gigamon/gigamon"
    }
  }
}

provider "gigamon" {
  fm_address  = "10.114.50.20"
  skip_verify = true
  api_token   = "eyJhbGciOiJIUzI1NiJ9.eyJ0b2tlbklkIjoiNDYxMDgyNDM1NDEzOTY5NCIsInN1YiI6Imdtb2hhbiIsImlhdCI6MTc4MTUxMjMyMywiZXhwIjoxNzg0MTA0MzIzfQ.mlP_dTGCIB42Y3PjpwoH6iKdlxFjDPktDBmdl1WFDhU"
}

# Store your existing monitoring session ID locally
# Format: monitoringSession::<platform>::<uuid>
locals {
  monitoring_session_id = "monitoringSession::vmware::0ddfdd2d-2a27-4abc-ae39-3432601bcd53"
}

resource "gigamon_app_ami" "minimal" {
  monitoring_session_id = local.monitoring_session_id
  alias                 = "ami-min"

  app_metadata = {
    flow_behavior    = "bidir"
    multi_collect    = true
    aggregate_mode   = false
    observ_domain_id = 0

    timeout = {
      idle = 300
    }

    dpi_inject_limit = 30

    exporters = [
      {
        aep_id = 2
        name   = "ami-exporter-2"

        exporter_config = {
          type = "cef"

          app_profile_config = [
            {
              applications = []
              type = "export"
            }
          ]
        }
      }
    ]

  }
}

resource "gigamon_tunnel_out" "ami_udp_out" {
  monitoring_session_id = local.monitoring_session_id
  alias                 = "ami-udp-out-min"
  description           = "UDP egress tunnel for AMI minimal example"
  remote_ip             = "198.51.100.10"

  udp = {
    source_port      = 50000
    destination_port = 50001
  }
}

resource "gigamon_link" "ami_to_udp" {
  monitoring_session_id = local.monitoring_session_id
  source_id             = gigamon_app_ami.minimal.id
  dest_id               = gigamon_tunnel_out.ami_udp_out.id

  depends_on = [
    gigamon_app_ami.minimal,
    gigamon_tunnel_out.ami_udp_out,
  ]
}
```

### End-to-end AMI + AFI traffic map

```terraform
# Complete end-to-end example showing AMI and traffic map resources.
# References an existing monitoring session by its TypedID (no management of the session itself).

# Create an AMI application using the monitoring session
resource "gigamon_app_ami" "example" {
  monitoring_session_id = local.monitoring_session_id
  alias                 = "ami-example"
  description           = "Example AMI application with typed configuration"

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
              type = "export"
            }
          ]
        }
      }
    ]

  }
}

resource "gigamon_tunnel_out" "ami_udp_out" {
  monitoring_session_id = local.monitoring_session_id
  alias                 = "ami-udp-out-example"
  description           = "UDP egress tunnel for AMI end-to-end example"
  remote_ip             = "198.51.100.10"

  udp = {
    source_port      = 50000
    destination_port = 50001
  }
}

resource "gigamon_link" "ami_to_udp" {
  monitoring_session_id = local.monitoring_session_id
  source_id             = gigamon_app_ami.example.id
  dest_id               = gigamon_tunnel_out.ami_udp_out.id

  depends_on = [
    gigamon_app_ami.example,
    gigamon_tunnel_out.ami_udp_out,
  ]
}

# Create a traffic map (AFI) using the monitoring session
resource "gigamon_traffic_map" "example" {
  monitoring_session_id = local.monitoring_session_id
  name                  = "afi-example"
  description           = "Example traffic map with typed ASF configuration"

  rule_sets = [
    {
      rule_set_id = "1"
      aep_id      = 2
      priority    = 1

      pass_rules = [
        {
          rule_id = 1
          ip_version = {
            ip_version = "v4"
          }
        }
      ]
    }
  ]

  asf = {
    asf_profile_config = {
      session_fields = [
        {
          pos  = 2
          type = "fiveTuple"
        }
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
}

# Outputs for verification
output "ami_id" {
  description = "The created AMI application typed ID"
  value       = gigamon_app_ami.example.id
}

output "traffic_map_id" {
  description = "The created traffic map typed ID"
  value       = gigamon_traffic_map.example.id
}
```

## Argument Reference

The following arguments are supported:

<!-- schema generated by tfplugindocs -->

## Schema

### Required

-   `alias` (String) Alias for the AMI application.
-   `monitoring_session_id` (String) Monitoring Session ID on which this AMI application is created.

### Optional

-   `app_metadata` (Attributes) Typed AMI appMetadata configuration. (see [below for nested schema](#nestedatt--app_metadata))
-   `description` (String) Description for the AMI application.

### Read-Only

-   `id` (String) Typed ID of this AMI app instance.

<a id="nestedatt--app_metadata"></a>

### Nested Schema for `app_metadata`

Optional:

-   `aggregate_mode` (Boolean)
-   `dpi_inject_limit` (Number)
-   `exporters` (Attributes List) (see [below for nested schema](#nestedatt--app_metadata--exporters))
-   `flow_behavior` (String)
-   `match` (Attributes) (see [below for nested schema](#nestedatt--app_metadata--match))
-   `multi_collect` (Boolean)
-   `observ_domain_id` (Number)
-   `persist_profile_config` (Attributes) (see [below for nested schema](#nestedatt--app_metadata--persist_profile_config))
-   `timeout` (Attributes) (see [below for nested schema](#nestedatt--app_metadata--timeout))

<a id="nestedatt--app_metadata--exporters"></a>

### Nested Schema for `app_metadata.exporters`

Required:

-   `aep_id` (Number)
-   `exporter_config` (Attributes) (see [below for nested schema](#nestedatt--app_metadata--exporters--exporter_config))
-   `name` (String)

<a id="nestedatt--app_metadata--exporters--exporter_config"></a>

### Nested Schema for `app_metadata.exporters.exporter_config`

Optional:

-   `app_profile_config` (Attributes List) (see [below for nested schema](#nestedatt--app_metadata--exporters--exporter_config--app_profile_config))
-   `cef` (Attributes) (see [below for nested schema](#nestedatt--app_metadata--exporters--exporter_config--cef))
-   `max_pkt_size` (Number)
-   `type` (String)

<a id="nestedatt--app_metadata--exporters--exporter_config--app_profile_config"></a>

### Nested Schema for `app_metadata.exporters.exporter_config.app_profile_config`

Optional:

-   `application_id` (Boolean)
-   `applications` (Attributes List) (see [below for nested schema](#nestedatt--app_metadata--exporters--exporter_config--app_profile_config--applications))
-   `counter` (Attributes) (see [below for nested schema](#nestedatt--app_metadata--exporters--exporter_config--app_profile_config--counter))
-   `family_id` (Boolean)
-   `ipv4` (Attributes) (see [below for nested schema](#nestedatt--app_metadata--exporters--exporter_config--app_profile_config--ipv4))
-   `ipv6` (Attributes) (see [below for nested schema](#nestedatt--app_metadata--exporters--exporter_config--app_profile_config--ipv6))
-   `tag_id` (Boolean)
-   `transport` (Attributes) (see [below for nested schema](#nestedatt--app_metadata--exporters--exporter_config--app_profile_config--transport))
-   `type` (String)

<a id="nestedatt--app_metadata--exporters--exporter_config--app_profile_config--applications"></a>

### Nested Schema for `app_metadata.exporters.exporter_config.app_profile_config.applications`

Required:

-   `name` (String)

Optional:

-   `attributes` (Attributes List) (see [below for nested schema](#nestedatt--app_metadata--exporters--exporter_config--app_profile_config--applications--attributes))

<a id="nestedatt--app_metadata--exporters--exporter_config--app_profile_config--applications--attributes"></a>

### Nested Schema for `app_metadata.exporters.exporter_config.app_profile_config.applications.attributes`

Required:

-   `name` (String)
-   `value` (String)

<a id="nestedatt--app_metadata--exporters--exporter_config--app_profile_config--counter"></a>

### Nested Schema for `app_metadata.exporters.exporter_config.app_profile_config.counter`

Optional:

-   `bytes` (Boolean)
-   `bytes_long` (Boolean)
-   `inner_byte` (Boolean)
-   `inner_byte_long` (Boolean)
-   `packets` (Boolean)
-   `packets_long` (Boolean)

<a id="nestedatt--app_metadata--exporters--exporter_config--app_profile_config--ipv4"></a>

### Nested Schema for `app_metadata.exporters.exporter_config.app_profile_config.ipv4`

Optional:

-   `destination` (Attributes) (see [below for nested schema](#nestedatt--app_metadata--exporters--exporter_config--app_profile_config--ipv4--destination))
-   `protocol` (Boolean)
-   `source` (Attributes) (see [below for nested schema](#nestedatt--app_metadata--exporters--exporter_config--app_profile_config--ipv4--source))

<a id="nestedatt--app_metadata--exporters--exporter_config--app_profile_config--ipv4--destination"></a>

### Nested Schema for `app_metadata.exporters.exporter_config.app_profile_config.ipv4.destination`

Optional:

-   `prefix_min_mask` (String)

<a id="nestedatt--app_metadata--exporters--exporter_config--app_profile_config--ipv4--source"></a>

### Nested Schema for `app_metadata.exporters.exporter_config.app_profile_config.ipv4.source`

Optional:

-   `prefix_min_mask` (String)

<a id="nestedatt--app_metadata--exporters--exporter_config--app_profile_config--ipv6"></a>

### Nested Schema for `app_metadata.exporters.exporter_config.app_profile_config.ipv6`

Optional:

-   `destination` (Attributes) (see [below for nested schema](#nestedatt--app_metadata--exporters--exporter_config--app_profile_config--ipv6--destination))
-   `next_header` (Boolean)
-   `source` (Attributes) (see [below for nested schema](#nestedatt--app_metadata--exporters--exporter_config--app_profile_config--ipv6--source))

<a id="nestedatt--app_metadata--exporters--exporter_config--app_profile_config--ipv6--destination"></a>

### Nested Schema for `app_metadata.exporters.exporter_config.app_profile_config.ipv6.destination`

Optional:

-   `prefix_min_mask` (String)

<a id="nestedatt--app_metadata--exporters--exporter_config--app_profile_config--ipv6--source"></a>

### Nested Schema for `app_metadata.exporters.exporter_config.app_profile_config.ipv6.source`

Optional:

-   `prefix_min_mask` (String)

<a id="nestedatt--app_metadata--exporters--exporter_config--app_profile_config--transport"></a>

### Nested Schema for `app_metadata.exporters.exporter_config.app_profile_config.transport`

Optional:

-   `dst_port` (Boolean)
-   `src_port` (Boolean)
-   `tcp` (Attributes) (see [below for nested schema](#nestedatt--app_metadata--exporters--exporter_config--app_profile_config--transport--tcp))

<a id="nestedatt--app_metadata--exporters--exporter_config--app_profile_config--transport--tcp"></a>

### Nested Schema for `app_metadata.exporters.exporter_config.app_profile_config.transport.tcp`

Optional:

-   `flags` (Boolean)

<a id="nestedatt--app_metadata--exporters--exporter_config--cef"></a>

### Nested Schema for `app_metadata.exporters.exporter_config.cef`

Optional:

-   `active_timeout` (Number)
-   `inactive_timeout` (Number)
-   `record_type` (String)

<a id="nestedatt--app_metadata--match"></a>

### Nested Schema for `app_metadata.match`

Optional:

-   `datalink` (Attributes) (see [below for nested schema](#nestedatt--app_metadata--match--datalink))
-   `ipv4` (Attributes) (see [below for nested schema](#nestedatt--app_metadata--match--ipv4))
-   `ipv6` (Attributes) (see [below for nested schema](#nestedatt--app_metadata--match--ipv6))
-   `transport` (Attributes) (see [below for nested schema](#nestedatt--app_metadata--match--transport))

<a id="nestedatt--app_metadata--match--datalink"></a>

### Nested Schema for `app_metadata.match.datalink`

Optional:

-   `vlan` (Boolean)

<a id="nestedatt--app_metadata--match--ipv4"></a>

### Nested Schema for `app_metadata.match.ipv4`

Optional:

-   `destination` (Attributes) (see [below for nested schema](#nestedatt--app_metadata--match--ipv4--destination))
-   `protocol` (Boolean)
-   `source` (Attributes) (see [below for nested schema](#nestedatt--app_metadata--match--ipv4--source))

<a id="nestedatt--app_metadata--match--ipv4--destination"></a>

### Nested Schema for `app_metadata.match.ipv4.destination`

Optional:

-   `prefix_min_mask` (String)

<a id="nestedatt--app_metadata--match--ipv4--source"></a>

### Nested Schema for `app_metadata.match.ipv4.source`

Optional:

-   `prefix_min_mask` (String)

<a id="nestedatt--app_metadata--match--ipv6"></a>

### Nested Schema for `app_metadata.match.ipv6`

Optional:

-   `destination` (Attributes) (see [below for nested schema](#nestedatt--app_metadata--match--ipv6--destination))
-   `next_header` (Boolean)
-   `source` (Attributes) (see [below for nested schema](#nestedatt--app_metadata--match--ipv6--source))

<a id="nestedatt--app_metadata--match--ipv6--destination"></a>

### Nested Schema for `app_metadata.match.ipv6.destination`

Optional:

-   `prefix_min_mask` (String)

<a id="nestedatt--app_metadata--match--ipv6--source"></a>

### Nested Schema for `app_metadata.match.ipv6.source`

Optional:

-   `prefix_min_mask` (String)

<a id="nestedatt--app_metadata--match--transport"></a>

### Nested Schema for `app_metadata.match.transport`

Optional:

-   `dst_port` (Boolean)
-   `src_port` (Boolean)
-   `tcp` (Attributes) (see [below for nested schema](#nestedatt--app_metadata--match--transport--tcp))

<a id="nestedatt--app_metadata--match--transport--tcp"></a>

### Nested Schema for `app_metadata.match.transport.tcp`

Optional:

-   `flags` (Boolean)

<a id="nestedatt--app_metadata--persist_profile_config"></a>

### Nested Schema for `app_metadata.persist_profile_config`

Required:

-   `alias` (String)

Optional:

-   `applications` (Attributes List) (see [below for nested schema](#nestedatt--app_metadata--persist_profile_config--applications))
-   `type` (String)

<a id="nestedatt--app_metadata--persist_profile_config--applications"></a>

### Nested Schema for `app_metadata.persist_profile_config.applications`

Required:

-   `name` (String)

Optional:

-   `attributes` (Attributes List) (see [below for nested schema](#nestedatt--app_metadata--persist_profile_config--applications--attributes))

<a id="nestedatt--app_metadata--persist_profile_config--applications--attributes"></a>

### Nested Schema for `app_metadata.persist_profile_config.applications.attributes`

Required:

-   `name` (String)
-   `value` (String)

<a id="nestedatt--app_metadata--timeout"></a>

### Nested Schema for `app_metadata.timeout`

Optional:

-   `idle` (Number)

## Attribute Reference

In addition to all arguments above, the following attributes are exported:

-   `id` - (String) Typed resource identifier of the form `app::ami::<uuid>`. Used for cross-resource references and import.

## Import

AMI resources can be imported using the monitoring session ID and the raw resource UUID:

```shell
terraform import gigamon_app_ami.example "<monitoring_session_id>::<raw_uuid>"
```

Example:

```shell
terraform import gigamon_app_ami.example "monitoringSession::vmware::aaa-bbb-ccc::11111111-2222-3333-4444-555555555555"
```
