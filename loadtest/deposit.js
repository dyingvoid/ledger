// k6 load test for ledger FundsService.Deposit.
//
// One iteration = one Deposit RPC = one deposit on its own account.
// Accounts are taken from a contiguous, pre-seeded ID range. The global
// scenario iteration index guarantees that no two iterations use the same
// account until the whole range has been consumed.
//
// Required env:
//   ACCOUNT_MIN_ID      first account id of the seeded range (from seed.env)
//   ACCOUNT_COUNT       number of accounts in the range (from seed.env)
//
// Common env:
//   MODE=steady|ramp    default: steady
//   RATE                steady arrival rate per second (default 6)
//   DURATION            steady duration (default 5m)
//   RAMP_STAGES         JSON stages for ramp mode
//   VUS                 pre-allocated/max VUs (default: peak_rate * VU_SECONDS)
//   VU_SECONDS          assumed latency used to size VUs (default 0.5)
//   TARGET              gRPC address (default 127.0.0.1:50051)
//   REFLECT             use server reflection (default true)
//   PROTO_IMPORT_PATH   proto import root when REFLECT=false (default ../proto,
//                       relative to this script)
//   CURRENCY            deposit currency (default USD)
//   AMOUNT_MINOR        deposit amount (default 100)
//   RUN_ID              unique run id; MUST change between runs
//   OUT                 path for the raw JSON summary
//   MAX_FAIL_RATE       threshold, default 0.01
//   P99_MS              threshold, default 1000
//   RPC_TIMEOUT         per-RPC timeout, default 10s
//
// Rate reference for 15M accounts where each account deposits once per period:
//   1 month   15M / 30d  = 5.79/s  -> 6/s
//   2 weeks   30M / 30d  = 11.57/s -> 12/s
//   1 day     450M / 30d = 173.61/s -> 174/s

import grpc from 'k6/net/grpc';
import { check } from 'k6';
import { Counter, Trend } from 'k6/metrics';
import exec from 'k6/execution';

const MODE = (__ENV.MODE || 'steady').toLowerCase();
const TARGET = __ENV.TARGET || '127.0.0.1:50051';
const PLAINTEXT = (__ENV.PLAINTEXT || 'true') === 'true';
const RPC_TIMEOUT = __ENV.RPC_TIMEOUT || '10s';
const REFLECT = (__ENV.REFLECT || 'true') === 'true';
const PROTO_IMPORT_PATH = __ENV.PROTO_IMPORT_PATH || '../proto';
const CURRENCY = __ENV.CURRENCY || 'USD';
const AMOUNT_MINOR = Number(__ENV.AMOUNT_MINOR || 100);
const RUN_ID = __ENV.RUN_ID || 'adhoc';

const RATE = Number(__ENV.RATE || 6);
const DURATION = __ENV.DURATION || '5m';

const DEFAULT_RAMP_STAGES =
  '[{"target":6,"duration":"1m"},{"target":12,"duration":"2m"},' +
  '{"target":174,"duration":"5m"},{"target":250,"duration":"5m"}]';
const RAMP_STAGES = JSON.parse(__ENV.RAMP_STAGES || DEFAULT_RAMP_STAGES);

const ACCOUNT_MIN_ID = Number(__ENV.ACCOUNT_MIN_ID || 4);
const ACCOUNT_COUNT = Number(__ENV.ACCOUNT_COUNT || 15000000);

const VU_SECONDS = Number(__ENV.VU_SECONDS || 0.5);
const peakRate = MODE === 'ramp'
  ? Math.max(Number(__ENV.START_RATE || 1), ...RAMP_STAGES.map((s) => Number(s.target)))
  : RATE;
const VUS = Number(__ENV.VUS || Math.max(20, Math.ceil(peakRate * VU_SECONDS)));

const MAX_FAIL_RATE = Number(__ENV.MAX_FAIL_RATE || 0.01);
const P99_MS = Number(__ENV.P99_MS || 1000);

if (!Number.isFinite(ACCOUNT_MIN_ID) || !Number.isFinite(ACCOUNT_COUNT) || ACCOUNT_COUNT <= 0) {
  throw new Error('ACCOUNT_MIN_ID and ACCOUNT_COUNT must be set to the seeded account range');
}
if (!Number.isFinite(VUS) || VUS <= 0) {
  throw new Error('VUS must be a positive number');
}
if (MODE === 'steady' && (!Number.isFinite(RATE) || RATE <= 0)) {
  throw new Error('RATE must be a positive number');
}

const scenario = MODE === 'ramp'
  ? {
      executor: 'ramping-arrival-rate',
      startRate: Number(__ENV.START_RATE || 1),
      timeUnit: '1s',
      stages: RAMP_STAGES,
      preAllocatedVUs: VUS,
      maxVUs: VUS,
      gracefulStop: '30s',
      tags: { mode: 'ramp', run_id: RUN_ID },
    }
  : {
      executor: 'constant-arrival-rate',
      rate: RATE,
      timeUnit: '1s',
      duration: DURATION,
      preAllocatedVUs: VUS,
      maxVUs: VUS,
      gracefulStop: '30s',
      tags: { mode: 'steady', rate: String(RATE), run_id: RUN_ID },
    };

export const options = {
  scenarios: { deposits: scenario },
  summaryTrendStats: ['avg', 'min', 'med', 'p(50)', 'p(90)', 'p(95)', 'p(99)', 'p(99.9)', 'max'],
  thresholds: {
    checks: [`rate>${Math.max(0, 1 - MAX_FAIL_RATE)}`],
    grpc_req_duration: [`p(99)<${P99_MS}`],
  },
};

// gRPC status code -> human name.
const CODE_NAMES = {
  0: 'OK',
  1: 'CANCELLED',
  2: 'UNKNOWN',
  3: 'INVALID_ARGUMENT',
  4: 'DEADLINE_EXCEEDED',
  5: 'NOT_FOUND',
  6: 'ALREADY_EXISTS',
  7: 'PERMISSION_DENIED',
  8: 'RESOURCE_EXHAUSTED',
  9: 'FAILED_PRECONDITION',
  10: 'ABORTED',
  11: 'OUT_OF_RANGE',
  12: 'UNIMPLEMENTED',
  13: 'INTERNAL',
  14: 'UNAVAILABLE',
  15: 'DATA_LOSS',
  16: 'UNAUTHENTICATED',
};

const depositRequests = new Counter('deposit_requests');
const depositOK = new Counter('deposit_ok');
const depositFailed = new Counter('deposit_failed');
const depositReplayed = new Counter('deposit_replayed');
const depositExceptions = new Counter('deposit_exceptions');
const poolExhausted = new Counter('deposit_account_pool_exhausted');
const depositOKDuration = new Trend('deposit_ok_duration', true);
const depositFailedDuration = new Trend('deposit_failed_duration', true);

const failByCode = {};
for (const code of Object.keys(CODE_NAMES)) {
  if (code !== '0') {
    failByCode[code] = new Counter(`deposit_fail_${code}`);
  }
}

const client = new grpc.Client();
if (!REFLECT) {
  client.load([PROTO_IMPORT_PATH], 'ledger/funds/v1/funds.proto');
}

let connected = false;

function connect() {
  if (REFLECT) {
    client.connect(TARGET, { plaintext: PLAINTEXT, timeout: '10s', reflect: true });
  } else {
    client.connect(TARGET, { plaintext: PLAINTEXT, timeout: '10s' });
  }
  connected = true;
}

function headerHas(headers, name, value) {
  if (!headers) return false;
  const raw = headers[name];
  if (raw === undefined) return false;
  if (Array.isArray(raw)) return raw.includes(value);
  return String(raw) === value;
}

// k6 v1 exposes response.status as a number; k6 v2 exposes a Status object
// that stringifies to the code name ("NotFound") and converts to the numeric
// code with Number(). Number() handles both versions.
function statusCode(response) {
  if (!response) return -1;
  const code = Number(response.status);
  return Number.isFinite(code) ? code : -1;
}

function isOK(response) {
  return statusCode(response) === 0;
}

export default function () {
  const seq = exec.scenario.iterationInTest; // 0-based, unique across the whole run

  if (seq >= ACCOUNT_COUNT) {
    // The range is exhausted. Refuse to reuse an account rather than silently
    // breaking the "one deposit per account" invariant.
    poolExhausted.add(1);
    return;
  }

  const accountId = ACCOUNT_MIN_ID + seq;
  const tags = {
    rate: String(MODE === 'ramp' ? 'ramp' : RATE),
    mode: MODE,
    run_id: RUN_ID,
  };

  if (!connected) {
    connect();
  }

  const started = Date.now();
  depositRequests.add(1, tags);
  let response;
  try {
    response = client.invoke(
      'ledger.funds.v1.FundsService/Deposit',
      {
        account_id: accountId,
        amount_minor: AMOUNT_MINOR,
        currency: CURRENCY,
      },
      {
        metadata: {
          // The server scopes this by account id, but a unique key per request
          // is what makes this an actual deposit instead of a replay.
          'x-idempotency-key': `lt-${RUN_ID}-${seq}`,
        },
        tags,
        timeout: RPC_TIMEOUT,
        discardResponseMessage: true,
      }
    );
  } catch (err) {
    // Transport-level failure (connection reset, DNS, TLS, ...). The client is
    // marked disconnected so the next iteration reconnects.
    connected = false;
    depositFailedDuration.add(Date.now() - started, tags);
    depositExceptions.add(1, tags);
    check(null, { 'deposit ok': () => false });
    return;
  }

  const elapsed = Date.now() - started;

  if (isOK(response)) {
    depositOK.add(1, tags);
    depositOKDuration.add(elapsed, tags);
    if (headerHas(response.headers, 'x-idempotent-replayed', 'true')) {
      depositReplayed.add(1, tags);
    }
  } else {
    const code = statusCode(response);
    depositFailed.add(1, { ...tags, code: String(code) });
    depositFailedDuration.add(elapsed, tags);
    const counter = failByCode[code];
    if (counter) {
      counter.add(1, tags);
    } else {
      depositExceptions.add(1, { ...tags, code: String(code) });
    }
  }

  check(response, { 'deposit ok': isOK });
}

function num(value, digits) {
  return Number(value).toFixed(digits);
}

function pct(part, total) {
  return total > 0 ? `${num((part / total) * 100, 2)}%` : 'n/a';
}

export function handleSummary(data) {
  const m = data.metrics;
  const reqs = (m.deposit_requests && m.deposit_requests.values) || {};
  const total = reqs.count || 0;
  const ok = (m.deposit_ok && m.deposit_ok.values.count) || 0;
  const failed = (m.deposit_failed && m.deposit_failed.values.count) || 0;
  const exceptions = (m.deposit_exceptions && m.deposit_exceptions.values.count) || 0;
  const replayed = (m.deposit_replayed && m.deposit_replayed.values.count) || 0;
  const exhausted = (m.deposit_account_pool_exhausted && m.deposit_account_pool_exhausted.values.count) || 0;
  const dropped = (m.dropped_iterations && m.dropped_iterations.values.count) || 0;
  const latency = (m.grpc_req_duration && m.grpc_req_duration.values) || {};

  const lines = [];
  lines.push('');
  lines.push('===== ledger deposit load test =====');
  lines.push(`mode         : ${MODE}`);
  lines.push(`target       : ${TARGET}`);
  lines.push(`run_id       : ${RUN_ID}`);
  lines.push(`rate         : ${MODE === 'ramp' ? 'ramp' : `${RATE}/s`}`);
  lines.push(`duration     : ${MODE === 'ramp' ? JSON.stringify(RAMP_STAGES) : DURATION}`);
  lines.push(`VUs          : ${VUS}`);
  lines.push('');
  lines.push(`RPCs         : ${total}${reqs.rate !== undefined ? `  (${num(reqs.rate, 1)}/s)` : ''}`);
  lines.push(`ok           : ${ok}  (${pct(ok, total)})`);
  lines.push(`non-OK       : ${failed}  (${pct(failed, total)})`);
  lines.push(`exceptions   : ${exceptions}  (${pct(exceptions, total)})`);
  for (const code of Object.keys(CODE_NAMES)) {
    if (code === '0') continue;
    const metric = m[`deposit_fail_${code}`];
    const count = metric && metric.values.count;
    if (count) {
      lines.push(`  code ${code} ${CODE_NAMES[code]}: ${count}`);
    }
  }
  lines.push(`replayed     : ${replayed}`);
  lines.push(`pool exhaust : ${exhausted}`);
  lines.push(`dropped iter : ${dropped}${dropped > 0 ? '  <-- k6 could not start every scheduled iteration' : ''}`);
  lines.push('');
  if (latency['p(50)'] !== undefined) {
    lines.push('latency (ms, grpc_req_duration):');
    lines.push(`  p50=${num(latency['p(50)'], 2)}  p90=${num(latency['p(90)'], 2)}  p95=${num(latency['p(95)'], 2)}  p99=${num(latency['p(99)'], 2)}  p99.9=${num(latency['p(99.9)'], 2)}  max=${num(latency.max, 2)}`);
  }
  const okLatency = (m.deposit_ok_duration && m.deposit_ok_duration.values) || {};
  if (okLatency['p(99)'] !== undefined) {
    lines.push(`  ok only: p50=${num(okLatency['p(50)'], 2)}  p99=${num(okLatency['p(99)'], 2)}  p99.9=${num(okLatency['p(99.9)'], 2)}`);
  }
  lines.push('');

  const out = {};
  out.stdout = `${lines.join('\n')}\n`;
  if (__ENV.OUT) {
    out[__ENV.OUT] = JSON.stringify(data, null, 2);
  }
  return out;
}
