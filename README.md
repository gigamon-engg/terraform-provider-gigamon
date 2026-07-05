# terraform-provider-gigamon

Terraform provider for **Gigamon Fabric Manager (FM)**, supporting FM version 6.14 and later.

## Quick start — Provider

You can use the provider in three common ways:

* use the published provider from the registry
* build it locally from this repository (including your fork), or
* download the published **v6.14.0** release artifact from GitHub Releases.

### Option 1: Use the published provider from the registry

Published provider: [Gigamon provider on Terraform Registry](https://registry.terraform.io/providers/gigamon-engg/gigamon/6.14.0)

```hcl
terraform {
  required_providers {
    gigamon = {
      source  = "gigamon-engg/gigamon"
      version = ">= 6.14"
    }
  }
}

provider "gigamon" {
  fm_address  = "<your-fm-host>"
  skip_verify = true
  api_token   = "<your-fm-api-token>" # or set the FM_API_TOKEN env var
}
```


### Option 2: Build locally from this repository or your fork

```bash
go build -o terraform-provider-gigamon
```

Install for local Terraform use:

```bash
mkdir -p ~/.terraform.d/plugins/local/gigamon/gigamon/6.14.0/linux_amd64
cp terraform-provider-gigamon \
  ~/.terraform.d/plugins/local/gigamon/gigamon/6.14.0/linux_amd64/
```

### Option 3: Download from the GitHub release

Download the **v6.14.0** release ZIP from the GitHub Releases page, extract it, rename the binary to `terraform-provider-gigamon` if needed, and place it in the same local Terraform plugin path shown above.

Release page:

[Gigamon provider v6.14.0 release](https://github.com/gigamon-engg/terraform-provider-gigamon/releases/tag/v6.14.0)

Install path:

```bash
mkdir -p ~/.terraform.d/plugins/local/gigamon/gigamon/6.14.0/linux_amd64
cp terraform-provider-gigamon \
  ~/.terraform.d/plugins/local/gigamon/gigamon/6.14.0/linux_amd64/
```

Whether you build from your fork or download the release artifact, the Terraform usage is the same after the binary is placed in the local plugin directory.

### Terraform configuration

```hcl
terraform {
  required_providers {
    gigamon = {
      source  = "local/gigamon/gigamon"
      version = "6.14.0"
    }
  }
}

provider "gigamon" {
  fm_address  = "<your-fm-host>"
  skip_verify = true
  api_token   = "<your-fm-api-token>" # or set the FM_API_TOKEN env var
}
```

See `examples/` for end-to-end configurations and `docs/` for the full resource and data source reference.

## Development

```bash
# Build provider
go build .

# Run all tests
go test ./...
```

Generated documentation under `docs/` is produced by [`tfplugindocs`](https://github.com/hashicorp/terraform-plugin-docs).

## License

See [LICENSE](LICENSE)


---

## Sources

- [Gigamon provider v6.14.0 release](https://github.com/gigamon-engg/terraform-provider-gigamon/releases/tag/v6.14.0)
