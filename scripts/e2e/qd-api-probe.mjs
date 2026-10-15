// Probe the QuickDesk local WebSocket API (default ws://127.0.0.1:9600).
//
//   node qd-api-probe.mjs listConnections
//   node qd-api-probe.mjs skillListTools '{"deviceId":"126735214"}'
//   node qd-api-probe.mjs skillExec '{"deviceId":"126735214","tool":"list_directory","args":{"path":"C:\\QuickDeskFilesTest"}}'
//
// Env: QD_API_URL (default ws://127.0.0.1:9600), QD_API_TIMEOUT_MS.

const url = process.env.QD_API_URL ?? 'ws://127.0.0.1:9600';
const method = process.argv[2];
const paramsArg = process.argv[3] ?? '{}';
const timeoutMs = Number(process.env.QD_API_TIMEOUT_MS ?? 20_000);

if (!method) {
  console.error('usage: qd-api-probe.mjs <method> [jsonParams]');
  process.exit(2);
}

let params;
try {
  params = JSON.parse(paramsArg);
} catch (error) {
  console.error(`bad params JSON: ${error.message}`);
  process.exit(2);
}

const started = Date.now();
const socket = new WebSocket(url);

const finish = (code) => {
  try {
    socket.close();
  } catch {
    // ignore
  }
  process.exit(code);
};

const timer = setTimeout(() => {
  console.error(`timeout after ${timeoutMs} ms`);
  finish(3);
}, timeoutMs);

socket.addEventListener('open', () => {
  console.log(`> ${method} ${JSON.stringify(params)}`);
  socket.send(JSON.stringify({ id: 1, method, params }));
});

socket.addEventListener('message', (event) => {
  clearTimeout(timer);
  const elapsed = Date.now() - started;
  console.log(`< ${elapsed} ms ${event.data}`);
  finish(0);
});

socket.addEventListener('error', (event) => {
  clearTimeout(timer);
  console.error(`socket error: ${event.message ?? event.type}`);
  finish(1);
});

socket.addEventListener('close', (event) => {
  clearTimeout(timer);
  console.error(`socket closed: code=${event.code} reason=${event.reason}`);
  finish(1);
});
