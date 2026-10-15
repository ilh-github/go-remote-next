// Minimal webhook receiver for verifying user notification channels.
//
//   QD_LISTEN=9101 QD_SECRET=e2e-secret QD_EXPECT=2 node notification-receiver.mjs
//
// Prints one JSON line per delivery (with signature verification), and exits 0
// once QD_EXPECT deliveries arrived (exit 3 on timeout, exit 4 on a bad
// signature). QD_TIMEOUT_MS defaults to 30s.

import { createServer } from 'node:http';
import { createHmac, timingSafeEqual } from 'node:crypto';

const port = Number(process.env.QD_LISTEN ?? 9101);
const secret = process.env.QD_SECRET ?? '';
const expect = Number(process.env.QD_EXPECT ?? 1);
const timeoutMs = Number(process.env.QD_TIMEOUT_MS ?? 30000);

let received = 0;
let badSignature = false;

const server = createServer((req, res) => {
  if (req.method !== 'POST') {
    res.writeHead(405).end();
    return;
  }
  const chunks = [];
  req.on('data', (chunk) => chunks.push(chunk));
  req.on('end', () => {
    const raw = Buffer.concat(chunks);
    const header = req.headers['x-webhook-signature'] ?? '';
    let signatureOK = null;
    if (secret) {
      const expected = `sha256=${createHmac('sha256', secret).update(raw).digest('hex')}`;
      signatureOK =
        expected.length === header.length &&
        timingSafeEqual(Buffer.from(expected), Buffer.from(header));
      if (!signatureOK) {
        badSignature = true;
      }
    }
    received += 1;
    console.log(
      `[${received}] ${raw.toString('utf8')} signature=${signatureOK === null ? 'n/a' : signatureOK}`,
    );
    res.writeHead(200, { 'content-type': 'application/json' }).end('{"ok":true}');
    if (received >= expect) {
      finish(badSignature ? 4 : 0);
    }
  });
});

const finish = (code) => {
  server.close(() => process.exit(code));
  setTimeout(() => process.exit(code), 500).unref();
};

server.listen(port, '127.0.0.1', () => {
  console.log(`listening on http://127.0.0.1:${port} (expecting ${expect} delivery/ies)`);
});

setTimeout(() => {
  console.error(`timeout: received ${received}/${expect}`);
  finish(3);
}, timeoutMs);
