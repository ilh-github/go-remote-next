/**
 * Tracks whether this client may drive the remote machine.
 *
 * The signaling server is the one authority on who holds control, because the
 * input itself travels peer to peer over the WebRTC DataChannel and nobody in
 * the middle can drop a byte. So the server says who holds it, and this object
 * is what the rest of the client reads before it sends anything.
 *
 * The starting state is "may drive": that is what every device already does,
 * and a client that began by refusing input would break control on every
 * machine whose owner never opened the setting.
 */
export class ControlGrantState extends EventTarget {
    constructor({ mode = 'shared', holder = '' } = {}) {
        super();
        this._mode = mode;
        this._holder = holder;
        this._canControl = true;
        // The holder a revoke was already reported for, so a replayed frame does
        // not put a second notice on screen.
        this._revokedFrom = '';
    }

    get mode() { return this._mode; }
    get holder() { return this._holder; }
    get canControl() { return this._canControl; }

    /**
     * @param {object} frame  a `control_grant` frame from the signaling server
     */
    applyGrant(frame) {
        if (!frame || typeof frame !== 'object') return;
        // Only an explicit yes/no is an answer. A frame without `can_control`
        // says nothing this client can act on, and guessing would either freeze
        // its input or hand it control it does not have.
        if (typeof frame.can_control !== 'boolean') return;
        const mode = frame.control_mode;
        if (mode !== 'shared' && mode !== 'exclusive') return;

        this._mode = mode;
        this._holder = mode === 'exclusive' ? String(frame.control_holder || '') : '';
        this._canControl = frame.can_control;
        // Regaining control retires the outstanding notice, so the next revoke
        // is reported again rather than being swallowed as a duplicate.
        if (this._canControl) this._revokedFrom = '';
    }

    /**
     * @param {object} frame  a `control_revoked` frame from the signaling server
     */
    applyRevoked(frame) {
        if (!frame || typeof frame !== 'object') return;
        if (frame.control_mode !== 'exclusive') return;
        const holder = String(frame.control_holder || '');
        if (holder && holder === this._revokedFrom) return;

        this._canControl = false;
        this._revokedFrom = holder;
        this.dispatchEvent(new CustomEvent('revoked', { detail: { holder } }));
    }

    /**
     * Routes one raw signaling frame. Anything that is not a control frame is
     * ignored, so this can sit on the same socket as the jingle traffic.
     * @param {string|object} raw
     */
    handleMessage(raw) {
        let frame = raw;
        if (typeof raw === 'string') {
            try { frame = JSON.parse(raw); }
            catch { return; }
        }
        if (!frame || typeof frame !== 'object') return;

        if (frame.type === 'control_grant') {
            this.applyGrant(frame);
            return;
        }
        if (frame.type === 'control_revoked') {
            this.applyRevoked(frame);
        }
    }
}
