/**
 * websocket-transport.js - 信令 WebSocket 传输层
 *
 * 协议（§2.13 / §2.6）：
 *   1. 建立 WS 到 /v1/realtime/signal（URL 不带 token/code）
 *   2. 客户端首帧发送 {type:"auth", signal_token, role:"client", device_id, client_id}
 *   3. 服务端回 {type:"auth_ok"} 后才允许发送 SDP/ICE jingle XML
 *   4. auth_ok 5s 超时则服务端会主动 close(4401)
 *   5. signal_token 一次性，WS auth_ok 后服务端立即 DEL
 *
 * 旧版 URL 里带 access_code 的做法已废弃（第 1.3 节缺陷 #1/#3）。
 */

const DEFAULT_AUTH_TIMEOUT_MS = 6000;
// How long a link may stay silent before it is treated as dead.
//
// The server pings every 30s, so a healthy connection is never quiet for long.
// The default leaves three ping intervals of room, which is loose enough that a
// slow or lossy link is not killed for being slow - the failure this catches is
// a link that stopped moving entirely, and only that.
//
// Why this exists at all: a socket that goes OPEN and then stops delivering is
// indistinguishable from a healthy one by any property the browser exposes.
// readyState stays OPEN, no error fires, no close fires. That is the shape of a
// proxy or VPN switch, where the route moved underneath a live connection, and
// without this the client reports "Connected" on a link that carries nothing.
const DEFAULT_STALENESS_TIMEOUT_MS = 90000;

export class WebSocketTransport {
    /**
     * @param {object} options
     * @param {string} options.signalingUrl  - 信令服务器基础 URL
     * @param {Function} options.onMessage   - 业务消息回调 (message: string)
     * @param {Function} options.onOpen      - WS 连接成功（尚未 auth_ok）
     * @param {Function} options.onAuthOk    - 收到 auth_ok，可以发 SDP
     * @param {Function} options.onClose     - (code, reason)
     * @param {Function} options.onError     - (error)
     * @param {number} [options.authTimeoutMs=6000] - auth_ok 超时，测试可缩短
     * @param {number} [options.stalenessTimeoutMs=90000] - 下行静默多久算链路已死
     */
    constructor(options) {
        this.signalingUrl = options.signalingUrl || 'ws://localhost:8000';
        this.onMessage = options.onMessage || (() => {});
        this.onOpen    = options.onOpen    || (() => {});
        this.onAuthOk  = options.onAuthOk  || (() => {});
        this.onClose   = options.onClose   || (() => {});
        this.onError   = options.onError   || (() => {});
        this.authTimeoutMs = Number.isFinite(options.authTimeoutMs) && options.authTimeoutMs >= 0
            ? options.authTimeoutMs
            : DEFAULT_AUTH_TIMEOUT_MS;
        this.stalenessTimeoutMs = Number.isFinite(options.stalenessTimeoutMs) && options.stalenessTimeoutMs > 0
            ? options.stalenessTimeoutMs
            : DEFAULT_STALENESS_TIMEOUT_MS;

        this.ws = null;
        this._authOk = false;
        this._closed = false;
        this._authTimer = null;
        this._stalenessTimer = null;
        this._authRejection = null;

        // Auth params for the first-frame handshake.
        this._deviceId = null;
        this._signalToken = null;
        this._clientId = null;
        this._role = 'client';
    }

    /**
     * @param {object} params
     * @param {string} params.deviceId
     * @param {string} params.signalToken  一次性 signal_token
     * @param {string} params.clientId     client 角色必填
     * @param {string} [params.role]       默认 'client'
     */
    connect(params) {
        this._deviceId    = params.deviceId;
        this._signalToken = params.signalToken;
        this._clientId    = params.clientId || '';
        this._role        = params.role || 'client';
        this._closed = false;
        this._authOk = false;
        this._authRejection = null;

        return this._doConnect();
    }

    _doConnect() {
        return new Promise((resolve, reject) => {
            let base = this.signalingUrl.replace(/\/+$/, '');
            // §2.13: URL carries NO token.
            const wsUrl = `${base}/v1/realtime/signal`;
            console.log(`[WebSocket] Connecting to: ${wsUrl}`);

            try {
                this.ws = new WebSocket(wsUrl);
            } catch (e) {
                reject(new Error(`Failed to create WebSocket: ${e.message}`));
                return;
            }

            this._resolve = resolve;
            this._reject  = reject;

            this.ws.onopen = () => {
                console.log('[WebSocket] Opened, sending auth frame');
                this.onOpen();
                this._sendAuthFrame();
                // Abandon if auth_ok doesn't arrive in time.
                this._authTimer = setTimeout(() => {
                    if (!this._authOk && !this._closed) {
                        console.error('[WebSocket] auth_ok timeout');
                        this._fail(new Error('Auth timeout'));
                    }
                }, this.authTimeoutMs);
            };

            this.ws.onerror = (event) => {
                console.error('[WebSocket] Error:', event);
                this.onError(event);
                // onclose will follow and handle cleanup + rejection.
            };

            this.ws.onmessage = (event) => this._handleMessage(event.data);

            this.ws.onclose = (event) => {
                const code = event && event.code;
                const reason = event && event.reason;
                console.log(`[WebSocket] Closed: code=${code}, reason=${reason}`);
                clearTimeout(this._authTimer); this._authTimer = null;
                this._disarmStalenessWatch();
                this.onClose(code, reason);
                if (!this._authOk && this._reject) {
                    // Closed before auth succeeded — treat as connect failure.
                    const rejection = this._authRejection;
                    const error = rejection
                        ? new Error(rejection.message)
                        : new Error(`WS closed before auth (code=${code})`);
                    if (rejection) {
                        // The server said why. A retry would ask the same
                        // question and get the same answer.
                        error.code = rejection.code;
                        error.retryable = false;
                    }
                    this._reject(error);
                    this._reject = this._resolve = null;
                }
                this.ws = null;
            };
        });
    }

    // A link that has gone quiet is closed from the client side, which is what
    // turns an invisible failure into one the reconnect path can act on. Closing
    // rather than reporting is deliberate: the session's close handler already
    // knows how to retry, so this reuses that machinery instead of adding a
    // second recovery path.
    _armStalenessWatch() {
        if (!this.stalenessTimeoutMs) return;
        clearTimeout(this._stalenessTimer);
        this._stalenessTimer = setTimeout(() => {
            this._stalenessTimer = null;
            if (!this._authOk || this._closed || !this.ws) return;
            console.warn(`[WebSocket] no downstream traffic for ${this.stalenessTimeoutMs}ms; treating the link as dead`);
            // An abnormal code, so the server side of this is not mistaken for a
            // clean shutdown the operator asked for.
            try { this.ws.close(4000, 'signaling link went silent'); }
            catch { /* already closing; onclose will still run */ }
        }, this.stalenessTimeoutMs);
    }

    _disarmStalenessWatch() {
        clearTimeout(this._stalenessTimer);
        this._stalenessTimer = null;
    }

    _sendAuthFrame() {
        const frame = {
            type: 'auth',
            signal_token: this._signalToken,
            role: this._role,
            device_id: this._deviceId,
        };
        if (this._clientId) frame.client_id = this._clientId;
        try { this.ws.send(JSON.stringify(frame)); }
        catch (e) { this._fail(e); }
    }

    _handleMessage(data) {
        // Any traffic at all proves the link is alive, so the watch is re-armed
        // here rather than only on auth_ok: a busy session keeps it alive frame
        // by frame without the timer ever firing.
        if (this._authOk) this._armStalenessWatch();
        const message = typeof data === 'string' ? data : data.toString();
        const trimmed = message.trim();

        // Prefer JSON control frames. Jingle XML is routed straight through.
        if (trimmed.startsWith('{')) {
            let json;
            try { json = JSON.parse(trimmed); }
            catch { /* not JSON; fall through to XML */ }
            if (json && typeof json === 'object') {
                if (json.type === 'auth_ok') {
                    this._authOk = true;
                    clearTimeout(this._authTimer); this._authTimer = null;
                    this._armStalenessWatch();
                    console.log('[WebSocket] auth_ok');
                    this.onAuthOk(json);
                    if (this._resolve) { this._resolve(); this._resolve = this._reject = null; }
                    return;
                }
                if (json.type === 'error') {
                    const code = json.code || (json.data && json.data.code) || '';
                    console.warn(`[WebSocket] server error code=${code}`);
                    // A rejected credential is not a bad link. Retrying it
                    // spends the whole backoff budget rediscovering a
                    // rejection the server has already stated, so the reason
                    // has to travel with the failure.
                    if (code === 'TOKEN_INVALID' || code === 'AUTH_INVALID') {
                        this._authRejection = { code, message: json.message || code };
                    }
                    // Surface to session as a message for state-machine logic
                    // (PEER_DISCONNECTED / HOST_OFFLINE etc. — §2.15).
                    this.onMessage(trimmed);
                    return;
                }
                // JSON envelope carrying Jingle XML: {payload:"<iq...>", client_id:"..."}
                // Unwrap and pass the XML payload to the session layer.
                if (json.payload && typeof json.payload === 'string') {
                    if (!this._authOk) {
                        console.warn('[WebSocket] Dropping pre-auth payload');
                        return;
                    }
                    this.onMessage(json.payload);
                    return;
                }
                // Other JSON control frames (session.revoked etc.).
                this.onMessage(trimmed);
                return;
            }
        }
        // XML (Jingle) — ignore if auth hasn't completed (server shouldn't
        // send these but guard anyway per §2.13).
        if (!this._authOk) {
            console.warn('[WebSocket] Dropping pre-auth payload');
            return;
        }
        this.onMessage(trimmed);
    }

    /**
     * Send a payload. Refuses to send SDP/ICE before auth_ok (§2.13).
     * Wraps XML (Jingle) messages in the JSON envelope {payload, client_id}
     * expected by the v1 signaling server (Chromium host parses this format).
     */
    send(message) {
        if (!this.ws || this.ws.readyState !== WebSocket.OPEN) {
            console.error('[WebSocket] Not connected, cannot send message');
            return false;
        }
        if (!this._authOk) {
            console.error('[WebSocket] auth not complete, refusing to send');
            return false;
        }
        // Wrap in JSON envelope matching Chromium client's SendJingleEnvelope format.
        const envelope = JSON.stringify({
            client_id: this._clientId || '',
            payload: message
        });
        this.ws.send(envelope);
        return true;
    }

    isConnected() { return !!(this.ws && this.ws.readyState === WebSocket.OPEN && this._authOk); }

    _fail(err) {
        this._closed = true;
        clearTimeout(this._authTimer); this._authTimer = null;
        const reject = this._reject;
        this._reject = this._resolve = null;
        if (reject) reject(err);
        try { this.ws && this.ws.close(); } catch { /* noop */ }
    }

    disconnect() {
        this._closed = true;
        clearTimeout(this._authTimer); this._authTimer = null;
        if (this.ws) {
            this.ws.onclose = null;
            try { this.ws.close(); } catch { /* noop */ }
            this.ws = null;
        }
    }
}
