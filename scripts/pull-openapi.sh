#!/usr/bin/env bash
set -euo pipefail

# pull-openapi.sh — Fetches and pins the canonical OpenAPI spec from the
# intelligence-cloud sibling repository.
#
# Usage:  scripts/pull-openapi.sh [--ref <branch-or-sha>]
#
# Environment:
#   IC_REPO_PATH  Path to the intelligence-cloud repository checkout.
#                 Default: ../intelligence-cloud (relative to this repo root)

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
IC_REPO_PATH="${IC_REPO_PATH:-${REPO_ROOT}/../intelligence-cloud}"
REF="main"

# Parse optional --ref argument.
while [[ $# -gt 0 ]]; do
    case "$1" in
        --ref)
            REF="$2"
            shift 2
            ;;
        *)
            echo "ERROR: Unknown argument: $1" >&2
            echo "Usage: scripts/pull-openapi.sh [--ref <branch-or-sha>]" >&2
            exit 1
            ;;
    esac
done

# Validate sibling repo exists.
if [[ ! -d "${IC_REPO_PATH}" ]]; then
    echo "ERROR: intelligence-cloud repository not found at ${IC_REPO_PATH}" >&2
    echo "Set IC_REPO_PATH to the correct location or clone the repository." >&2
    exit 1
fi

SRC="${IC_REPO_PATH}/backend/docs/api/openapi.yaml"

# Check out the requested ref in the sibling repo.
(cd "${IC_REPO_PATH}" && git checkout "${REF}" --quiet)

# Validate source file exists.
if [[ ! -f "${SRC}" ]]; then
    echo "ERROR: OpenAPI spec not found at ${SRC}" >&2
    exit 1
fi

# Validate YAML is parseable.
if command -v python3 &>/dev/null; then
    python3 -c "import yaml, sys; yaml.safe_load(open(sys.argv[1]))" "${SRC}" 2>/dev/null || {
        echo "ERROR: ${SRC} is not valid YAML." >&2
        exit 1
    }
fi

# Ensure target directory exists.
mkdir -p "${REPO_ROOT}/testdata"

# Copy the spec.
cp "${SRC}" "${REPO_ROOT}/testdata/openapi.yaml"

# Record the commit SHA.
SHA="$(cd "${IC_REPO_PATH}" && git rev-parse HEAD)"
printf '%s\n' "${SHA}" > "${REPO_ROOT}/testdata/openapi.commit"

echo "Pinned openapi.yaml at commit ${SHA}"
