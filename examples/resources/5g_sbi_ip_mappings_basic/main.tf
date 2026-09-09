# Minimal SBI IP mappings upload example.

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

resource "gigamon_5g_apps_sbi_ip_mappings" "example" {
  name             = "sbi-map50"
  type             = "ericssonVTap"
  allow_duplicates = true
  csv_path         = var.csv_file_path
}
