# WebClient Reconnect E2E

`reconnect.e2e.mjs` runs the real `remote.html` module graph in Chrome while
replacing the signaling WebSocket and `RTCPeerConnection` with deterministic
test doubles. It does not require a Windows host, TURN server, or live
SignalingServer.

```bash
cd WebClient
npm install
npm run test:e2e:reconnect
```

The suite covers:

- recovery on reconnect attempts 1, 2, and 3;
- exhaustion after all four attempts fail;
- immediate stop after a non-retryable access-policy rejection;
- recording upload retry continuing while signaling reconnects;
- stale Session events being ignored after the replacement Session connects.

Set `QD_BROWSER_CHANNEL` to use a channel other than the locally installed
Google Chrome.
