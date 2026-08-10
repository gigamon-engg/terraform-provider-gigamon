# Agent Guide

## Repository Summary

This repository contains Gigamon's Terraform integration for Fabric Manager (FM). It is a Go workspace with two main deliverables:

- `terraform-provider/`: the `terraform-provider-gigamon` Terraform provider, built with `github.com/hashicorp/terraform-plugin-framework`. It manages Gigamon FM cloud resources through FM REST APIs.
- `tf_fm_backend/`: a standalone Go/Gin HTTP backend service implementing Terraform/OpenTofu HTTP state backend endpoints. It stores state and locks in FM's MongoDB database.

Supporting content includes provider examples, generated Terraform provider docs, release packaging scripts, and Ansible playbooks for deploying the backend service and related proxy/service files.

## Important Paths

- `go.work`: workspace file listing the provider, backend, and internal provider packages as separate modules.
- `terraform-provider/main.go`: Terraform provider entry point and default provider version.
- `terraform-provider/internal/provider/provider.go`: provider schema, configuration, resource registration, and data source registration.
- `terraform-provider/internal/fmclient/client.go`: FM REST client, auth token handling, version compatibility check, error model, and non-GET request serialization.
- `terraform-provider/internal/commonutils/`: shared typed ID helpers and common update utilities.
- `terraform-provider/internal/commonresources/`: resources shared across supported platforms, including monitoring sessions, apps, maps, links, tunnels, raw endpoints, and endpoint-interface mappings.
- `terraform-provider/internal/esxiresources/` and `terraform-provider/internal/esxidatasources/`: VMware ESXi resources and data sources.
- `terraform-provider/internal/thirdpartyorchestrationresources/` and `terraform-provider/internal/thirdpartyorchestrationdatasources/`: third-party orchestration/anyCloud resources and data sources.
- `terraform-provider/internal/securetunnelcertsresources/` and `terraform-provider/internal/securetunnelcertsdatasources/`: secure tunnel certificate resources and data sources.
- `terraform-provider/docs/`: generated Terraform provider documentation.
- `terraform-provider/examples/`: sample Terraform configurations for install verification, resources, and broader usage tests.
- `tf_fm_backend/main.go`: HTTP state backend server.
- `playbooks/`: Ansible deployment assets and service/proxy configuration.
- `.github/copilot-instructions.md`: existing contributor/assistant guidance with project-specific conventions.

## Build and Verification Commands

Run commands from the repository root unless noted.

```bash
# Build/install the provider to $GOBIN.
go install ./terraform-provider

# Build the backend service.
go build ./tf_fm_backend

# Compile all Go modules in the workspace.
go test ./...

# Generate provider docs when tfplugindocs is installed.
tfplugindocs

# Release build; this checks out/pulls the target branch, bumps release_version.txt,
# commits, cross-compiles artifacts, and tags the repo.
./release.sh <branch> [true|false]
```

There are currently no checked-in `*_test.go` files. For behavior changes, prefer at least a compile-level verification with `go test ./...`, and add focused tests where a package is practical to test without a live FM.

## Provider Architecture

The provider uses Terraform Plugin Framework models and diagnostics throughout. Provider configuration requires `fm_address`; authentication uses an API token from `FM_API_TOKEN` first, then the optional `api_token` provider attribute. `skip_verify` controls TLS certificate verification.

`provider.go` registers the resource and data source factories. When adding a new resource or data source, implement it in the appropriate domain package, then add its constructor to `Resources` or `DataSources`.

Most resources follow this pattern:

- A Terraform model struct using `tfsdk` tags.
- One or more FM API structs using `json` tags.
- CRUD methods implementing `resource.Resource`.
- Import support via `resource.ResourceWithImportState` where the resource can be imported.
- `ModifyPlan` only when the resource needs plan-time normalization or validation that cannot live in schema validators.

## FM Client Rules

Use `terraform-provider/internal/fmclient` for FM API access. Avoid introducing ad hoc HTTP clients in resource packages.

`FmClient` intentionally serializes non-GET operations with an internal mutex because FM can behave incorrectly when concurrent writes occur. Preserve that behavior unless FM semantics are proven to have changed.

FM errors should use `FMErrors` and the exported error codes (`ObjectNotFound`, `RequestConflict`, `TooManyRequests`, `CommunicationErrors`, `GeneralErrors`) so callers can distinguish not-found, conflict, rate limiting, transport, and generic failures.

## Typed ID Convention

Terraform resource IDs are typed IDs in this format:

```text
<module>::<type>::<uuid>
```

Always use helpers from `terraform-provider/internal/commonutils/typedid_utils.go`:

- `MakeTypedID`
- `ParseTypedID`
- `ModuleFromTypedID`
- `TypeFromTypedID`
- `UUIDFromTypedID`

Do not construct or parse typed IDs with manual string concatenation or splitting in resource code.

## Backend Architecture

`tf_fm_backend` exposes Terraform/OpenTofu HTTP backend routes under `/terraform-state/:project`:

- `GET /terraform-state/:project`: read state.
- `POST /terraform-state/:project`: write state.
- `GET /terraform-state/:project/lock`: inspect lock.
- `LOCK /terraform-state/:project/lock`: acquire lock.
- `UNLOCK /terraform-state/:project/lock`: release lock.
- `DELETE /terraform-state/:project/lock`: compatibility lock removal path.

State is stored in MongoDB database `fmdb2`, collection `terraformBackendState`. Each project has a single document containing state and lock fields. Project names are validated with a bounded safe-character regexp.

Authorization is delegated to FM's local auth service, defaulting to `http://127.0.0.1:6687/authorize`. The backend accepts Bearer tokens and Terraform-compatible Basic auth where the token is supplied as the Basic password.

The backend includes request size limits and gzip/decompression limits. Preserve those limits or update them deliberately with MongoDB document limits and denial-of-service risk in mind.

## Documentation and Examples

Generated provider documentation lives in `terraform-provider/docs/`. Source-level schema descriptions drive generated docs, so update schema descriptions alongside user-visible behavior changes.

The root `README.md` still describes local provider installation using the fixed local plugin path:

```text
~/.terraform.d/plugins/local/gigamon/gigamon/1.0.0/linux_amd64/terraform-provider-gigamon
```

After rebuilding the provider for local testing, consumer Terraform projects may need `.terraform/` and `.terraform.lock.hcl` removed before running `terraform init`, because the local plugin version path is fixed while checksums change.

## Release Notes for Agents

`release.sh` is stateful and performs git operations: checkout, pull, version bump commit, push, cross-compile, tag, and tag push. Do not run it casually during routine development.

Release version is stored in `release_version.txt`. Build-time provider version can be injected with:

```bash
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "-X 'main.version=v<version>'" ./terraform-provider
```

## Coding Guidance

- Keep changes scoped to the relevant module and package.
- Preserve the GPL header style when adding new Go files.
- Prefer existing package patterns over new abstractions.
- Use Terraform Plugin Framework diagnostics instead of panics or raw returns in provider/resource methods.
- Use `types.*` consistently in Terraform models and account for unknown/null values.
- Keep API payload structs separate from Terraform state models when FM request/response shapes differ.
- Do not check secrets, private keys, tokens, generated credentials, or local Terraform state into the repository.
- Be careful around existing build artifacts under `terraform-provider/build/`; avoid modifying them unless the task explicitly concerns packaging or release artifacts.
