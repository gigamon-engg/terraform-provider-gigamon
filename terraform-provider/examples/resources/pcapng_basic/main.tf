# Copyright (c) HashiCorp, Inc.

# Minimal typed PCapNG example.
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

resource "gigamon_app_pcapng" "minimal" {
  monitoring_session_id = local.monitoring_session_id
  capture_mode          = "triggered"

  packet_filter = {
    bpf_syntax  = "udp and port 2152"
    source_ip   = "10.114.50.10"
    dest_ip     = "10.114.50.20"
    vlan_filter = [110, 120]
  }

  output_config = {
    file_path     = "/var/log/gigamon/pcapng/sbi-capture.pcapng"
    max_file_size = 250
    rotation      = true
    compression   = "xz"
  }

  performance = {
    buffer_size    = 128
    thread_count   = 8
    packet_snaplen = 2048
  }
}
