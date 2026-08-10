# Security Review: fm_terraform_provider

## Scope

The scan reviewed the canonical include paths and exclusions listed below.

- Scan mode: repository
- Target kind: git_worktree
- Target ID: fm_terraform_provider-ce956a21ac1f
- Revision: ce956a21ac1f
- Snapshot digest: codex-security-snapshot/v1:sha256:6dcdf8aa10353b1ac7e164b152ae751cef7c3448db879841c61b3216e077b287
- Inventory strategy: repository
- Included paths: .
- Excluded paths: none
- Runtime or test status: not recorded
- Scan context: Deep repository-wide scan with four completed discovery rounds, 49-row exhaustive source worklist, static validation, and attack-path analysis.

### Scan Summary

| Field | Value |
| --- | --- |
| Reportable findings | 10 |
| Severity mix | high: 3, medium: 7 |
| Confidence mix | high: 6, medium: 4 |
| Coverage | complete |
| Validation mode | not recorded |

Canonical artifacts: `scan-manifest.json`, `findings.json`, and `coverage.json`. This report is a deterministic projection of those files.

## Threat Model

Terraform provider and FM backend surfaces handling FM tokens, Terraform state, uploaded files, deployment configuration, and signing/TLS keys. Trust boundaries include Terraform configuration, HTTP backend clients, FM auth/MongoDB, local filesystem uploads, and deployment artifacts.

## Findings

| Finding | Severity | Confidence |
| --- | --- | --- |
| [Provider registry signing private keys are checked into the repository and used by build tooling](#finding-1) | high | high |
| [Terraform backend authorization is not scoped to the requested project](#finding-2) | high | medium |
| [Unauthenticated documentation upload writes attacker-controlled files to arbitrary directories](#finding-3) | high | high |
| [FM client logs full response and error bodies from authenticated API calls](#finding-4) | medium | medium |
| [Documentation registry service runs Flask debug mode on a network-facing listener](#finding-5) | medium | medium |
| [Documentation route can render markdown files outside the docs directory](#finding-6) | medium | medium |
| [AMX HTTP exporter Authorization headers are stored as non-sensitive Terraform state](#finding-7) | medium | high |
| [Deployment ships checked-in HAProxy TLS private key material](#finding-8) | medium | high |
| [ESXi connection create diagnostics can disclose write-only vCenter passwords](#finding-9) | medium | high |
| [Deployment playbook installs privileged service and proxy configuration as world-writable](#finding-10) | medium | high |

### Confidence Scale

| Label | Meaning |
| --- | --- |
| high | Direct evidence supports the finding with no material unresolved blocker. |
| medium | Evidence supports a plausible issue, but material runtime or reachability proof remains. |
| low | Evidence is incomplete and the item is retained only for explicit follow-up. |

<a id="finding-1"></a>

### [1] Provider registry signing private keys are checked into the repository and used by build tooling

| Field | Value |
| --- | --- |
| Severity | high |
| Confidence | high |
| Confidence rationale | Static validation found direct source/control/sink evidence; confidence is lowered where runtime or deployment-specific behavior remains unproven. |
| Category | Supply-chain signing key exposure |
| CWE | CWE-798, CWE-321 |
| Affected lines | terraform-provider/build/gpg_keys/private-keys-v1.d/0BC85F438C34F15E2E9E1E63E3AFAE606F2F8C50.key:1-2, terraform-provider/build/gpg_keys/private-keys-v1.d/2133D3B65F862D2C5517E3A08FC36C9F6D03BB7F.key:1-2, terraform-provider/build/build.py:37-39, terraform-provider/build/build.py:70-84 |

#### Summary

GnuPG private key files are stored under the provider build directory, and build tooling uses that repository-local GPG home to sign provider checksum files advertised in registry metadata.

#### Validation

Static validation found direct source/control/sink evidence; confidence is lowered where runtime or deployment-specific behavior remains unproven. Validation details were not recorded separately.

Validation method: static code trace

#### Dataflow

The canonical finding records the affected path at terraform-provider/build/gpg_keys/private-keys-v1.d/0BC85F438C34F15E2E9E1E63E3AFAE606F2F8C50.key:1-2, terraform-provider/build/gpg_keys/private-keys-v1.d/2133D3B65F862D2C5517E3A08FC36C9F6D03BB7F.key:1-2, terraform-provider/build/build.py:37-39, terraform-provider/build/build.py:70-84, but no expanded source-to-sink narrative was recorded.

#### Reachability

Reachability was not recorded beyond the canonical finding summary and affected locations.

#### Severity

**High** — The scan assigned high severity; no separate canonical severity rationale was recorded.

Additional runtime or deployment evidence could raise or lower this severity.

#### Remediation

Revoke and rotate the exposed signing keys, remove private keys from the repository and history, move signing to a controlled secret manager or CI signing service, and publish new trusted public keys.

<a id="finding-2"></a>

### [2] Terraform backend authorization is not scoped to the requested project

| Field | Value |
| --- | --- |
| Severity | high |
| Confidence | medium |
| Confidence rationale | Static validation found direct source/control/sink evidence; confidence is lowered where runtime or deployment-specific behavior remains unproven. |
| Category | Authorization bypass / cross-project state access |
| CWE | CWE-862, CWE-639 |
| Affected lines | tf_fm_backend/main.go:98-123, tf_fm_backend/main.go:307-343, tf_fm_backend/main.go:527-586, tf_fm_backend/main.go:682-704 |

#### Summary

The backend authorizes only a global TRAFFIC_CONTROL/write permission, then uses the caller-controlled project path to select the MongoDB state and lock document. Repository evidence does not show a per-project authorization check.

#### Validation

Static validation found direct source/control/sink evidence; confidence is lowered where runtime or deployment-specific behavior remains unproven. Validation details were not recorded separately.

Validation method: static code trace

#### Dataflow

The canonical finding records the affected path at tf_fm_backend/main.go:98-123, tf_fm_backend/main.go:307-343, tf_fm_backend/main.go:527-586, tf_fm_backend/main.go:682-704, but no expanded source-to-sink narrative was recorded.

#### Reachability

Reachability was not recorded beyond the canonical finding summary and affected locations.

#### Severity

**High** — The scan assigned high severity; no separate canonical severity rationale was recorded.

Additional runtime or deployment evidence could raise or lower this severity.

#### Remediation

Include the requested project in the authz decision or enforce a project ACL before any state or lock read/write; add tests proving a token for one project cannot read, lock, unlock, or write another project.

<a id="finding-3"></a>

### [3] Unauthenticated documentation upload writes attacker-controlled files to arbitrary directories

| Field | Value |
| --- | --- |
| Severity | high |
| Confidence | high |
| Confidence rationale | Static validation found direct source/control/sink evidence; confidence is lowered where runtime or deployment-specific behavior remains unproven. |
| Category | Arbitrary file write / missing authentication |
| CWE | CWE-434, CWE-22, CWE-306 |
| Affected lines | terraform-provider/doc_render/render.py:264-265, terraform-provider/doc_render/render.py:284-291, playbooks/roles/tf/files/haproxy.cfg:36-43 |

#### Summary

The deployed Flask documentation/registry service exposes a POST upload route without authentication and converts the URL path into an absolute filesystem directory before creating it world-writable and saving the uploaded file.

#### Validation

Static validation found direct source/control/sink evidence; confidence is lowered where runtime or deployment-specific behavior remains unproven. Validation details were not recorded separately.

Validation method: static code trace

#### Dataflow

The canonical finding records the affected path at terraform-provider/doc_render/render.py:264-265, terraform-provider/doc_render/render.py:284-291, playbooks/roles/tf/files/haproxy.cfg:36-43, but no expanded source-to-sink narrative was recorded.

#### Reachability

Reachability was not recorded beyond the canonical finding summary and affected locations.

#### Severity

**High** — The scan assigned high severity; no separate canonical severity rationale was recorded.

Additional runtime or deployment evidence could raise or lower this severity.

#### Remediation

Remove the endpoint or require authentication, restrict uploads to a fixed owned directory, reject absolute and parent paths, avoid chmod 0777, and add tests for traversal and unauthenticated upload attempts.

<a id="finding-4"></a>

### [4] FM client logs full response and error bodies from authenticated API calls

| Field | Value |
| --- | --- |
| Severity | medium |
| Confidence | medium |
| Confidence rationale | Static validation found direct source/control/sink evidence; confidence is lowered where runtime or deployment-specific behavior remains unproven. |
| Category | Sensitive data exposure in logs |
| CWE | CWE-532 |
| Affected lines | terraform-provider/internal/fmclient/client.go:392-405, terraform-provider/internal/esxiutils/connection.go:54-78, terraform-provider/internal/commonresources/apps.go:4008-4041 |

#### Summary

The shared FM client logs every response body and also embeds non-2xx response bodies in returned errors. Several FM responses can include sensitive provider-managed values such as connection data or AMX headers.

#### Validation

Static validation found direct source/control/sink evidence; confidence is lowered where runtime or deployment-specific behavior remains unproven. Validation details were not recorded separately.

Validation method: static code trace

#### Dataflow

The canonical finding records the affected path at terraform-provider/internal/fmclient/client.go:392-405, terraform-provider/internal/esxiutils/connection.go:54-78, terraform-provider/internal/commonresources/apps.go:4008-4041, but no expanded source-to-sink narrative was recorded.

#### Reachability

Reachability was not recorded beyond the canonical finding summary and affected locations.

#### Severity

**Medium** — The scan assigned medium severity; no separate canonical severity rationale was recorded.

Additional runtime or deployment evidence could raise or lower this severity.

#### Remediation

Redact or omit response bodies from info logs and error strings by default, with narrowly scoped opt-in debug logging that applies structured redaction.

<a id="finding-5"></a>

### [5] Documentation registry service runs Flask debug mode on a network-facing listener

| Field | Value |
| --- | --- |
| Severity | medium |
| Confidence | medium |
| Confidence rationale | Static validation found direct source/control/sink evidence; confidence is lowered where runtime or deployment-specific behavior remains unproven. |
| Category | Debug mode exposure |
| CWE | CWE-489 |
| Affected lines | terraform-provider/doc_render/render.py:310-326, playbooks/roles/tf/files/tf_provider.service:4-9, playbooks/roles/tf/files/haproxy.cfg:36-43 |

#### Summary

The service defaults to 0.0.0.0 and starts Flask with debug=True, while the playbook service and HAProxy config expose the service behind a TLS frontend. Werkzeug debugger exploitability depends on runtime debugger/PIN behavior, but debug mode should not be enabled in production.

#### Validation

Static validation found direct source/control/sink evidence; confidence is lowered where runtime or deployment-specific behavior remains unproven. Validation details were not recorded separately.

Validation method: static code trace

#### Dataflow

The canonical finding records the affected path at terraform-provider/doc_render/render.py:310-326, playbooks/roles/tf/files/tf_provider.service:4-9, playbooks/roles/tf/files/haproxy.cfg:36-43, but no expanded source-to-sink narrative was recorded.

#### Reachability

Reachability was not recorded beyond the canonical finding summary and affected locations.

#### Severity

**Medium** — The scan assigned medium severity; no separate canonical severity rationale was recorded.

Additional runtime or deployment evidence could raise or lower this severity.

#### Remediation

Run with debug disabled, use a production WSGI server, bind only to intended interfaces, and add deployment checks that fail if debug mode is enabled.

<a id="finding-6"></a>

### [6] Documentation route can render markdown files outside the docs directory

| Field | Value |
| --- | --- |
| Severity | medium |
| Confidence | medium |
| Confidence rationale | Static validation found direct source/control/sink evidence; confidence is lowered where runtime or deployment-specific behavior remains unproven. |
| Category | Path traversal / local file disclosure |
| CWE | CWE-22 |
| Affected lines | terraform-provider/doc_render/render.py:124-134, terraform-provider/doc_render/render.py:180-181, terraform-provider/doc_render/render.py:215-220 |

#### Summary

The public documentation route joins URL path components into a markdown filename and then joins that value under the docs directory without validating the resolved path remains under docs. A res_type such as .. can select files one directory above docs when the route shape permits it.

#### Validation

Static validation found direct source/control/sink evidence; confidence is lowered where runtime or deployment-specific behavior remains unproven. Validation details were not recorded separately.

Validation method: static code trace

#### Dataflow

The canonical finding records the affected path at terraform-provider/doc_render/render.py:124-134, terraform-provider/doc_render/render.py:180-181, terraform-provider/doc_render/render.py:215-220, but no expanded source-to-sink narrative was recorded.

#### Reachability

Reachability was not recorded beyond the canonical finding summary and affected locations.

#### Severity

**Medium** — The scan assigned medium severity; no separate canonical severity rationale was recorded.

Additional runtime or deployment evidence could raise or lower this severity.

#### Remediation

Map route values to known generated documentation names, reject dot segments, and verify the real path stays under the docs directory before opening it.

<a id="finding-7"></a>

### [7] AMX HTTP exporter Authorization headers are stored as non-sensitive Terraform state

| Field | Value |
| --- | --- |
| Severity | medium |
| Confidence | high |
| Confidence rationale | Static validation found direct source/control/sink evidence; confidence is lowered where runtime or deployment-specific behavior remains unproven. |
| Category | Sensitive data exposure in Terraform state |
| CWE | CWE-312 |
| Affected lines | terraform-provider/internal/commonresources/apps.go:3190-3218, terraform-provider/internal/commonresources/apps.go:3673-3713, terraform-provider/internal/commonresources/apps.go:4008-4041, terraform-provider/internal/commonresources/apps.go:4696-4705 |

#### Summary

The AMX HTTP exporter schema documents headers such as Authorization: Bearer but models headers and endpoints as normal strings/lists. The provider sends them to FM, reconstructs them from FM responses, and preserves them in state across reads.

#### Validation

Static validation found direct source/control/sink evidence; confidence is lowered where runtime or deployment-specific behavior remains unproven. Validation details were not recorded separately.

Validation method: static code trace

#### Dataflow

The canonical finding records the affected path at terraform-provider/internal/commonresources/apps.go:3190-3218, terraform-provider/internal/commonresources/apps.go:3673-3713, terraform-provider/internal/commonresources/apps.go:4008-4041, terraform-provider/internal/commonresources/apps.go:4696-4705, but no expanded source-to-sink narrative was recorded.

#### Reachability

Reachability was not recorded beyond the canonical finding summary and affected locations.

#### Severity

**Medium** — The scan assigned medium severity; no separate canonical severity rationale was recorded.

Additional runtime or deployment evidence could raise or lower this severity.

#### Remediation

Mark secret-bearing header and endpoint fields sensitive or write-only, split secret values from non-secret names, and redact these fields in logs, state, diagnostics, and docs.

<a id="finding-8"></a>

### [8] Deployment ships checked-in HAProxy TLS private key material

| Field | Value |
| --- | --- |
| Severity | medium |
| Confidence | high |
| Confidence rationale | Static validation found direct source/control/sink evidence; confidence is lowered where runtime or deployment-specific behavior remains unproven. |
| Category | Hardcoded private key material |
| CWE | CWE-798, CWE-321 |
| Affected lines | playbooks/roles/tf/files/haproxy.pem:1-3, playbooks/roles/tf/files/apache_server.key:1-3, playbooks/roles/tf/tasks/main.yml:73-80, playbooks/roles/tf/files/haproxy.cfg:36-37 |

#### Summary

The playbook repository contains TLS private key material and copies haproxy.pem into /etc/ssl/private for the public HAProxy listener. Anyone with repository access can impersonate or decrypt traffic for that deployed identity if the certificate is used.

#### Validation

Static validation found direct source/control/sink evidence; confidence is lowered where runtime or deployment-specific behavior remains unproven. Validation details were not recorded separately.

Validation method: static code trace

#### Dataflow

The canonical finding records the affected path at playbooks/roles/tf/files/haproxy.pem:1-3, playbooks/roles/tf/files/apache_server.key:1-3, playbooks/roles/tf/tasks/main.yml:73-80, playbooks/roles/tf/files/haproxy.cfg:36-37, but no expanded source-to-sink narrative was recorded.

#### Reachability

Reachability was not recorded beyond the canonical finding summary and affected locations.

#### Severity

**Medium** — The scan assigned medium severity; no separate canonical severity rationale was recorded.

Additional runtime or deployment evidence could raise or lower this severity.

#### Remediation

Rotate the certificate/key pair, remove private keys from the repository and history, provision certificates from a secret manager, and ensure only public certificates or templates are checked in.

<a id="finding-9"></a>

### [9] ESXi connection create diagnostics can disclose write-only vCenter passwords

| Field | Value |
| --- | --- |
| Severity | medium |
| Confidence | high |
| Confidence rationale | Static validation found direct source/control/sink evidence; confidence is lowered where runtime or deployment-specific behavior remains unproven. |
| Category | Sensitive data exposure in diagnostics |
| CWE | CWE-209, CWE-532 |
| Affected lines | terraform-provider/internal/esxiresources/connection.go:132-136, terraform-provider/internal/esxiresources/connection.go:261-267, terraform-provider/internal/esxiresources/connection.go:293-297, terraform-provider/internal/esxiutils/connection.go:30-37 |

#### Summary

The password is correctly marked Sensitive and WriteOnly in the Terraform schema, but Create copies it into an FM request struct and formats that full struct into a diagnostic on API failure.

#### Validation

Static validation found direct source/control/sink evidence; confidence is lowered where runtime or deployment-specific behavior remains unproven. Validation details were not recorded separately.

Validation method: static code trace

#### Dataflow

The canonical finding records the affected path at terraform-provider/internal/esxiresources/connection.go:132-136, terraform-provider/internal/esxiresources/connection.go:261-267, terraform-provider/internal/esxiresources/connection.go:293-297, terraform-provider/internal/esxiutils/connection.go:30-37, but no expanded source-to-sink narrative was recorded.

#### Reachability

Reachability was not recorded beyond the canonical finding summary and affected locations.

#### Severity

**Medium** — The scan assigned medium severity; no separate canonical severity rationale was recorded.

Additional runtime or deployment evidence could raise or lower this severity.

#### Remediation

Never format structs containing secrets into diagnostics; implement redacted String methods or construct diagnostic messages from non-sensitive fields only.

<a id="finding-10"></a>

### [10] Deployment playbook installs privileged service and proxy configuration as world-writable

| Field | Value |
| --- | --- |
| Severity | medium |
| Confidence | high |
| Confidence rationale | Static validation found direct source/control/sink evidence; confidence is lowered where runtime or deployment-specific behavior remains unproven. |
| Category | Insecure file permissions / local privilege escalation |
| CWE | CWE-732 |
| Affected lines | playbooks/roles/tf/tasks/main.yml:64-69, playbooks/roles/tf/tasks/main.yml:82-87, playbooks/roles/tf/tasks/main.yml:89-101, playbooks/roles/tf/tasks/main.yml:114-120 |

#### Summary

The Ansible role copies systemd, HAProxy, and nginx configuration with world-writable or world-writable-like modes and creates code-coverage directories as 0777 before privileged services restart or serve those paths.

#### Validation

Static validation found direct source/control/sink evidence; confidence is lowered where runtime or deployment-specific behavior remains unproven. Validation details were not recorded separately.

Validation method: static code trace

#### Dataflow

The canonical finding records the affected path at playbooks/roles/tf/tasks/main.yml:64-69, playbooks/roles/tf/tasks/main.yml:82-87, playbooks/roles/tf/tasks/main.yml:89-101, playbooks/roles/tf/tasks/main.yml:114-120, but no expanded source-to-sink narrative was recorded.

#### Reachability

Reachability was not recorded beyond the canonical finding summary and affected locations.

#### Severity

**Medium** — The scan assigned medium severity; no separate canonical severity rationale was recorded.

Additional runtime or deployment evidence could raise or lower this severity.

#### Remediation

Use least-privilege file modes such as 0644 for root-owned config and 0750/0755 for directories, separate upload ownership from service config, and add Ansible assertions for expected modes.

## Reviewed Surfaces

| Surface | Risk Area | Outcome | Notes |
| --- | --- | --- | --- |
| Unauthenticated documentation upload writes attacker-controlled files to arbitrary directories | Arbitrary file write / missing authentication | Reported | The deployed Flask documentation/registry service exposes a POST upload route without authentication and converts the URL path into an absolute filesystem directory before creating it world-writable and saving the uploaded file. Evidence: artifacts/05_findings/CANON-002/candidate_ledger.jsonl |
| Terraform backend authorization is not scoped to the requested project | Authorization bypass / cross-project state access | Reported | The backend authorizes only a global TRAFFIC_CONTROL/write permission, then uses the caller-controlled project path to select the MongoDB state and lock document. Repository evidence does not show a per-project authorization check. Evidence: artifacts/05_findings/CANON-001/candidate_ledger.jsonl |
| Provider registry signing private keys are checked into the repository and used by build tooling | Supply-chain signing key exposure | Reported | GnuPG private key files are stored under the provider build directory, and build tooling uses that repository-local GPG home to sign provider checksum files advertised in registry metadata. Evidence: artifacts/05_findings/CANON-006/candidate_ledger.jsonl |
| Documentation registry service runs Flask debug mode on a network-facing listener | Debug mode exposure | Reported | The service defaults to 0.0.0.0 and starts Flask with debug=True, while the playbook service and HAProxy config expose the service behind a TLS frontend. Werkzeug debugger exploitability depends on runtime debugger/PIN behavior, but debug mode should not be enabled in production. Evidence: artifacts/05_findings/CANON-003/candidate_ledger.jsonl |
| Documentation route can render markdown files outside the docs directory | Path traversal / local file disclosure | Reported | The public documentation route joins URL path components into a markdown filename and then joins that value under the docs directory without validating the resolved path remains under docs. A res_type such as .. can select files one directory above docs when the route shape permits it. Evidence: artifacts/05_findings/CANON-009/candidate_ledger.jsonl |
| ESXi connection create diagnostics can disclose write-only vCenter passwords | Sensitive data exposure in diagnostics | Reported | The password is correctly marked Sensitive and WriteOnly in the Terraform schema, but Create copies it into an FM request struct and formats that full struct into a diagnostic on API failure. Evidence: artifacts/05_findings/CANON-004/candidate_ledger.jsonl |
| FM client logs full response and error bodies from authenticated API calls | Sensitive data exposure in logs | Reported | The shared FM client logs every response body and also embeds non-2xx response bodies in returned errors. Several FM responses can include sensitive provider-managed values such as connection data or AMX headers. Evidence: artifacts/05_findings/CANON-005/candidate_ledger.jsonl |
| AMX HTTP exporter Authorization headers are stored as non-sensitive Terraform state | Sensitive data exposure in Terraform state | Reported | The AMX HTTP exporter schema documents headers such as Authorization: Bearer but models headers and endpoints as normal strings/lists. The provider sends them to FM, reconstructs them from FM responses, and preserves them in state across reads. Evidence: artifacts/05_findings/CANON-007/candidate_ledger.jsonl |
| Deployment playbook installs privileged service and proxy configuration as world-writable | Insecure file permissions / local privilege escalation | Reported | The Ansible role copies systemd, HAProxy, and nginx configuration with world-writable or world-writable-like modes and creates code-coverage directories as 0777 before privileged services restart or serve those paths. Evidence: artifacts/05_findings/CANON-008/candidate_ledger.jsonl |
| Deployment ships checked-in HAProxy TLS private key material | Hardcoded private key material | Reported | The playbook repository contains TLS private key material and copies haproxy.pem into /etc/ssl/private for the public HAProxy listener. Anyone with repository access can impersonate or decrypt traffic for that deployed identity if the certificate is used. Evidence: artifacts/05_findings/CANON-010/candidate_ledger.jsonl |
