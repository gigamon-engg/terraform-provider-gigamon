#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

if ! command -v rg >/dev/null 2>&1; then
  echo "error: ripgrep (rg) is required for tools/security_lint.sh" >&2
  exit 2
fi

if [[ $# -gt 0 ]]; then
  mapfile -t TARGET_FILES < <(printf '%s\n' "$@")
else
  mapfile -t TARGET_FILES < <(rg --files internal -g '*.go')
fi

if [[ ${#TARGET_FILES[@]} -eq 0 ]]; then
  echo "No Go files selected for security lint."
  exit 0
fi

TMP_INPUT="$(mktemp)"
trap 'rm -f "$TMP_INPUT"' EXIT
printf '%s\n' "${TARGET_FILES[@]}" > "$TMP_INPUT"

echo "[security-lint] Checking secret-like schema fields are Sensitive + WriteOnly"

SCHEMA_ERRORS=0
while IFS= read -r file; do
  [[ -f "$file" ]] || continue

  if ! awk '
function count_char(s, c, t) {
  t = s
  return gsub(c, "", t)
}

function is_secret_name(n) {
  return (n ~ /(password|secret|token|authorization|api[_-]?key|access[_-]?key|private[_-]?key|client_secret|credential|headers)/)
}

BEGIN {
  in_attr = 0
  attr_name = ""
  depth = 0
  has_sensitive = 0
  has_write_only = 0
  is_secret = 0
  start_line = 0
}

{
  line = $0

  if (!in_attr) {
    if (match(line, /"([^"]+)"[[:space:]]*:[[:space:]]*schema\.[A-Za-z0-9_]+Attribute[[:space:]]*\{/, m)) {
      in_attr = 1
      attr_name = m[1]
      is_secret = is_secret_name(tolower(attr_name))
      has_sensitive = 0
      has_write_only = 0
      start_line = FNR
      depth = count_char(line, /\{/) - count_char(line, /\}/)

      if (line ~ /Sensitive:[[:space:]]*true/) {
        has_sensitive = 1
      }
      if (line ~ /WriteOnly:[[:space:]]*true/) {
        has_write_only = 1
      }

      if (depth <= 0) {
        if (is_secret && (!has_sensitive || !has_write_only)) {
          printf("%s:%d: attribute \"%s\" must set both Sensitive: true and WriteOnly: true\n", FILENAME, start_line, attr_name)
          bad = 1
        }
        in_attr = 0
      }
    }
  } else {
    if (line ~ /Sensitive:[[:space:]]*true/) {
      has_sensitive = 1
    }
    if (line ~ /WriteOnly:[[:space:]]*true/) {
      has_write_only = 1
    }

    depth += count_char(line, /\{/) - count_char(line, /\}/)

    if (depth <= 0) {
      if (is_secret && (!has_sensitive || !has_write_only)) {
        printf("%s:%d: attribute \"%s\" must set both Sensitive: true and WriteOnly: true\n", FILENAME, start_line, attr_name)
        bad = 1
      }
      in_attr = 0
    }
  }
}

END {
  exit(bad)
}
' "$file"; then
    SCHEMA_ERRORS=1
  fi
done < "$TMP_INPUT"

echo "[security-lint] Advisory scan for potential struct dumps in diagnostics/logs"
rg -n --no-heading 'fmt\.Sprintf\("[^"]*%v[^"]*"|fmt\.Printf\("[^"]*%v[^"]*"|tflog\.(Debug|Info|Warn|Error)\(.*%v' "${TARGET_FILES[@]}" \
  | rg -v 'redact|redacted|sanitize|sanitized' || true

echo "[security-lint] Advisory: review any %v diagnostics/log lines above and ensure secret fields are redacted before logging structs."

if [[ "$SCHEMA_ERRORS" -ne 0 ]]; then
  echo "[security-lint] FAILED"
  exit 1
fi

echo "[security-lint] PASSED"
