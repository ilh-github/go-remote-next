// Exported so a test can assert against the real limit instead of a copy of the
// number. A boundary that only exists as a literal in two places is a boundary
// nobody can check: raise one side and nothing goes red.
export const MAX_CHAT_TEXT_LENGTH = 4000;

export class SessionChatModel extends EventTarget {
    constructor({ send, now = () => new Date().toISOString(), idFactory } = {}) {
        super();
        if (typeof send !== 'function') {
            throw new TypeError('send must be a function');
        }
        this._send = send;
        this._now = now;
        this._idFactory = idFactory || (() => {
            const random = Math.random().toString(36).slice(2, 10);
            return `web-chat-${Date.now().toString(36)}-${random}`;
        });
        this._nextId = 1;
        this._unreadCount = 0;
        this._outgoingIds = new Set();
    }

    get unreadCount() {
        return this._unreadCount;
    }

    send(text) {
        const normalized = String(text ?? '').trim();
        if (!normalized) {
            this._emitError('Chat message must not be empty');
            return false;
        }
        if (normalized.length > MAX_CHAT_TEXT_LENGTH) {
            this._emitError(`Chat message exceeds ${MAX_CHAT_TEXT_LENGTH} characters`);
            return false;
        }

        try {
            const id = this._idFactory(this._nextId++);
            this._send({
                id,
                type: 'chatSend',
                text: normalized,
                sentAt: this._now(),
            });
            this._outgoingIds.add(id);
            return true;
        } catch (error) {
            this._emitError(`Failed to send chat message: ${error.message || error}`);
            return false;
        }
    }

    handleMessage(message) {
        if (!message || typeof message !== 'object') return;
        const type = message.type || '';

        if (type === 'chatMessage') {
            const outgoing = String(message.from || '') === 'controller'
                && this._outgoingIds.has(String(message.id || ''));
            if (outgoing) {
                this._outgoingIds.delete(String(message.id || ''));
            }
            const detail = {
                id: String(message.id || ''),
                from: String(message.from || 'host'),
                text: String(message.text || ''),
                sentAt: String(message.sentAt || ''),
                outgoing,
            };
            if (!detail.text) return;
            // QD-316's boundary is bidirectional. The send side enforced it; the
            // receive side did not, so a host that was older, different, or not
            // validating could push an unbounded string into the DOM - and the
            // renderer is the side that pays for it. Refused rather than
            // truncated: a shortened message reads as the sender's words.
            if (detail.text.length > MAX_CHAT_TEXT_LENGTH) {
                this._emitError(`Chat message exceeds ${MAX_CHAT_TEXT_LENGTH} characters`);
                return;
            }
            if (!outgoing) {
                this._unreadCount++;
                this._emitUnreadChanged();
            }
            this.dispatchEvent(new CustomEvent('messageAdded', { detail }));
            return;
        }

        if (type === 'chatAck' && message.accepted === false) {
            this._outgoingIds.delete(String(message.id || ''));
            this._emitError(String(message.error || 'Host rejected the chat message'));
        }
    }

    markRead() {
        if (this._unreadCount === 0) return;
        this._unreadCount = 0;
        this._emitUnreadChanged();
    }

    _emitUnreadChanged() {
        this.dispatchEvent(new CustomEvent('unreadChanged', {
            detail: { count: this._unreadCount },
        }));
    }

    _emitError(message) {
        this.dispatchEvent(new CustomEvent('error', {
            detail: { message },
        }));
    }
}
