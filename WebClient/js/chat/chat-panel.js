import { t } from '../i18n.js';

export class ChatPanel {
    constructor(container, model) {
        this.container = container;
        this.model = model;
        this._build();

        this.model.addEventListener('messageAdded', event => {
            this._appendMessage(event.detail);
        });
        this.model.addEventListener('unreadChanged', event => {
            this._unreadBadge.textContent = String(event.detail.count);
            this._unreadBadge.hidden = event.detail.count === 0;
        });
        this.model.addEventListener('error', event => {
            this._status.textContent = event.detail.message;
            this._status.hidden = false;
        });
    }

    get isOpen() {
        return this._root.hidden === false;
    }

    open() {
        this._root.hidden = false;
        this._status.hidden = true;
        this.model.markRead();
        this._input.focus();
        requestAnimationFrame(() => {
            this._messages.scrollTop = this._messages.scrollHeight;
        });
    }

    close() {
        this._root.hidden = true;
    }

    refreshI18n() {
        this._title.textContent = t('chat.title');
        this._subtitle.textContent = t('chat.subtitle');
        this._closeButton.title = t('chat.close');
        this._emptyTitle.textContent = t('chat.emptyTitle');
        this._emptyDescription.textContent = t('chat.emptyDescription');
        this._input.placeholder = t('chat.placeholder');
        this._sendButton.textContent = t('chat.send');
    }

    _build() {
        const root = document.createElement('div');
        root.className = 'chat-root';
        root.hidden = true;
        root.innerHTML = `
            <div class="chat-backdrop"></div>
            <section class="chat-panel" role="dialog" aria-modal="false">
                <header class="chat-header">
                    <div>
                        <div class="chat-title"></div>
                        <div class="chat-subtitle" data-i18n="chat.subtitle">Encrypted in the active session</div>
                    </div>
                    <button type="button" class="chat-close" title="Close">&times;</button>
                </header>
                <div class="chat-messages" role="log" aria-live="polite"></div>
                <div class="chat-empty">
                    <div class="chat-empty-title"></div>
                    <div class="chat-empty-description"></div>
                </div>
                <div class="chat-status" hidden></div>
                <form class="chat-composer">
                    <input class="chat-input" type="text" maxlength="4000" autocomplete="off">
                    <button type="submit" class="chat-send"></button>
                </form>
                <div class="chat-unread" hidden>0</div>
            </section>
        `;
        this.container.appendChild(root);

        this._root = root;
        this._title = root.querySelector('.chat-title');
        this._subtitle = root.querySelector('.chat-subtitle');
        this._closeButton = root.querySelector('.chat-close');
        this._messages = root.querySelector('.chat-messages');
        this._emptyTitle = root.querySelector('.chat-empty-title');
        this._emptyDescription = root.querySelector('.chat-empty-description');
        this._status = root.querySelector('.chat-status');
        this._form = root.querySelector('.chat-composer');
        this._input = root.querySelector('.chat-input');
        this._sendButton = root.querySelector('.chat-send');
        this._unreadBadge = root.querySelector('.chat-unread');

        this._closeButton.addEventListener('click', () => this.close());
        root.querySelector('.chat-backdrop').addEventListener('click', () => this.close());
        this._form.addEventListener('submit', event => {
            event.preventDefault();
            if (this.model.send(this._input.value)) {
                this._input.value = '';
                this._sendButton.disabled = true;
                this._status.hidden = true;
            }
        });
        this._input.addEventListener('input', () => {
            this._sendButton.disabled = !this._input.value.trim();
        });
        this._sendButton.disabled = true;
        root.addEventListener('keydown', event => {
            if (event.key === 'Escape') this.close();
        });

        this.refreshI18n();
    }

    _appendMessage(message) {
        const row = document.createElement('div');
        row.className = `chat-message ${message.outgoing ? 'outgoing' : 'incoming'}`;

        const bubble = document.createElement('div');
        bubble.className = 'chat-bubble';
        bubble.textContent = message.text;

        const meta = document.createElement('div');
        meta.className = 'chat-meta';
        const sender = message.outgoing
            ? t('chat.you')
            : (message.from === 'host' ? t('chat.host') : t('chat.controller'));
        const time = this._formatTime(message.sentAt);
        meta.textContent = time ? `${sender} · ${time}` : sender;

        row.appendChild(bubble);
        row.appendChild(meta);
        this._messages.appendChild(row);
        while (this._messages.children.length > 200) {
            this._messages.firstElementChild.remove();
        }
        this._emptyTitle.parentElement.hidden = true;
        requestAnimationFrame(() => {
            this._messages.scrollTop = this._messages.scrollHeight;
        });
    }

    _formatTime(value) {
        const date = new Date(value);
        if (Number.isNaN(date.getTime())) return '';
        return new Intl.DateTimeFormat(undefined, {
            hour: '2-digit',
            minute: '2-digit',
        }).format(date);
    }
}
