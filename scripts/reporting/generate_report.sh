#!/usr/bin/env bash
# ==============================================================================
# ExecLens Automated Report Generation Script
# ==============================================================================
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

cd "${REPO_ROOT}"

OUTPUT_DIR="${1:-reports}"
BASELINE="${2:-cpp}"
DB_PATH="${3:-execlens.db}"

echo "=========================================================="
echo "      ExecLens Automated Report & Chart Generator         "
echo "=========================================================="
echo "Output Directory: ${OUTPUT_DIR}"
echo "Baseline:         ${BASELINE}"
echo "Database:         ${DB_PATH}"

if [ ! -f "./execlens" ]; then
    echo "Building execlens binary..."
    go build -trimpath -o execlens ./cmd/execlens
fi

./execlens report-gen \
    --db="${DB_PATH}" \
    --output="${OUTPUT_DIR}" \
    --baseline="${BASELINE}" \
    --update-readme=true \
    --readme=README.md

echo "Report generation completed successfully."
