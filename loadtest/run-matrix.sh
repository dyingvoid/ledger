#!/usr/bin/env bash
#
# Run the deposit load test for the three target rates.
#
# 15M accounts, one deposit per account per period:
#
#   6   deposits/s  = one deposit per account per month   (5.79/s exact)
#   12  deposits/s  = one deposit per account per 2 weeks (11.57/s exact)
#   174 deposits/s  = one deposit per account per day     (173.61/s exact)
#
# Each run gets its own contiguous slice of the seeded accounts so that no
# account receives a second deposit between runs.
#
# Usage:
#   ./loadtest/run-matrix.sh
#   RATES="6 12" DURATION=10m ./loadtest/run-matrix.sh
#   MODE=ramp ./loadtest/run-matrix.sh          # ramp to find max throughput
#
# Env:
#   RATES          default "6 12 174"
#   DURATION       default 5m (steady mode)
#   MODE           steady (default) or ramp
#   RAMP_STAGES    JSON stages used in ramp mode
#   ACCOUNT_SLICE  accounts reserved per steady run (default 5000000)
#   VUS            override VU count (default computed by the k6 script)
#   RUN_ID         prefix for run ids (default timestamp)
#   K6             k6 binary (default k6)
#   TARGET         gRPC address (default 127.0.0.1:50051)
#   EXTRA_ARGS     extra args passed to k6 run

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

if [[ -f .env ]]; then
  set -a
  # shellcheck disable=SC1091
  . ./.env
  set +a
fi

if [[ ! -f loadtest/seed.env ]]; then
  echo "loadtest/seed.env not found; it must match the range seeded by migration 000005" >&2
  exit 1
fi
# shellcheck disable=SC1091
. loadtest/seed.env

MODE="${MODE:-steady}"
RATES="${RATES:-6 12 174}"
DURATION="${DURATION:-5m}"
ACCOUNT_SLICE="${ACCOUNT_SLICE:-5000000}"
RUN_ID="${RUN_ID:-$(date +%Y%m%d-%H%M%S)}"
K6="${K6:-k6}"
TARGET="${TARGET:-127.0.0.1:50051}"
RAMP_STAGES="${RAMP_STAGES:-}"
if [[ -z "$RAMP_STAGES" ]]; then
  RAMP_STAGES='[{"target":6,"duration":"1m"},{"target":12,"duration":"2m"},{"target":174,"duration":"5m"},{"target":250,"duration":"5m"}]'
fi
RESULTS_DIR="$ROOT/loadtest/results"
mkdir -p "$RESULTS_DIR"

if ! command -v "$K6" >/dev/null 2>&1; then
  echo "$K6 not found in PATH" >&2
  exit 1
fi

common_env=(
  -e "TARGET=$TARGET"
  -e "REFLECT=${REFLECT:-true}"
  -e "CURRENCY=${SEED_CURRENCY:-USD}"
  -e "AMOUNT_MINOR=${AMOUNT_MINOR:-100}"
  -e "VU_SECONDS=${VU_SECONDS:-0.5}"
  -e "RPC_TIMEOUT=${RPC_TIMEOUT:-10s}"
  -e "MAX_FAIL_RATE=${MAX_FAIL_RATE:-0.01}"
  -e "P99_MS=${P99_MS:-1000}"
)

if [[ -n "${VUS:-}" ]]; then
  common_env+=(-e "VUS=$VUS")
fi
if [[ -n "${EXTRA_ARGS:-}" ]]; then
  # shellcheck disable=SC2206
  extra=($EXTRA_ARGS)
else
  extra=()
fi

run_k6() {
  local name="$1"
  shift
  local out="$RESULTS_DIR/${RUN_ID}-${name}.json"

  echo
  echo "=================================================================="
  echo "run: $name"
  echo "out: $out"
  echo "=================================================================="

  set +e
  "$K6" run "${common_env[@]}" "${extra[@]}" -e "OUT=$out" "$@" loadtest/deposit.js
  local rc=$?
  set -e

  # 99 = threshold crossed, which is an expected result for a saturation test.
  if ((rc != 0 && rc != 99)); then
    echo "k6 exited with $rc" >&2
    exit "$rc"
  fi
}

if [[ "$MODE" == "ramp" ]]; then
  count=$ACCOUNT_COUNT
  if ((count > ACCOUNT_SLICE)); then
    count=$ACCOUNT_SLICE
  fi
  run_k6 "ramp" \
    -e "MODE=ramp" \
    -e "RUN_ID=${RUN_ID}-ramp" \
    -e "ACCOUNT_MIN_ID=$ACCOUNT_MIN_ID" \
    -e "ACCOUNT_COUNT=$count" \
    -e "RAMP_STAGES=$RAMP_STAGES"
  exit 0
fi

index=0
for rate in $RATES; do
  offset=$((index * ACCOUNT_SLICE))
  if ((offset >= ACCOUNT_COUNT)); then
    echo "account range exhausted at rate $rate (offset $offset >= $ACCOUNT_COUNT)" >&2
    exit 1
  fi
  slice_min=$((ACCOUNT_MIN_ID + offset))
  slice_count=$ACCOUNT_SLICE
  if ((offset + slice_count > ACCOUNT_COUNT)); then
    slice_count=$((ACCOUNT_COUNT - offset))
  fi

  run_k6 "rate-${rate}" \
    -e "MODE=steady" \
    -e "RATE=$rate" \
    -e "DURATION=$DURATION" \
    -e "RUN_ID=${RUN_ID}-r${rate}" \
    -e "ACCOUNT_MIN_ID=$slice_min" \
    -e "ACCOUNT_COUNT=$slice_count"

  index=$((index + 1))
done

echo
echo "raw summaries:"
ls -1 "$RESULTS_DIR"/"${RUN_ID}"-*.json 2>/dev/null || true
