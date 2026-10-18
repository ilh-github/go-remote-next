# Browser E2E Tests

These tests use Playwright with the locally installed Google Chrome. Install the
WebClient dependencies first:

```bash
cd WebClient
npm install
```

## Multi-Tab Coordination

`tab-coordinator.e2e.mjs` opens real same-origin Chrome tabs and verifies
BroadcastChannel arbitration for control and audio. It covers newer-owner
revocation, reclaim after release, audio-only revocation, and device isolation.

```bash
cd WebClient
npm run test:e2e:tab-coordinator
```

Set `QD_BROWSER_CHANNEL` to use a channel other than the locally installed
Google Chrome.
