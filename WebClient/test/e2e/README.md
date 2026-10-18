# WebClient Browser E2E

## Reconnect

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
- recording upload retry continuing while signaling reconnects, when that
  fork-specific upload module is present;
- stale Session events being ignored after the replacement Session connects.

## File Manager

`file-manager.e2e.mjs` loads the real `RemoteFileManager` and
`FileManagerPanel` modules in Chrome against an in-page fake host. It covers
directory navigation, create, rename, truncated text preview, download, chunked
upload, and trash.

```bash
cd WebClient
npm run test:e2e:file-manager
```

Set `QD_BROWSER_CHANNEL` to use a channel other than the locally installed
Google Chrome.
