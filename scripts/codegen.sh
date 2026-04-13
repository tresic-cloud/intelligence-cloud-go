#!/usr/bin/env bash
set -euo pipefail

# codegen.sh — Runs oapi-codegen against the pinned OpenAPI spec.
#
# Usage:  scripts/codegen.sh
#
# Produces:
#   internal/generated/types.gen.go
#   internal/generated/client.gen.go
#
# KNOWN ISSUE (2026-04-13): The pinned canonical OpenAPI document is OpenAPI
# 3.1, which oapi-codegen v2.6.0 does not yet fully support
# (https://github.com/oapi-codegen/oapi-codegen/issues/373). Real codegen
# emits a name collision between the schema-level `LoginResponse` type and
# the WithResponses wrapper of the same name. As a stop-gap the generated
# files in this repo are hand-curated to match what oapi-codegen v2 would
# produce once 3.1 support lands (or once IDB-1354 downgrades the canonical
# spec to 3.0). Run this script when either condition is true; if the diff
# is clean, drop this comment.

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
SPEC="${REPO_ROOT}/testdata/openapi.yaml"
OUTDIR="${REPO_ROOT}/internal/generated"

# Verify oapi-codegen is on PATH.
if ! command -v oapi-codegen &>/dev/null; then
    echo "ERROR: oapi-codegen not found on PATH." >&2
    echo "Install via: go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest" >&2
    exit 1
fi

# Verify spec exists.
if [[ ! -f "${SPEC}" ]]; then
    echo "ERROR: ${SPEC} not found. Run scripts/pull-openapi.sh first." >&2
    exit 1
fi

# Ensure output directory exists.
mkdir -p "${OUTDIR}"

echo "Generating types from ${SPEC}..."
oapi-codegen --config "${REPO_ROOT}/oapi-codegen-types.yaml" "${SPEC}"

echo "Generating client from ${SPEC}..."
oapi-codegen --config "${REPO_ROOT}/oapi-codegen-client.yaml" "${SPEC}"

# Format generated files (usually a no-op since oapi-codegen formats output).
gofmt -w "${OUTDIR}"/*.gen.go

echo "Code generation complete."
echo "  ${OUTDIR}/types.gen.go"
echo "  ${OUTDIR}/client.gen.go"
