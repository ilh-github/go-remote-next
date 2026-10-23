// Shares annotation strokes between viewers of the same device.
//
// BroadcastChannel is the transport for the same reason tab coordination uses it
// (QD-313): every viewer of a device is a same-origin page, and the channel
// reaches all of them without a server round trip. A stroke is a few dozen
// numbers; routing it through the control plane would put a row in a durable
// outbox for every pointer move, which is the wrong shape for data that is only
// meaningful while somebody is looking at the screen right now.
//
// The messages are deliberately additive: a peer tells the others "here is a
// stroke" or "here is a stroke to drop". Nobody's model is the authority, and
// nobody's clear wipes anybody else's work - which is why clearing is scoped by
// author rather than being a broadcast "empty yourself".

const CHANNEL_PREFIX = 'quickdesk:annotations:'

export const ANNOTATION_MSG_ADD = 'add'
export const ANNOTATION_MSG_REMOVE = 'remove'
export const ANNOTATION_MSG_CLEAR_AUTHOR = 'clear-author'

/**
 * @param {object} options
 * @param {string} options.deviceId   scopes the channel to one device
 * @param {string} options.viewerId   this viewer's identity, for clear-mine
 * @param {Function} options.onStroke      receives a stroke added elsewhere
 * @param {Function} [options.onRemoved]   receives a removed stroke id
 * @param {Function} [options.onAuthorCleared] receives the cleared author id
 * @param {Function} [options.now]         injectable clock for stroke ids
 */
export class AnnotationChannel {
    constructor({
        deviceId,
        viewerId,
        onStroke,
        onRemoved,
        onAuthorCleared,
        now = () => Date.now(),
        BroadcastChannelImpl = globalThis.BroadcastChannel,
        storage = null,
    } = {}) {
        this.deviceId = deviceId || ''
        this.viewerId = viewerId || ''
        this._onStroke = onStroke || (() => {})
        this._onRemoved = onRemoved || (() => {})
        this._onAuthorCleared = onAuthorCleared || (() => {})
        this._now = now
        this._BroadcastChannelImpl = BroadcastChannelImpl
        this._storage = storage
        this._channel = null
        this._started = false
        this._nextId = 1
    }

    start() {
        if (this._started || !this.deviceId) return
        this._started = true
        if (this._BroadcastChannelImpl) {
            try {
                this._channel = new this._BroadcastChannelImpl(CHANNEL_PREFIX + this.deviceId)
                this._channel.onmessage = (event) => this._handle(event.data)
                return
            } catch {
                this._channel = null
            }
        }
        // No BroadcastChannel: annotations stay local to this viewer rather than
        // failing. A viewer that cannot share is a degraded viewer, not a broken
        // one.
    }

    /** A stroke id that is unique per viewer and per stroke. */
    nextStrokeId() {
        const random = Math.random().toString(36).slice(2, 8)
        return `ann-${this.viewerId || 'viewer'}-${this._now().toString(36)}-${this._nextId++}-${random}`
    }

    /** Publishes a stroke this viewer just finished. */
    publishAdd(stroke) {
        if (!stroke) return
        this._post({ type: ANNOTATION_MSG_ADD, viewerId: this.viewerId, stroke: stroke.toJSON() })
    }

    publishRemove(strokeId) {
        if (!strokeId) return
        this._post({ type: ANNOTATION_MSG_REMOVE, viewerId: this.viewerId, strokeId })
    }

    /**
     * Publishes "I cleared mine". Peers drop only this viewer's strokes, so a
     * clear can never wipe work the sender never drew.
     */
    publishClearAuthor() {
        this._post({ type: ANNOTATION_MSG_CLEAR_AUTHOR, viewerId: this.viewerId })
    }

    destroy() {
        if (!this._started) return
        this._started = false
        if (this._channel) {
            this._channel.close()
            this._channel = null
        }
    }

    _post(message) {
        if (!this._channel) return
        try {
            this._channel.postMessage({ ...message, deviceId: this.deviceId })
        } catch {
            // A closed or full channel must not break the annotation the user
            // is drawing right now.
        }
    }

    _handle(message) {
        if (!message || typeof message !== 'object') return
        // A message for another device is not ours, and neither is one this
        // viewer sent: the sender already has the stroke in its own model.
        if (message.deviceId !== this.deviceId) return
        if (message.viewerId === this.viewerId) return

        switch (message.type) {
            case ANNOTATION_MSG_ADD: {
                const stroke = Stroke.fromJSON(message.stroke)
                if (stroke) this._onStroke(stroke)
                return
            }
            case ANNOTATION_MSG_REMOVE:
                if (message.strokeId) this._onRemoved(String(message.strokeId))
                return
            case ANNOTATION_MSG_CLEAR_AUTHOR:
                if (message.viewerId) this._onAuthorCleared(String(message.viewerId))
                return
            default:
        }
    }
}

import { Stroke } from './annotationModel.js'
