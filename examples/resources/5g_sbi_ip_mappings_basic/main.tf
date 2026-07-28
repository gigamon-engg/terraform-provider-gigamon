# Minimal SBI IP mappings upload example.

terraform {
  required_providers {
    gigamon = {
      source = "local/gigamon/gigamon"
    }
  }
}

provider "gigamon" {
  fm_address  = "10.114.83.72"
  skip_verify = true
  api_token   = "eyJhbGciOiJIUzI1NiJ9.eyJ0b2tlbklkIjoiMzE3MzIwMDQwNDI4NzQyMyIsInN1YiI6IlRva2VuMSIsImlhdCI6MTc4NDAwNzc5NCwiZXhwIjoxNzg2NTk5Nzk0fQ.Z2hHcfSdCYmQGW5ZjoF6lU9ms7-aehyHLFao3JyOJow"
}

resource "gigamon_5g_apps_sbi_ip_mappings" "example" {
  name             = "sbi-map50"
  type             = "ericssonVTap"
  allow_duplicates = true
  csv_path         = "/home/mustaq/ipmapping/sbi_mapping/5gsbi-nf.csv"
}
