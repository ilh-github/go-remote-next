// Concurrency / churn harness for the signaling server.
//
//   # one burst of 20 controllers, holding each session for 5s
//   QD_SIGNAL_TOKEN=<client_token> QD_CLIENTS=20 QD_HOLD_MS=5000 node signal-load.mjs
//
//   # churn: 5 rounds x 10 connect/auth/close cycles
//   QD_SIGNAL_TOKEN=<client_token> QD_ROUNDS=5 QD_CLIENTS=10 node signal-load.mjs
//
//   # assert a concurrency cap really admits exactly N controllers
//   QD_SIGNAL_TOKEN=<token> QD_CLIENTS=20 QD_EXPECT_ACCEPTED=5 QD_EXPECT_REJECTED=15 node signal-load.mjs
//
// A client signal token is reusable (the server validates-and-extends it so a
// blipped client can reconnect), so one `access-code:verify` is enough for the
// whole run - the harness never trips the per-IP verify rate limit.
//
// Exit codes: 0 = finished within expectations, 3 = counts did not match
// QD_EXPECT_ACCEPTED / QD_EXPECT_REJECTED, 4 = transport-level failures
// (socket error or timeout without a server verdict).

const server = process.env.QD_SERVER ?? 'ws://127.0.0.1:8000';
const token = process.env.QD_SIGNAL_TOKEN;
const deviceId = process.env.QD_DEVICE ?? '126735214';
const clients = Number(process.env.QD_CLIENTS ?? 10);
const rounds = Number(process.env.QD_ROUNDS ?? 1);
const holdMs = Number(process.env.QD_HOLD_MS ?? 0);
const timeoutMs = Number(process.env.QD_TIMEOUT_MS ?? 20000);
const expectAccepted = process.env.QD_EXPECT_ACCEPTED ? Number(process.env.QD_EXPECT_ACCEPTED) : null;
const expectRejected = process.env.QD_EXPECT_REJECTED ? Number(process.env.QD_EXPECT_REJECTED) : null;
const label = process.env.QD_LABEL ?? 'load';

if (!token) {
  console.error('QD_SIGNAL_TOKEN is required (mint one with access-code:verify)');
  process.exit(2);
}

const url = `${server.replace(/\/$/, '')}/v1/realtime/signal`;

// runClient opens one controller socket and resolves with the server's
// verdict: { verdict: 'accepted'|'rejected'|'error', code?, latencyMs, reason? }
function runClient(round, index) {
  return new Promise((resolve) => {
    const clientId = `${label}-${round}-${index}`;
    const started = Date.now();
    const socket = new WebSocket(url);
    let settled = false;

    // authLatencyMs is measured when the *server verdict* arrives, so a long
    // hold does not inflate the connect latency.
    const done = (result, authLatencyMs = Date.now() - started) => {
      if (settled) {
        return;
      }
      settled = true;
      clearTimeout(timer);
      try {
        socket.close();
      } catch {
        // ignore
      }
      resolve({ ...result, latencyMs: authLatencyMs });
    };

    const timer = setTimeout(() => done({ verdict: 'error', reason: 'timeout' }), timeoutMs);

    socket.addEventListener('open', () => {
      socket.send(JSON.stringify({
        type: 'auth',
        signal_token: token,
        device_id: deviceId,
        role: 'client',
        client_id: clientId,
      }));
    });

    socket.addEventListener('message', (event) => {
      const text = typeof event.data === 'string' ? event.data : '';
      let payload;
      try {
        payload = JSON.parse(text);
      } catch {
        return;
      }
      if (payload.type === 'auth_ok') {
        const authLatency = Date.now() - started;
        if (holdMs > 0) {
          setTimeout(() => done({ verdict: 'accepted' }, authLatency), holdMs);
        } else {
          done({ verdict: 'accepted' }, authLatency);
        }
        return;
      }
      if (payload.type === 'error') {
        done({
          verdict: 'rejected',
          code: payload.code ?? payload.data?.code ?? 'UNKNOWN',
          reason: payload.detail ?? payload.data?.detail ?? '',
        }, Date.now() - started);
      }
    });

    socket.addEventListener('close', () => done({ verdict: 'error', reason: 'closed before verdict' }));
    socket.addEventListener('error', (event) =>
      done({ verdict: 'error', reason: `socket error: ${event.message ?? event.type}` }));
  });
}

const percentile = (sorted, p) => {
  if (sorted.length === 0) {
    return 0;
  }
  const idx = Math.min(sorted.length - 1, Math.max(0, Math.ceil((p / 100) * sorted.length) - 1));
  return sorted[idx];
};

const totals = { accepted: 0, rejected: 0, error: 0 };
const codes = new Map();
const errorReasons = [];
const latencies = [];
const runStarted = Date.now();

for (let round = 0; round < rounds; round += 1) {
  const roundStarted = Date.now();
  const results = await Promise.all(
    Array.from({ length: clients }, (_, index) => runClient(round, index)),
  );
  for (const result of results) {
    totals[result.verdict] += 1;
    latencies.push(result.latencyMs);
    if (result.verdict === 'rejected') {
      codes.set(result.code, (codes.get(result.code) ?? 0) + 1);
    }
    if (result.verdict === 'error') {
      errorReasons.push(result.reason);
    }
  }
  const accepted = results.filter((r) => r.verdict === 'accepted').length;
  const rejected = results.filter((r) => r.verdict === 'rejected').length;
  console.log(
    `round ${round + 1}/${rounds}: accepted=${accepted} rejected=${rejected} ` +
      `errors=${results.length - accepted - rejected} in ${Date.now() - roundStarted} ms`,
  );
}

latencies.sort((a, b) => a - b);
const wallMs = Date.now() - runStarted;
console.log(
  `SUMMARY clients=${clients} rounds=${rounds} hold=${holdMs}ms ` +
    `accepted=${totals.accepted} rejected=${totals.rejected} errors=${totals.error} ` +
    `latency_ms p50=${percentile(latencies, 50)} p95=${percentile(latencies, 95)} ` +
    `max=${latencies[latencies.length - 1] ?? 0} wall=${wallMs}ms`,
);
if (codes.size > 0) {
  console.log(`REJECT_CODES ${[...codes.entries()].map(([code, n]) => `${code}=${n}`).join(' ')}`);
}
if (errorReasons.length > 0) {
  console.log(`ERROR_REASONS ${JSON.stringify(errorReasons.slice(0, 5))}`);
}

if (expectAccepted !== null && totals.accepted !== expectAccepted) {
  console.error(`expected ${expectAccepted} accepted, saw ${totals.accepted}`);
  process.exit(3);
}
if (expectRejected !== null && totals.rejected !== expectRejected) {
  console.error(`expected ${expectRejected} rejected, saw ${totals.rejected}`);
  process.exit(3);
}
process.exit(totals.error > 0 ? 4 : 0);
