// Watch a user's realtime WebSocket for device.access.* events while an
// access attempt is triggered elsewhere.
//
//   QD_TOKEN=<access_token> QD_SERVER=ws://127.0.0.1:8000 node realtime-access-events.mjs
//
// Exits 0 only when at least one device.access.* event arrived before
// QD_WAIT_MS elapsed, so it can gate a verification run.

const server = process.env.QD_SERVER ?? 'ws://127.0.0.1:8000';
const token = process.env.QD_TOKEN;
const waitMs = Number(process.env.QD_WAIT_MS ?? 15000);

if (!token) {
  console.error('QD_TOKEN is required');
  process.exit(2);
}

const url = `${server.replace(/\/$/, '')}/v1/realtime/events`;
const socket = new WebSocket(url);
let matched = 0;
let seen = 0;

const finish = (code) => {
  try {
    socket.close();
  } catch {
    // ignore
  }
  process.exit(code);
};

socket.addEventListener('open', () => {
  console.log(`> connected ${url}`);
  socket.send(JSON.stringify({ type: 'auth', access_token: token }));
});

socket.addEventListener('message', (event) => {
  const text = typeof event.data === 'string' ? event.data : '';
  seen += 1;
  if (text.includes('device.access.')) {
    matched += 1;
    console.log(`< ${text}`);
  } else if (seen <= 3) {
    console.log(`< ${text.slice(0, 200)}`);
  }
});

socket.addEventListener('error', (event) => {
  console.error(`socket error: ${event.message ?? event.type}`);
});

setTimeout(() => {
  if (matched > 0) {
    console.log(`OK: saw ${matched} device.access event(s)`);
    finish(0);
  }
  console.error('NO device.access events seen');
  finish(3);
}, waitMs);
