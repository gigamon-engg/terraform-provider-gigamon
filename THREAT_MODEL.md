# Threat Model

## Overview

This repository contains two production-relevant components for Gigamon Fabric Manager (FM) Terraform workflows:

- `terraform-provider/`: a Terraform provider named `gigamon` that manages FM cloud resources through FM REST APIs. It accepts operator-authored Terraform configuration, reads local files for uploads, sends authenticated HTTPS requests to FM, and stores FM object identifiers in Terraform state.
- `tf_fm_backend/`: a Go/Gin HTTP service implementing Terraform/OpenTofu HTTP backend routes under `/terraform-state/:project`. It stores Terraform state and lock metadata in FM's MongoDB database (`fmdb2.terraformBackendState`) and delegates authorization to the local FM auth service.

The provider is primarily a client-side automation tool run by trusted infrastructure operators or CI systems. The backend is a network service intended to run on FM localhost behind HAProxy and expose state-management endpoints to authenticated Terraform clients.

Security-sensitive assets include FM API tokens, Terraform state, private keys and certificates uploaded through provider resources, FM-managed cloud inventory and monitoring configuration, lock tokens, MongoDB state documents, and FM authorization decisions.

## Threat Model, Trust Boundaries, and Assumptions

### Actors

- Trusted FM administrators and infrastructure operators who author Terraform configuration and run Terraform/OpenTofu.
- CI/CD systems that may run the provider with FM credentials.
- FM and its REST APIs, treated as the authoritative control plane for resources managed by the provider.
- The FM auth service at `http://127.0.0.1:6687/authorize`, treated as the source of backend authorization decisions.
- MongoDB on FM localhost, treated as the persistence layer for backend state.
- Network attackers, malicious lower-privileged users, compromised CI jobs, or compromised Terraform modules that can influence configuration, state backend requests, local files, environment variables, or network traffic.

### Trust Boundaries

- Terraform configuration to provider code: HCL values, provider attributes, resource attributes, file paths, and import IDs are operator-controlled and may be attacker-influenced in shared module or CI contexts.
- Provider to FM API: `fmclient.FmClient` in `terraform-provider/internal/fmclient/client.go` crosses from local execution into FM over HTTPS using a Bearer API token.
- Local filesystem to provider: image, private key, and certificate upload resources read paths from Terraform configuration and stream file contents to FM.
- Terraform/OpenTofu client to backend: requests under `/terraform-state/:project` are remote client-controlled HTTP inputs.
- Backend to FM auth service: backend authorization depends on the local auth service accepting or rejecting the supplied token for a required permission.
- Backend to MongoDB: backend state and locks are persisted in MongoDB and must preserve project isolation and atomic lock semantics.
- HAProxy/service deployment to backend: `tf_fm_backend` listens on `127.0.0.1:8893`, so the intended external exposure and TLS termination depend on deployment configuration outside the Go process.

### Assumptions

- FM API tokens are secrets with sufficient privileges to create, update, delete, deploy, and inspect FM resources exposed by the provider.
- Terraform state may contain sensitive data, including FM object IDs, configuration values, lock IDs, certificate metadata, and possibly secrets from user configuration or provider behavior.
- Terraform operators are generally trusted, but Terraform modules, variables, CI inputs, and state backend HTTP requests may be attacker-influenced in realistic enterprise workflows.
- The backend is intended to be reachable only through FM-controlled routing and authorization. Direct exposure of `127.0.0.1:8893` beyond localhost is out of design scope and should be treated as a deployment flaw.
- The provider accepts `skip_verify`; disabling TLS verification is an operator-controlled insecure mode and materially changes network attacker risk.

### Security Invariants

- FM API tokens must not be logged, exposed in diagnostics, or sent to non-FM destinations.
- Provider requests must target the configured FM host and must not let resource-level inputs redirect authenticated requests to attacker-controlled services.
- Local file upload resources must only read files explicitly selected by the operator and should avoid accidental disclosure through logs or diagnostics.
- Terraform IDs must preserve typed ID module/type/UUID semantics and must not allow one resource type to operate on another resource type's FM object.
- Backend state reads and writes must be authorized, project-scoped, bounded in size, and resistant to lock bypass or cross-project access.
- Backend lock and state writes must remain atomic so concurrent clients cannot corrupt state or bypass locks.
- Error handling and logs must not disclose state bodies, private keys, certificates, tokens, or lock IDs.

## Attack Surface, Mitigations, and Attacker Stories

### Terraform Provider Attack Surface

- Provider configuration in `terraform-provider/internal/provider/provider.go`: `fm_address`, `api_token`, `FM_API_TOKEN`, and `skip_verify`.
- FM HTTP client in `terraform-provider/internal/fmclient/client.go`: URL construction, TLS verification behavior, retries, logging, authorization headers, response handling, and non-GET serialization.
- Resource CRUD implementations under `terraform-provider/internal/*resources/`: Terraform attributes are transformed into FM API payloads and can create, update, delete, or deploy FM resources.
- Data sources under `terraform-provider/internal/*datasources/`: data source attributes query FM inventory and may disclose metadata into Terraform state or output.
- File upload paths in ESXi image and secure tunnel certificate resources: resources read local files and upload contents to FM through multipart requests.
- Import-state paths and typed IDs: import IDs and state IDs influence which FM objects are read, updated, or deleted.
- Generated docs and examples are not production runtime surfaces, but inaccurate examples can lead to insecure deployment patterns.

### Provider Mitigations and Controls

- `FM_API_TOKEN` takes precedence over the provider `api_token`, reducing the need to place tokens directly in Terraform configuration.
- Terraform schema validators restrict several aliases, enum-like fields, lengths, and numeric options.
- Sensitive provider token configuration is marked `Sensitive`.
- `fmclient.FmClient` uses HTTPS by default and only disables certificate validation when `skip_verify` is set.
- Non-GET FM operations are serialized by an internal mutex to reduce FM-side race conditions during Terraform parallelism.
- Typed ID helpers in `terraform-provider/internal/commonutils/typedid_utils.go` provide a consistent `<module>::<type>::<uuid>` format.
- Multipart uploads use `filepath.Base` for uploaded filenames, limiting filename path disclosure in multipart metadata.

### Provider Attacker Stories

- A malicious Terraform module or variable value points an upload resource at a sensitive local file path. If a trusted operator or CI runner applies it, the provider may read and upload that file to FM.
- A compromised CI environment sets `FM_API_TOKEN` or `fm_address` to make provider operations run with unexpected credentials or against an unexpected FM host.
- A network attacker can intercept provider-to-FM traffic if an operator sets `skip_verify=true` or trusts an attacker-controlled certificate path outside this codebase.
- A malformed or cross-type typed ID could cause a resource to act on the wrong FM object if resource code fails to validate module/type before extracting the UUID.
- FM API responses can contain unexpected JSON shapes or large response bodies. Provider code must fail closed and avoid leaking sensitive response data in logs or diagnostics.

### FM Terraform Backend Attack Surface

- HTTP routes in `tf_fm_backend/main.go`:
  - `GET /terraform-state/:project`
  - `POST /terraform-state/:project`
  - `GET /terraform-state/:project/lock`
  - `DELETE /terraform-state/:project/lock`
  - `LOCK /terraform-state/:project/lock`
  - `UNLOCK /terraform-state/:project/lock`
- Authorization parsing supports Bearer tokens and Terraform-compatible Basic auth where the password is treated as the token.
- Project names, lock IDs, query parameters, request bodies, content types, and compressed/decompressed state bodies are client-controlled.
- MongoDB collection `terraformBackendState` stores one document per project with state body, lock fields, usernames, timestamps, and compression metadata.
- The local FM auth service is called for every backend operation, and backend security depends on the correctness and availability of that service.
- HAProxy/systemd/Ansible assets in `playbooks/` affect real-world exposure, TLS termination, and service routing.

### Backend Mitigations and Controls

- Backend binds to `127.0.0.1:8893`, limiting direct network exposure when deployed as designed.
- All routes use `requireBearerAuthorization`; state operations require `TRAFFIC_CONTROL/write`, while lock inspection and force-delete require `ALL/ALL`.
- Project names are validated with a bounded safe-character regexp before database access.
- The service uses structured BSON filters rather than string-built MongoDB queries.
- Request body sizes are bounded: state write bodies, lock bodies, auth responses, and decompressed state output have explicit limits.
- Gzip decompression uses a limited reader to reduce zip-bomb risk.
- State writes require a matching lock ID and update state atomically with the lock check.
- Lock acquisition and release use MongoDB atomic operations and a unique project index.
- Access logs omit query strings to avoid logging lock IDs, and recovery middleware avoids stack traces that might contain sensitive state.
- Server timeouts are configured for header, read, write, and idle operations.

### Backend Attacker Stories

- An authenticated but low-privileged user attempts to read or overwrite another project's state by choosing a project name. Authorization is currently permission-based, not visibly project-scoped in this repository, so tenant/project scoping depends on the FM auth service and deployment model.
- An attacker with a valid `TRAFFIC_CONTROL/write` token attempts to corrupt state by writing without a lock or with a wrong lock ID. The backend should reject this with locked/not-found behavior.
- An attacker submits oversized or compressed malicious state to exhaust memory or storage. Body size and decompression limits reduce but do not remove resource consumption risk.
- A compromised local process calls the backend directly on localhost. Binding to localhost does not protect against same-host attackers; token authorization remains the primary control.
- A malicious client repeatedly locks projects or writes large allowed-size states to cause denial of service. The service has size/time limits but no explicit per-user or per-project rate limiting in this repository.
- If HAProxy exposes the backend without TLS or strips/forges authorization headers incorrectly, token confidentiality and authorization assumptions can fail outside the Go process.

### Out of Scope or Lower-Relevance Stories

- Browser XSS and CSRF are generally out of scope because the provider is not a browser application and the backend exposes API endpoints for Terraform clients, not cookie-authenticated web pages.
- SQL injection is not directly relevant; persistence uses MongoDB with structured BSON operations.
- Public unauthenticated internet attackers are out of intended scope for `tf_fm_backend` because it binds to localhost and is expected to sit behind FM-controlled proxying, but accidental external exposure would raise severity.
- Malicious FM itself is out of scope. The provider and backend trust FM APIs, FM auth, and FM MongoDB as authoritative platform components.

## Severity Calibration

### Critical

Critical issues compromise FM administrative credentials, allow unauthenticated or cross-tenant modification of Terraform state, or enable arbitrary FM resource mutation/deletion at scale.

Examples:

- Backend route allows unauthenticated state read/write, lock bypass, or force-unlock for arbitrary projects.
- Provider sends `FM_API_TOKEN` or `api_token` to an attacker-controlled host because resource-level input can override the FM request destination.
- Backend authorization accepts a forged token or treats any Basic credential as authorized without consulting FM auth.
- Typed ID confusion lets a low-risk resource delete or deploy unrelated high-impact FM objects without validating object type or module.

### High

High issues expose sensitive state or secrets to authenticated users who should not have them, permit unauthorized FM configuration changes, or weaken transport security in non-obvious ways.

Examples:

- Backend project authorization is only global permission-based where deployment requires per-project isolation, allowing one authorized user to read another team's Terraform state.
- Provider logs private key contents, API tokens, Terraform state, or full sensitive FM responses.
- Upload resources can be induced by untrusted module input to exfiltrate local CI secrets or SSH keys through FM uploads.
- Backend lock validation is bypassable, allowing concurrent writes and state corruption for another active Terraform run.

### Medium

Medium issues require an already-authenticated operator, compromised CI input, insecure configuration, or limited race/resource-exhaustion conditions.

Examples:

- `skip_verify=true` enables man-in-the-middle attacks in environments where operators expect strong TLS but have chosen the insecure option.
- Missing schema validation allows malformed FM payloads that cause unintended resource state drift or denial of a specific Terraform apply.
- Backend accepts allowed-size but expensive state operations without rate limiting, enabling authenticated denial of service.
- Import-state handling accepts malformed typed IDs and causes confusing state or operations on unintended but same-privilege resources.

### Low

Low issues are mostly hardening gaps, diagnostics quality issues, or developer-only risks that do not cross a meaningful trust boundary by themselves.

Examples:

- Error messages disclose non-sensitive FM object names or existence information to already-authorized users.
- Generated documentation or examples recommend less secure local testing defaults without affecting runtime code.
- Provider response parsing returns broad errors that reduce debuggability but do not leak secrets or change authorization.
- Release or packaging scripts have unsafe behavior only when run manually by trusted maintainers on a trusted workstation.
