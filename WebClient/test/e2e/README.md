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

## Multi-Tab Coordination

`tab-coordinator.e2e.mjs` opens real same-origin Chrome tabs and verifies
BroadcastChannel arbitration for control and audio. It covers newer-owner
revocation, reclaim after release, audio-only revocation, and device isolation.

```bash
cd WebClient
npm run test:e2e:tab-coordinator
```

## Screen Wall

`wall.e2e.mjs` loads the real `WallPage.vue` through Vite, mocks the owner-scoped
device APIs, and replaces each embedded session with a same-origin iframe that
reports host state. It covers live tiles, exclusive audio and input, focus,
reconnect, columns, and removal.

```bash
cd WebClient
npm run test:e2e:wall
```

## Devices

`devices.e2e.mjs` loads the real `DevicesPage.vue` through Vite and drives the
owner-scoped device, favorite, connection-log, and private-group APIs. It covers
group filtering and management, remarks, access-code checks, connect handoff,
refresh, favorite removal, and unbind.

```bash
cd WebClient
npm run test:e2e:devices
```

Set `QD_BROWSER_CHANNEL` to use a channel other than the locally installed
Google Chrome.
