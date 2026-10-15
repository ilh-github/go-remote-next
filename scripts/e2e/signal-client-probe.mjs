// Open a controller signaling socket and report what the server decided.
//
//   QD_SIGNAL_TOKEN=<token> QD_DEVICE=<device_id> [QD_HOLD_MS=8000] \
//     node signal-client-probe.mjs
//
// Exit codes: 0 = auth_ok (accepted), 3 = server sent an error frame
// (e.g. SESSION_LIMIT_REACHED), 4 = closed or timed out without auth_ok.
// QD_HOLD_MS keeps an accepted socket open, which is how a concurrency test
// occupies a slot while a second probe tries to get in.

const server = process.env.QD_SERVER ?? 'ws://127.0.0.1:8000';
const token = process.env.QD_SIGNAL_TOKEN;
const deviceId = process.env.QD_DEVICE;
const clientId = process.env.QD_CLIENT_ID ?? `probe-${process.pid}`;
const holdMs = Number(process.env.QD_HOLD_MS ?? 0);
const timeoutMs = Number(process.env.QD_TIMEOUT_MS ?? 10000);

if (!token || !deviceId) {
  console.error('QD_SIGNAL_TOKEN and QD_DEVICE are required');
  process.exit(2);
}

const url = `${server.replace(/\/$/, '')}/v1/realtime/signal`;
const socket = new WebSocket(url);
let settled = false;

const finish = (code) => {
  try {
    socket.close();
  } catch {
    // ignore
  }
  process.exit(code);
};

const timer = setTimeout(() => {
  if (!settled) {
    settled = true;
    console.error('timeout waiting for auth_ok');
    finish(4);
  }
}, timeoutMs);

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
  console.log(`< ${text}`);
  let payload;
  try {
    payload = JSON.parse(text);
  } catch {
    return;
  }
  if (payload.type === 'auth_ok') {
    console.log(`accepted: session=${payload.session_id}`);
    if (settled) {
      return;
    }
    settled = true;
    clearTimeout(timer);
    if (holdMs > 0) {
      console.log(`holding the connection for ${holdMs} ms`);
      setTimeout(() => finish(0), holdMs);
    } else {
      finish(0);
    }
    return;
  }
  if (payload.type === 'error') {
    settled = true;
    clearTimeout(timer);
    // The signaling WS puts the code at the top level (the events WS nests it
    // under `data`), so accept both shapes.
    const code = payload.code ?? payload.data?.code ?? 'unknown';
    const detail = payload.detail ?? payload.data?.detail ?? '';
    console.error(`rejected: ${code} ${detail}`);
    finish(3);
  }
});

socket.addEventListener('close', (event) => {
  if (!settled) {
    settled = true;
    clearTimeout(timer);
    console.error(`closed before auth_ok: code=${event.code} reason=${event.reason}`);
    finish(4);
  }
});

socket.addEventListener('error', (event) => {
  console.error(`socket error: ${event.message ?? event.type}`);
});
