# Minimal typed GVHTTP2 example.
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

resource "gigamon_app_gvhttp2" "minimal" {
  monitoring_session_id = local.monitoring_session_id
  http2_mode            = "enabled"

  protocol_config = {
    stream_multiplexing = true
    server_push         = false
  }

  compression = {
    enabled = true
    level   = 6
  }

  flow_control = {
    window_size         = 65535
    initial_window_size = 65535
  }
}
