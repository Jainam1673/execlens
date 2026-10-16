#!/usr/bin/env bash
# ==============================================================================
# ExecLens Report Verification Script
# ==============================================================================
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

cd "${REPO_ROOT}"

REPORT_DIR="${1:-reports/latest}"
README_PATH="${2:-README.md}"

echo "=========================================================="
echo "      ExecLens Report Audit & Verification Suite          "
echo "=========================================================="
echo "Report Dir:  ${REPORT_DIR}"
echo "README Path: ${README_PATH}"

if [ ! -f "./execlens" ]; then
    echo "Building execlens binary..."
    go build -trimpath -o execlens ./cmd/execlens
fi

./execlens report-verify --dir="${REPORT_DIR}" --readme="${README_PATH}"
