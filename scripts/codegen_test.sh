#!/usr/bin/env bash
set -euo pipefail

# codegen_test.sh — Verifies that re-running code generation produces no
# changes to the committed generated files. This catches cases where the
# pinned OpenAPI spec was updated but codegen was not re-run, or where
# hand-edits crept into generated files.
#
# Usage:  scripts/codegen_test.sh
# Exit 0: Generated code is up to date.
# Exit 1: Drift detected — generated code differs from committed version.

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"

echo "Re-running code generation..."
bash "${REPO_ROOT}/scripts/codegen.sh"

echo "Checking for drift in internal/generated/..."
if ! git diff --exit-code "${REPO_ROOT}/internal/generated/"; then
    echo "" >&2
    echo "ERROR: Generated code is out of date." >&2
    echo "The files in internal/generated/ differ after re-running codegen." >&2
    echo "Run 'scripts/codegen.sh' and commit the updated files." >&2
    exit 1
fi

echo "No drift detected. Generated code is up to date."
