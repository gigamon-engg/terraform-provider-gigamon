# Minimal typed App Viz example.
# This example uses an existing monitoring session that is imported.

terraform {
  required_providers {
    gigamon = {
      source = "local/gigamon/gigamon"
    }
  }
}

provider "gigamon" {
  fm_address  = var.fm_ip_address
  skip_verify = true
  api_token   = var.api_token
}

# Store your existing monitoring session ID locally
# Format: monitoringSession::<platform>::<uuid>
locals {
  monitoring_session_id = var.monitoring_session_id
}

resource "gigamon_app_viz" "minimal" {
  monitoring_session_id = local.monitoring_session_id

  alias         = "app_viz1"
  description   = ""
  action        = true
  mgmt_interface = "internal"

  exporter_config = {
    monitor = {
      timeout = 300
    }
  }
}
