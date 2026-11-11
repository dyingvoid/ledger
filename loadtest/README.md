# Deposit load test

k6 load tests for ledger:

- `accounts.js` creates accounts through `AccountService/CreateAccount`,
  ids `1,2,3,...` until a `1m` deadline;
- `deposit.js` deposits through `FundsService/Deposit`, ids also starting at
  `1`, until a `1m` deadline (or at a steady/ramp arrival rate).

The deposit service exposes exactly one write path relevant here:

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

Every request also needs `x-idempotency-key` metadata. The last segment after
`:` is the external account ID, so the load test sends
`<unique-run-key>:<account_id>`. The server scopes the key by that external
account ID, so a key that repeats for the same account is a replay, not a new
deposit. The k6 script sends a unique key per request.

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

Accounts are created through the gRPC `AccountService/CreateAccount` endpoint
by `loadtest/accounts.js`. Each iteration creates one account: account ids
start at `ACCOUNT_MIN_ID` (default `1`) and increase by one until the
`DURATION` deadline (default `1m`).

The idempotency key ends with the account id, so the server stores the account
in TigerBeetle under that id. Creating an account that already exists is
treated as success: the run counts it as `exists` and continues.

```bash
# create as many accounts as possible in 1 minute, starting at id 1
make loadtest-accounts

# or run k6 directly
k6 run -e DURATION=1m -e VUS=50 -e ACCOUNT_MIN_ID=1 loadtest/accounts.js
```

The account range in `loadtest/seed.env` is `1` through `15000000`. Raise
`ACCOUNT_MIN_ID` to continue from where a previous run stopped.

The deposit path also needs a bank cash account (`account_type = 1000`) in the
deposit currency. The TigerBeetle deposit repository creates it lazily on the
first deposit.

## 2. Prepare the service

The defaults in `docker-compose.yml` are not a throughput configuration.
The idempotency store and the account/currency reference data still live in
Postgres, so raise its connection limit before drawing conclusions:

```bash
POSTGRES_MAX_CONNS=100 docker compose up -d --build --wait
```

Deposits themselves are written to TigerBeetle. Each deposit is still one
idempotency transaction in Postgres, so throughput is bounded by Postgres
round-trips plus the TigerBeetle transfer.

Also consider `LOG_LEVEL=warn` during the benchmark. At `info`, every account
creation and deposit writes a JSON log line, which is real work but usually not
the bottleneck.

## 3. Run the test

```bash
# 1 minute, ids 1,2,3,... until the deadline, as fast as the service allows
make loadtest-deposit

# all three rates, 1 minute each
./loadtest/run-matrix.sh
```

or one rate:

```bash
set -a; . ./.env; . ./loadtest/seed.env; set +a
k6 run \
  -e MODE=steady -e RATE=174 -e DURATION=1m \
  -e ACCOUNT_MIN_ID="$ACCOUNT_MIN_ID" \
  -e ACCOUNT_COUNT=5000000 \
  loadtest/deposit.js
```

`MODE=deadline` (the default) uses `VUS` concurrent workers for `DURATION`
and increases the account id every iteration. `MODE=steady` uses a
constant arrival rate; `MODE=ramp` increases it until saturation.

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

- **Account ids.** `exec.scenario.iterationInTest` is a global, zero-based
  iteration counter, so deadline mode deposits to `ACCOUNT_MIN_ID + iteration`
  and ids keep increasing until the deadline. Steady and ramp modes wrap with
  `ACCOUNT_MIN_ID + (iteration % ACCOUNT_COUNT)` so the arrival rate is kept
  for the full duration. Each matrix run still gets its own slice
  (`ACCOUNT_SLICE`, default 5,000,000 accounts) so separate runs start from
  different ids.
- **Slice size.** Any slice larger than zero works: steady/ramp runs reuse
  accounts once the range is used up. A larger slice only spreads the
  deposits over more ids.
- **Idempotency key.** `RUN_ID` is included in every key. Do not reuse a
  `RUN_ID`; a repeated key would return a replay instead of creating a
  deposit. The matrix generates a timestamped prefix automatically.
- **Account creation is idempotent.** `accounts.js` treats an existing account
  (gRPC `OK` with a replay header, or `ALREADY_EXISTS`) as success and keeps
  going. Re-running the seed test from the same id is safe.
- **Idempotency table growth.** Every successful deposit and account creation
  leaves one row for `IDEMPOTENCY_TTL` (default 24h). Short runs are fine;
  long soak tests will grow this table substantially.
- **k6 capacity.** One k6 instance is enough for the arrival-rate tests. For a
  deadline/max-throughput run, raise `VUS` until `dropped_iterations` is
  non-zero; if it stays non-zero while the service is healthy, the load
  generator is the limit. Run k6 on a separate machine for a production-like
  result.
- **Percentile sample size.** p99.9 needs roughly 100,000+ requests to be
  meaningful. A 1-minute run at 6/s produces only ~360 requests, so its
  p99.9 is noisy. Use 30–60 minutes for the low rates if p99.9 matters.
- **Reflection.** `REFLECT=true` (default) makes each VU load descriptors from
  the server once. Set `REFLECT=false` to load the proto file instead.

## Files

| File                    | Purpose                                             |
|-------------------------|-----------------------------------------------------|
| `loadtest/accounts.js`  | creates accounts 1,2,3,... through gRPC for 1m      |
| `loadtest/deposit.js`   | deposit scenario (deadline/steady/ramp) + summary   |
| `loadtest/run-matrix.sh`    | runs 6/12/174 (or ramp mode) with fresh slices   |
| `loadtest/seed.env`         | account id range used by both load tests         |
| `loadtest/results/`         | JSON summaries written by `run-matrix.sh`        |
