import assert from 'node:assert/strict';
import test from 'node:test';

import { SessionChatModel, MAX_CHAT_TEXT_LENGTH } from '../js/chat/session-chat.js';

test('sending chat emits a correlated chatSend and receives host replies', () => {
    const sent = [];
    const model = new SessionChatModel({
        send: message => sent.push(message),
        now: () => '2026-10-18T00:00:00.000Z',
        idFactory: () => 'chat-1',
    });
    const added = [];
    const unread = [];
    model.addEventListener('messageAdded', event => added.push(event.detail));
    model.addEventListener('unreadChanged', event => unread.push(event.detail.count));

    model.send('  hello  ');
    assert.deepEqual(sent, [{
        id: 'chat-1',
        type: 'chatSend',
        text: 'hello',
        sentAt: '2026-10-18T00:00:00.000Z',
    }]);

    model.handleMessage({
        id: 'host-chat-1',
        type: 'chatMessage',
        from: 'host',
        text: 'hi',
        sentAt: '2026-10-18T00:00:01.000Z',
    });
    assert.equal(model.unreadCount, 1);
    assert.deepEqual(unread, [1]);
    assert.deepEqual(added, [{
        id: 'host-chat-1',
        from: 'host',
        text: 'hi',
        sentAt: '2026-10-18T00:00:01.000Z',
        outgoing: false,
    }]);

    model.markRead();
    assert.equal(model.unreadCount, 0);
});

test('the echoed controller message is outgoing and does not increment unread', () => {
    const model = new SessionChatModel({
        send: () => {},
        now: () => '2026-10-18T00:00:00.000Z',
        idFactory: () => 'chat-2',
    });
    const added = [];
    model.addEventListener('messageAdded', event => added.push(event.detail));

    assert.equal(model.send('hello'), true);
    model.handleMessage({
        id: 'chat-2',
        type: 'chatMessage',
        from: 'controller',
        text: 'hello',
        sentAt: '2026-10-18T00:00:00.100Z',
    });

    assert.equal(model.unreadCount, 0);
    assert.deepEqual(added, [{
        id: 'chat-2',
        from: 'controller',
        text: 'hello',
        sentAt: '2026-10-18T00:00:00.100Z',
        outgoing: true,
    }]);
});

// The length boundary is enforced in both the controller and the host, and until
// now nothing asserted it on either side. It is the kind of limit that gets
// "cleaned up" by someone reading the constant as an arbitrary magic number.
test('a message at the limit is sent, one over it is refused', () => {
    const sent = [];
    const model = new SessionChatModel({ send: message => sent.push(message) });

    assert.equal(model.send('a'.repeat(MAX_CHAT_TEXT_LENGTH)), true);
    assert.equal(sent.length, 1);

    assert.equal(model.send('a'.repeat(MAX_CHAT_TEXT_LENGTH + 1)), false);
    assert.equal(sent.length, 1, 'the refused message must not have been sent');
});

test('the refusal explains the limit rather than failing silently', () => {
    const errors = [];
    const model = new SessionChatModel({ send: () => {} });
    model.addEventListener('error', event => errors.push(event.detail.message));
    model.send('a'.repeat(MAX_CHAT_TEXT_LENGTH + 1));
    assert.equal(errors.length, 1);
    assert.match(errors[0], new RegExp(`${MAX_CHAT_TEXT_LENGTH}`));
});

test('the limit counts characters, not bytes', () => {
    // A message of multi-byte characters must not be rejected for being long in
    // bytes. Both ends count UTF-16 code units, so this is the same number.
    const sent = [];
    const model = new SessionChatModel({ send: message => sent.push(message) });
    assert.equal(model.send('中'.repeat(MAX_CHAT_TEXT_LENGTH)), true);
    assert.equal(sent.length, 1);
});
