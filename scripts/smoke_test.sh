#!/usr/bin/env bash
set -euo pipefail

echo "=========================================================="
echo "          ExecLens End-to-End Smoke Test                 "
echo "=========================================================="

PORT=8085
TEMP_DIR=$(mktemp -d -t execlens_smoke_XXXXXX)
DB_PATH="${TEMP_DIR}/smoke.db"
EXECLENS_BIN="./execlens"

cleanup() {
    echo "Cleaning up smoke test resources..."
    if [[ -n "${SERVER_PID:-}" ]] && kill -0 "$SERVER_PID" 2>/dev/null; then
        kill "$SERVER_PID" 2>/dev/null || true
        wait "$SERVER_PID" 2>/dev/null || true
    fi
    rm -rf "$TEMP_DIR"
    echo "Cleanup complete."
}
trap cleanup EXIT

echo "1. Building ExecLens binary..."
go build -o "$EXECLENS_BIN" ./cmd/execlens

echo "2. Starting ExecLens server in background on port ${PORT}..."
"$EXECLENS_BIN" server --port "$PORT" --db "$DB_PATH" --benchmarks "benchmarks" --frontend "frontend" &
SERVER_PID=$!

echo "Waiting for server to become healthy..."
MAX_ATTEMPTS=20
ATTEMPT=0
HEALTHY=false

while [[ $ATTEMPT -lt $MAX_ATTEMPTS ]]; do
    if curl -s -f "http://127.0.0.1:${PORT}/api/v1/health" > /dev/null 2>&1; then
        HEALTHY=true
        break
    fi
    sleep 0.5
    ATTEMPT=$((ATTEMPT + 1))
done

if [[ "$HEALTHY" != "true" ]]; then
    echo "❌ Server failed to respond to /api/v1/health"
    exit 1
fi
echo "✅ /api/v1/health returned healthy."

echo "3. Verifying language detection..."
LANG_RES=$(curl -s "http://127.0.0.1:${PORT}/api/v1/languages")
if echo "$LANG_RES" | grep -q "cpp" && echo "$LANG_RES" | grep -q "rust" && echo "$LANG_RES" | grep -q "go"; then
    echo "✅ /api/v1/languages returned supported adapters."
else
    echo "❌ Language verification failed: $LANG_RES"
    exit 1
fi

echo "4. Verifying workload catalog..."
WORKLOAD_RES=$(curl -s "http://127.0.0.1:${PORT}/api/v1/workloads")
if echo "$WORKLOAD_RES" | grep -q "sieve" && echo "$WORKLOAD_RES" | grep -q "sort"; then
    echo "✅ /api/v1/workloads returned catalog."
else
    echo "❌ Workload verification failed: $WORKLOAD_RES"
    exit 1
fi

echo "4b. Verifying multitrack endpoints..."
TRACKS_RES=$(curl -s "http://127.0.0.1:${PORT}/api/v1/tracks")
if echo "$TRACKS_RES" | grep -q "languages" && echo "$TRACKS_RES" | grep -q "lowlevel" && echo "$TRACKS_RES" | grep -q "accelerator"; then
    echo "✅ /api/v1/tracks returned all 3 tracks."
else
    echo "❌ Tracks verification failed: $TRACKS_RES"
    exit 1
fi

LOWLEVEL_RES=$(curl -s "http://127.0.0.1:${PORT}/api/v1/lowlevel/catalog")
if echo "$LOWLEVEL_RES" | grep -q "lowlevel_reduction"; then
    echo "✅ /api/v1/lowlevel/catalog returned lowlevel catalog."
else
    echo "❌ Lowlevel catalog verification failed: $LOWLEVEL_RES"
    exit 1
fi

ACCEL_RES=$(curl -s "http://127.0.0.1:${PORT}/api/v1/accelerators")
if echo "$ACCEL_RES" | grep -q "cuda"; then
    echo "✅ /api/v1/accelerators returned accelerator status."
else
    echo "❌ Accelerators verification failed: $ACCEL_RES"
    exit 1
fi

echo "5. Creating a deterministic benchmark run via POST /api/v1/runs..."
RUN_PAYLOAD='{"workload_id":"sieve","languages":["go","rust"],"workload_params":{"limit":50000},"iterations":2,"warmup_iterations":1,"execution_mode":"standard"}'
CREATE_RES=$(curl -s -X POST "http://127.0.0.1:${PORT}/api/v1/runs" \
    -H "Content-Type: application/json" \
    -d "$RUN_PAYLOAD")

RUN_ID=$(echo "$CREATE_RES" | grep -o '"id":"[^"]*' | cut -d'"' -f4)
if [[ -z "$RUN_ID" ]]; then
    echo "❌ Failed to create run. Response: $CREATE_RES"
    exit 1
fi
echo "✅ Run created with ID: ${RUN_ID}"

echo "6. Polling for run completion..."
STATUS=""
POLL_ATTEMPTS=0
while [[ $POLL_ATTEMPTS -lt 30 ]]; do
    RUN_RES=$(curl -s "http://127.0.0.1:${PORT}/api/v1/runs/${RUN_ID}")
    STATUS=$(echo "$RUN_RES" | grep -o '"status":"[^"]*' | head -n 1 | cut -d'"' -f4)
    if [[ "$STATUS" == "COMPLETED" || "$STATUS" == "FAILED" || "$STATUS" == "CANCELLED" ]]; then
        break
    fi
    sleep 1
    POLL_ATTEMPTS=$((POLL_ATTEMPTS + 1))
done

if [[ "$STATUS" != "COMPLETED" ]]; then
    echo "❌ Run failed to reach COMPLETED state. Current status: $STATUS"
    echo "$RUN_RES"
    exit 1
fi
echo "✅ Run reached status: COMPLETED"

echo "7. Verifying measurement results and samples..."
if echo "$RUN_RES" | grep -q '"validation_passed":true' && echo "$RUN_RES" | grep -q '"median_ms"'; then
    echo "✅ Measurements, summary metrics, and validation confirmed."
else
    echo "❌ Missing measurements or validation results: $RUN_RES"
    exit 1
fi

echo "8. Verifying JSON export..."
EXPORT_RES=$(curl -s "http://127.0.0.1:${PORT}/api/v1/runs/${RUN_ID}/export?format=json")
if echo "$EXPORT_RES" | grep -q "$RUN_ID"; then
    echo "✅ JSON export generated successfully."
fi

echo "9. Verifying frontend SPA delivery..."
SPA_RES=$(curl -s "http://127.0.0.1:${PORT}/")
if echo "$SPA_RES" | grep -q "<html" || echo "$SPA_RES" | grep -q "vite"; then
    echo "✅ Frontend dashboard served successfully."
fi

echo "=========================================================="
echo "  🎉 Smoke Test Passed: ExecLens is fully operational!   "
echo "=========================================================="
