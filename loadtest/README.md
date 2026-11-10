# Deposit load test

Load test for `ledger.funds.v1.FundsService/Deposit`.

The service exposes exactly one write path relevant here:

```proto
service FundsService {
  rpc Deposit(DepositRequest) returns (DepositResponse);
}

message DepositRequest {
  int64 account_id = 1;
  int64 amount_minor = 2;
  string currency = 3;
}
```

Every request also needs `x-idempotency-key` metadata. The server turns it into
`funds.deposit:<account_id>:<key>`, so a key that repeats for the same account
is a replay, not a new deposit. The k6 script sends a unique key per request.

The server registers gRPC reflection, so k6 does not need a local copy of the
`.proto` file. Set `REFLECT=false` to load `proto/ledger/funds/v1/funds.proto`
instead. The default import root is `../proto` (relative to `loadtest/`); the
k6 version's proto loader resolves paths relative to the script file.

## What the rates mean

15,000,000 accounts, one deposit per account per period:

| Period    | Calculation                    | Average rate | k6 target |
|-----------|--------------------------------|--------------|-----------|
| 1 month   | 15,000,000 / 2,592,000 s       | 5.79 /s      | 6 /s      |
| 2 weeks   | 30,000,000 / 2,592,000 s       | 11.57 /s     | 12 /s     |
| 1 day     | 450,000,000 / 2,592,000 s      | 173.61 /s    | 174 /s    |

The 2-week and daily rows use 30M and 450M deposits per 30-day month, because
each account deposits once per period (2x and 30x per month respectively).

k6 arrival-rate executors require integer rates, so the targets are rounded to
6, 12 and 174 per second. The monthly and biweekly targets are ~4% above the
exact average; the daily target is within 0.3%.

These are steady-state averages, not burst sizes. That is why the test uses
`constant-arrival-rate` (open model: k6 starts the next request on schedule
regardless of how many are still in flight).

"Concurrent deposits" is then a derived number, not a knob:

```
in-flight requests ≈ achieved requests/s × latency
```

For example, 174 requests/s at 200 ms latency is only ~35 concurrent
requests; at 2 s latency it is ~350. The VU count is sized from that estimate
and can be overridden with `VUS`.

## 1. Accounts

Migration `000005_seed_deposit_accounts` creates the 15,000,000 client deposit
accounts directly in Postgres on a fresh database, in one transaction. The load
test itself creates no accounts: `deposit.js` only calls
`FundsService/Deposit`.

On a fresh database the seeded range is ids `4` through `15000003`
(ids 1–3 are the bank cash accounts). `loadtest/seed.env` records that range as
`ACCOUNT_MIN_ID`, `ACCOUNT_MAX_ID`, and `ACCOUNT_COUNT`; keep it in sync if your
database already had accounts before the migration ran.

Verify the range after `make migrate-up`:

```bash
docker compose exec -T postgres psql -U ledger -d ledger -c \
  "SELECT count(*), min(id), max(id)
   FROM ledger.account
   WHERE account_type = 2100 AND currency = 'USD';"
```

Expected: `15000000`, `4`, `15000003`.

The deposit function also needs a bank cash account (`account_type = 1000`) in
the deposit currency; the migration creates one per seeded currency.

## 2. Prepare the service

The defaults in `docker-compose.yml` are not a throughput configuration:

```bash
POSTGRES_MAX_CONNS=10 docker compose up -d --build --wait
```

Each deposit is one Postgres transaction that inserts/updates an idempotency
row and calls `ledger.add_deposit()`. With 10 pool connections, throughput is
bounded by `10 / transaction_duration`. Raise `POSTGRES_MAX_CONNS` (and
Postgres `max_connections`) before drawing conclusions:

```bash
POSTGRES_MAX_CONNS=100 docker compose up -d --build --wait
```

Also consider `LOG_LEVEL=warn` during the benchmark. At `info`, every deposit
writes a JSON log line, which is real work but usually not the bottleneck.

### Important: the bank cash row is a global lock

`ledger.add_deposit()` runs this before writing:

```sql
SELECT a.id
FROM ledger.account a
WHERE a.account_type = 1000
  AND a.currency = p_currency
  AND a.deleted_at IS NULL
ORDER BY a.id
LIMIT 1
FOR UPDATE;
```

Every deposit in a currency takes a `FOR UPDATE` lock on **the same bank cash
account row**, then updates that row (`dr = dr + amount`). Deposits in one
currency are therefore serialized at the database, regardless of how many app
instances or gRPC connections you add. The 174/s daily target is the test
that will expose this. Running the test with one currency measures the worst
case; multiple currencies get one lock domain each.

If the service saturates well below 174/s while Postgres CPU is low, look at:

```sql
SELECT pid, wait_event_type, wait_event, state, query
FROM pg_stat_activity
WHERE datname = current_database();

SELECT relation::regclass, mode, granted, count(*)
FROM pg_locks
WHERE relation = 'ledger.account'::regclass
GROUP BY 1, 2, 3;
```

## 3. Run the test

```bash
# all three rates, 1 minute each
./loadtest/run-matrix.sh
```

or one rate:

```bash
set -a; . ./.env; . ./loadtest/seed.env; set +a
k6 run \
  -e RATE=174 -e DURATION=1m \
  -e ACCOUNT_MIN_ID="$ACCOUNT_MIN_ID" \
  -e ACCOUNT_COUNT=5000000 \
  loadtest/deposit.js
```

k6 exits with code `99` when a threshold is crossed. That is expected in a
saturation run and is not a script failure; the runner treats it as a normal
result. Thresholds are `checks: rate > 0.99` and `grpc_req_duration: p(99) <
1000 ms` by default (`MAX_FAIL_RATE`, `P99_MS`).

### Find the maximum sustainable rate

Ramp mode increases the arrival rate over time until latency or errors run
away:

```bash
MODE=ramp ./loadtest/run-matrix.sh
```

Default stages: 6/s for 1m → 12/s for 2m → 174/s for 5m → 250/s for 5m.
Override with `RAMP_STAGES` (JSON array of `{target,duration}`) and
`START_RATE`.

Read the point where one of these happens as saturation:

- `grpc_req_duration` p99 starts growing super-linearly;
- non-OK gRPC codes appear (especially `UNAVAILABLE`, `DEADLINE_EXCEEDED`,
  `RESOURCE_EXHAUSTED`, `INTERNAL`);
- `dropped_iterations` becomes non-zero (k6 could not start every scheduled
  request; raise `VUS` only if the service is not already saturated);
- the achieved request rate (`deposit_requests` rate in the summary) falls
  below the target rate.

Then run a steady test just below and just above that rate to confirm.

## 4. What the summary prints

Each run prints and optionally writes a JSON summary (`OUT`) with:

- `deposit_requests` — RPCs attempted, and achieved per-second rate;
- `deposit_ok` / `deposit_failed` percentages, failed requests split by gRPC
  status code;
- `deposit_exceptions` — transport-level failures before a status was
  received;
- `deposit_replayed` — should be 0; non-zero means an idempotency key
  repeated;
- `dropped_iterations` — k6 could not keep up with its own schedule;
- latency percentiles from `grpc_req_duration`: p50, p90, p95, p99, p99.9,
  max, plus the same for successful deposits only (`deposit_ok_duration`).

A useful result table:

| Rate (target) | Achieved req/s | OK % | Failed by code | p50 | p95 | p99 | p99.9 | Max | Dropped |
|---------------|----------------|------|----------------|-----|-----|-----|-------|-----|---------|
| 6             |                |      |                |     |     |     |       |     |         |
| 12            |                |      |                |     |     |     |       |     |         |
| 174           |                |      |                |     |     |     |       |     |         |

## 5. Test design notes and limits

- **Account reuse.** `exec.scenario.iterationInTest` is a global, zero-based
  iteration counter, so `ACCOUNT_MIN_ID + (iteration % ACCOUNT_COUNT)` gives
  each iteration an account from the seeded range and wraps around when the
  range is exhausted. The target rate is therefore kept for the full duration,
  even if the run performs more deposits than there are accounts. Each
  matrix run still gets its own slice (`ACCOUNT_SLICE`, default 5,000,000
  accounts) so separate runs start from different ids.
- **Slice size.** Any slice larger than zero works: the test no longer skips
  requests when the range is used up, it just starts reusing accounts. A
  larger slice only spreads the deposits over more ids.
- **Idempotency key.** `RUN_ID` is included in every key. Do not reuse a
  `RUN_ID`; a repeated key would return a replay instead of creating a
  deposit. The matrix generates a timestamped prefix automatically.
- **15M accounts vs. 5M active accounts.** The table and its indexes are
  large, so every lookup sees realistic index depth. Most deposits land in the
  first few million ids during a short test; the long tail is not exercised.
  That is intentional and does not change the hot-row contention behaviour.
- **Idempotency table growth.** Every successful deposit leaves one row for
  `IDEMPOTENCY_TTL` (default 24h). A full day at 174/s is 15M rows. Short
  runs are fine; long soak tests will grow this table substantially.
- **k6 capacity.** One k6 instance is enough for 174/s, but run it on a
  separate machine from the service and DB for a production-like result. If
  `dropped_iterations` is non-zero while the service is healthy, the load
  generator or `VUS` is the limit.
- **Percentile sample size.** p99.9 needs roughly 100,000+ requests to be
  meaningful. A 1-minute run at 6/s produces only ~360 requests, so its
  p99.9 is noisy. Use 30–60 minutes for the low rates if p99.9 matters.
- **Reflection.** `REFLECT=true` (default) makes each VU load descriptors from
  the server once. Set `REFLECT=false` to load the proto file instead.

## Files

| File                    | Purpose                                             |
|-------------------------|-----------------------------------------------------|
| `loadtest/deposit.js`   | k6 scenario, metrics, summary                       |
| `loadtest/run-matrix.sh`    | runs 6/12/174 (or ramp mode) with fresh slices   |
| `loadtest/seed.env`         | account range seeded by migration 000005         |
| `loadtest/results/`         | JSON summaries written by `run-matrix.sh`        |
