// k6 load test for ledger AccountService.
//
// One iteration = one CreateAccountInfo RPC followed by OpenLedgerAccount.
// Account ids start at ACCOUNT_MIN_ID (default 1) and increase by one for
// every iteration of the run. The scenario runs for DURATION (default 1m)
// and keeps creating accounts until the deadline.
//
// Account creation is idempotent: the account id is also the idempotency key.
// Re-running the test for the same id is safe.
//
// Common env:
//   DURATION            deadline (default 1m)
//   VUS                 concurrent VUs (default 50)
//   ACCOUNT_MIN_ID      first account id (default 1)
//   USER_ID             account owner (default 228)
//   ACCOUNT_TYPE        account type code (default 2100, client deposit)
//   TARGET              gRPC address (default 127.0.0.1:50051)
//   REFLECT             use server reflection (default true)
//   PROTO_IMPORT_PATH   proto import root when REFLECT=false (default ../proto,
//                       relative to this script)
//   CURRENCY            account currency (default USD)
//   RUN_ID              unique run id; change it to force fresh attempts
//   OUT                 path for the raw JSON summary
//   MAX_FAIL_RATE       threshold, default 0.01
//   P99_MS              threshold, default 1000
//   RPC_TIMEOUT         per-RPC timeout, default 10s

import grpc from 'k6/net/grpc';
import { check } from 'k6';
import { Counter, Trend } from 'k6/metrics';
import exec from 'k6/execution';

const TARGET = __ENV.TARGET || '127.0.0.1:50051';
const PLAINTEXT = (__ENV.PLAINTEXT || 'true') === 'true';
const RPC_TIMEOUT = __ENV.RPC_TIMEOUT || '10s';
const REFLECT = (__ENV.REFLECT || 'true') === 'true';
const PROTO_IMPORT_PATH = __ENV.PROTO_IMPORT_PATH || '../proto';
const CURRENCY = __ENV.CURRENCY || 'USD';
const USER_ID = Number(__ENV.USER_ID || 228);
const ACCOUNT_TYPE = Number(__ENV.ACCOUNT_TYPE || 2100);
const RUN_ID = __ENV.RUN_ID || 'adhoc';

const DURATION = __ENV.DURATION || '1m';
const VUS = Number(__ENV.VUS || 50);
const ACCOUNT_MIN_ID = Number(__ENV.ACCOUNT_MIN_ID || 1);

const MAX_FAIL_RATE = Number(__ENV.MAX_FAIL_RATE || 0.01);
const P99_MS = Number(__ENV.P99_MS || 1000);

if (!Number.isFinite(ACCOUNT_MIN_ID) || ACCOUNT_MIN_ID <= 0) {
  throw new Error('ACCOUNT_MIN_ID must be a positive number');
}
if (!Number.isFinite(VUS) || VUS <= 0) {
  throw new Error('VUS must be a positive number');
}

export const options = {
  scenarios: {
    accounts: {
      executor: 'constant-vus',
      vus: VUS,
      duration: DURATION,
      gracefulStop: '30s',
      tags: { mode: 'deadline', run_id: RUN_ID },
    },
  },
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

const accountRequests = new Counter('account_requests');
const accountOK = new Counter('account_ok');
const accountExists = new Counter('account_exists');
const accountFailed = new Counter('account_failed');
const accountExceptions = new Counter('account_exceptions');
const accountOKDuration = new Trend('account_ok_duration', true);
const accountFailedDuration = new Trend('account_failed_duration', true);

const failByCode = {};
for (const code of Object.keys(CODE_NAMES)) {
  if (code !== '0') {
    failByCode[code] = new Counter(`account_fail_${code}`);
  }
}

const client = new grpc.Client();
if (!REFLECT) {
  client.load([PROTO_IMPORT_PATH], 'ledger/accounts/v1/accounts.proto');
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
  const accountId = ACCOUNT_MIN_ID + seq;

  const tags = {
    mode: 'deadline',
    run_id: RUN_ID,
  };

  if (!connected) {
    connect();
  }

  const started = Date.now();
  accountRequests.add(1, tags);
  const accountID = String(accountId);
  let response;
  try {
    response = client.invoke(
      'ledger.accounts.v1.AccountService/CreateAccountInfo',
      {
        id: accountID,
        user_id: USER_ID,
        name: `deposit-seed-${accountId}`,
        description: 'loadtest account seed',
        type: ACCOUNT_TYPE,
        currency: CURRENCY,
      },
      {
        tags,
        timeout: RPC_TIMEOUT,
        discardResponseMessage: true,
      }
    );

    if (isOK(response)) {
      response = client.invoke(
        'ledger.accounts.v1.AccountService/OpenLedgerAccount',
        { id: accountID },
        {
          tags,
          timeout: RPC_TIMEOUT,
          discardResponseMessage: true,
        }
      );
    }
  } catch (err) {
    connected = false;
    accountFailedDuration.add(Date.now() - started, tags);
    accountExceptions.add(1, tags);
    check(null, { 'account ok or exists': () => false });
    return;
  }

  const elapsed = Date.now() - started;
  const code = statusCode(response);

  if (isOK(response)) {
    accountOK.add(1, tags);
    accountOKDuration.add(elapsed, tags);
  } else if (code === 6) {
    // ALREADY_EXISTS: keep going, an existing account is fine.
    accountExists.add(1, tags);
    accountOKDuration.add(elapsed, tags);
  } else {
    accountFailed.add(1, { ...tags, code: String(code) });
    accountFailedDuration.add(elapsed, tags);
    const counter = failByCode[code];
    if (counter) {
      counter.add(1, tags);
    } else {
      accountExceptions.add(1, { ...tags, code: String(code) });
    }
  }

  check(response, {
    'account ok or exists': (r) => {
      const c = statusCode(r);
      return c === 0 || c === 6;
    },
  });
}

function num(value, digits) {
  return Number(value).toFixed(digits);
}

function pct(part, total) {
  return total > 0 ? `${num((part / total) * 100, 2)}%` : 'n/a';
}

export function handleSummary(data) {
  const m = data.metrics;
  const reqs = (m.account_requests && m.account_requests.values) || {};
  const total = reqs.count || 0;
  const ok = (m.account_ok && m.account_ok.values.count) || 0;
  const exists = (m.account_exists && m.account_exists.values.count) || 0;
  const failed = (m.account_failed && m.account_failed.values.count) || 0;
  const exceptions = (m.account_exceptions && m.account_exceptions.values.count) || 0;
  const dropped = (m.dropped_iterations && m.dropped_iterations.values.count) || 0;
  const latency = (m.grpc_req_duration && m.grpc_req_duration.values) || {};

  const lines = [];
  lines.push('');
  lines.push('===== ledger account seed load test =====');
  lines.push(`target       : ${TARGET}`);
  lines.push(`run_id       : ${RUN_ID}`);
  lines.push(`duration     : ${DURATION}`);
  lines.push(`VUs          : ${VUS}`);
  lines.push(`id range     : ${ACCOUNT_MIN_ID}..${ACCOUNT_MIN_ID + total}`);
  lines.push('');
  lines.push(`RPCs         : ${total}${reqs.rate !== undefined ? `  (${num(reqs.rate, 1)}/s)` : ''}`);
  lines.push(`ok           : ${ok}  (${pct(ok, total)})`);
  lines.push(`exists       : ${exists}  (${pct(exists, total)})`);
  lines.push(`non-OK       : ${failed}  (${pct(failed, total)})`);
  lines.push(`exceptions   : ${exceptions}  (${pct(exceptions, total)})`);
  for (const code of Object.keys(CODE_NAMES)) {
    if (code === '0' || code === '6') continue;
    const metric = m[`account_fail_${code}`];
    const count = metric && metric.values.count;
    if (count) {
      lines.push(`  code ${code} ${CODE_NAMES[code]}: ${count}`);
    }
  }
  lines.push(`dropped iter : ${dropped}`);
  lines.push('');
  if (latency['p(50)'] !== undefined) {
    lines.push('latency (ms, grpc_req_duration):');
    lines.push(`  p50=${num(latency['p(50)'], 2)}  p90=${num(latency['p(90)'], 2)}  p95=${num(latency['p(95)'], 2)}  p99=${num(latency['p(99)'], 2)}  p99.9=${num(latency['p(99.9)'], 2)}  max=${num(latency.max, 2)}`);
  }
  lines.push('');

  const out = {};
  out.stdout = `${lines.join('\n')}\n`;
  if (__ENV.OUT) {
    out[__ENV.OUT] = JSON.stringify(data, null, 2);
  }
  return out;
}
