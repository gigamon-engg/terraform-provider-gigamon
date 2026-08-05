# Minimal typed 5G-EVP example.
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

resource "gigamon_app_5gevp" "minimal" {
  monitoring_session_id = local.monitoring_session_id

  network_mode = "standalone"

  traffic_optimization = {
    enabled = true
    level   = 5
  }

  performance_tuning = {
    cache_size   = 256
    buffer_depth = 1000
  }
}
