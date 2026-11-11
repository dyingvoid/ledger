# Load tests

k6 load tests for the ledger:

- `accounts.js` creates account metadata through
  `AccountService/CreateAccountInfo`, then registers each account in
  TigerBeetle through `AccountService/OpenLedgerAccount`;
- `deposit.js` deposits through `FundsService/Deposit`.

## Ledger API

Account creation is split into two steps:

```proto
service AccountService {
  rpc CreateAccountInfo(CreateAccountInfoRequest) returns (CreateAccountInfoResponse);
  rpc OpenLedgerAccount(OpenLedgerAccountRequest) returns (OpenLedgerAccountResponse);
}

message CreateAccountInfoRequest {
  string id = 1;
  int64 user_id = 2;
  string name = 3;
  string description = 4;
  int32 type = 5;
  string currency = 6;
}

message OpenLedgerAccountRequest {
  string id = 1;
}
```

Deposits take a caller-provided 128-bit transfer ID:

```proto
service FundsService {
  rpc Deposit(DepositRequest) returns (DepositResponse);
}

message DepositRequest {
  string id = 1;
  string account_id = 2;
  int64 amount_minor = 3;
  string currency = 4;
}
```

IDs are 128-bit values encoded as decimal or `0x`-prefixed hexadecimal
strings.

## Idempotency

The ID is the idempotency key everywhere:

- `CreateAccountInfo` inserts the provided ID into Postgres explicitly. The
  primary key conflict makes a repeated call with identical data return the
  existing account.
- `OpenLedgerAccount` uses TigerBeetle's native create-account idempotence.
- `Deposit` uses the provided ID as the TigerBeetle transfer ID, so TigerBeetle
  detects repeated transfers.

There is no separate idempotency table.

## Accounts

```bash
# create as many accounts as possible in 1 minute, starting at id 1
make loadtest-accounts

# or run k6 directly
k6 run -e DURATION=1m -e VUS=50 -e ACCOUNT_MIN_ID=1 loadtest/accounts.js
```

Each iteration performs `CreateAccountInfo` followed by `OpenLedgerAccount`
for the same ID. Both calls are idempotent, so re-running the seed test from
the same ID is safe.

The deposit path also needs a bank cash account (`account_type = 1000`) in the
deposit currency. The TigerBeetle deposit repository creates it lazily on the
first deposit.

## Deposits

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
and increases the account id every iteration. `MODE=steady` uses a constant
arrival rate; `MODE=ramp` increases it until saturation.

Every iteration generates a unique 128-bit transfer ID from the current
timestamp and iteration number. Reusing a transfer ID would return the
existing transfer instead of creating a new deposit.

## Rates

15,000,000 accounts, one deposit per account per period:

| Period | Average rate | k6 target |
|--------|--------------|-----------|
| 1 month | 5.79 /s | 6 /s |
| 2 weeks | 11.57 /s | 12 /s |
| 1 day | 173.61 /s | 174 /s |

k6 exits with code `99` when a threshold is crossed. That is expected in a
saturation run and is not a script failure. Thresholds are
`checks: rate > 0.99` and `grpc_req_duration: p(99) < 1000 ms` by default
(`MAX_FAIL_RATE`, `P99_MS`).

## Find the maximum sustainable rate

```bash
MODE=ramp ./loadtest/run-matrix.sh
```

Default stages: 6/s for 1m → 12/s for 2m → 174/s for 5m → 250/s for 5m.
Override with `RAMP_STAGES` (JSON array of `{target,duration}`) and
`START_RATE`.

Read saturation from:

- `grpc_req_duration` p99 growing super-linearly;
- non-OK gRPC codes (`UNAVAILABLE`, `DEADLINE_EXCEEDED`, `RESOURCE_EXHAUSTED`,
  `INTERNAL`);
- non-zero `dropped_iterations`;
- achieved request rate falling below the target.

## What the summary prints

- `*_requests` — RPCs attempted and achieved per-second rate;
- `*_ok` / `*_failed` — success percentage and failures split by gRPC status;
- `*_exceptions` — transport-level failures before a status was received;
- `dropped_iterations` — k6 could not keep up with its schedule;
- latency percentiles from `grpc_req_duration`.

## Files

| File | Purpose |
|------|---------|
| `loadtest/accounts.js` | create account info and open ledger accounts |
| `loadtest/deposit.js` | deposit scenario (deadline/steady/ramp) + summary |
| `loadtest/run-matrix.sh` | runs 6/12/174 (or ramp mode) with fresh slices |
| `loadtest/seed.env` | account id range used by both load tests |
| `loadtest/results/` | JSON summaries written by `run-matrix.sh` |
