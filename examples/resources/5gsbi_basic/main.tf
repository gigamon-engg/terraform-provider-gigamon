# Copyright (c) HashiCorp, Inc.

# Minimal typed 5G-SBI example.
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

resource "gigamon_app_5gsbi" "minimal" {
  monitoring_session_id            = local.monitoring_session_id
  sbi_mode                         = "nrf"
  alias                            = "sbi5gAppTemplate"
  name                             = "sbi5g"
  type                             = "ericssonVTap"
  ipMappingAlias                   = "nfinstance"
  http2SynthesizeToolMtuPacketSize = 0
  http2SynthesizeIndexedHeaders    = true
  http2SynthesizeCompressedHeaders = true
  transactionLog                   = true
  transactionLogFileInterval       = 60
  logFolderSize                    = 0
  statsLog                         = true
  logFolderLoc                     = "/var/log"

  protocol_handlers = ["http", "https"]

  authentication = {
    enabled   = false
    cert_path = "/etc/gigamon/certs/sbi.crt"
    key_path  = "/etc/gigamon/certs/sbi.key"
  }

  ericssonVTapConfig = {
    mode                 = "L7json"
    eevtapVersion        = "2"
    numTCPFlows          = 256
    tcpFlowTimeout       = 1800
    numStreamsPerFlow    = 8192
    http2RequestTimeout  = 10
    http2ResponseTimeout = 2
    destinationIP        = "SCP"
    fqdnMappingAlias     = "5g-sbiFQDN"
  }
}
