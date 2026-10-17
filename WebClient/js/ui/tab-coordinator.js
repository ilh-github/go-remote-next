/**
 * Coordinates exclusive resources (control input and audio playback) between
 * same-origin tabs that are connected to the same device.
 *
 * BroadcastChannel is the primary transport. A localStorage event is used as a
 * fallback for browsers that do not expose BroadcastChannel.
 */
export class TabCoordinator extends EventTarget {
    constructor({
        deviceId,
        tabId,
        channelName = 'quickdesk:tab-bus',
        now = () => Date.now(),
        BroadcastChannelImpl = globalThis.BroadcastChannel,
        storage = null,
        storageEventTarget = null,
    } = {}) {
        super();
        this.deviceId = deviceId || '';
        this.tabId = tabId || `tab-${Math.random().toString(36).slice(2)}`;
        this.channelName = channelName;
        this._now = now;
        this._BroadcastChannelImpl = BroadcastChannelImpl;
        this._storage = storage;
        this._storageEventTarget = storageEventTarget;
        this._channel = null;
        this._started = false;
        this._claims = new Map();
        this._onStorage = this._onStorage.bind(this);
    }

    start() {
        if (this._started) return;
        this._started = true;

        if (this._BroadcastChannelImpl) {
            try {
                this._channel = new this._BroadcastChannelImpl(this.channelName);
                this._channel.onmessage = (event) => this._onMessage(event.data);
                return;
            } catch {
                this._channel = null;
            }
        }

        if (!this._storage) {
            try {
                this._storage = globalThis.localStorage;
            } catch {
                this._storage = null;
            }
        }
        if (!this._storageEventTarget) {
            this._storageEventTarget = globalThis.window;
        }
        this._storageEventTarget?.addEventListener?.('storage', this._onStorage);
    }

    claim(resource) {
        if (!resource) return;
        const at = this._now();
        this._claims.set(resource, at);
        this._post({ type: 'claim', resource, at });
    }

    release(resource) {
        if (!resource) return;
        if (!this._claims.delete(resource)) return;
        this._post({ type: 'release', resource });
    }

    releaseAll() {
        for (const resource of [...this._claims.keys()]) {
            this.release(resource);
        }
    }

    hasClaim(resource) {
        return this._claims.has(resource);
    }

    destroy() {
        if (!this._started) return;
        this._started = false;
        this._claims.clear();

        if (this._channel) {
            this._channel.close();
            this._channel = null;
            return;
        }
        this._storageEventTarget?.removeEventListener?.('storage', this._onStorage);
    }

    _post(message) {
        if (!this.deviceId || !this.tabId) return;
        const envelope = {
            ...message,
            deviceId: this.deviceId,
            tabId: this.tabId,
            nonce: Math.random().toString(36).slice(2),
        };

        if (this._channel) {
            this._channel.postMessage(envelope);
            return;
        }

        try {
            this._storage?.setItem(this.channelName, JSON.stringify(envelope));
        } catch {
            // Private mode or a full quota must not break remote control.
        }
    }

    _onStorage(event) {
        if (!event || event.key !== this.channelName || !event.newValue) return;
        try {
            this._onMessage(JSON.parse(event.newValue));
        } catch {
            // Ignore malformed messages from another tab.
        }
    }

    _onMessage(message) {
        if (!message || message.deviceId !== this.deviceId) return;
        if (!message.tabId || message.tabId === this.tabId) return;

        if (message.type === 'release') {
            return;
        }
        if (message.type !== 'claim' || !message.resource) {
            return;
        }

        const localAt = this._claims.get(message.resource);
        if (localAt === undefined) {
            return;
        }

        const incomingAt = Number(message.at);
        const incomingWins =
            incomingAt > localAt ||
            (incomingAt === localAt && String(message.tabId) > String(this.tabId));
        if (!incomingWins) {
            return;
        }

        this._claims.delete(message.resource);
        this.dispatchEvent(new CustomEvent('revoked', {
            detail: {
                resource: message.resource,
                ownerTabId: message.tabId,
            },
        }));
    }
}
