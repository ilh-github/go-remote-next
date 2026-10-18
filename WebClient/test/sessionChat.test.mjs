import assert from 'node:assert/strict';
import test from 'node:test';

import { SessionChatModel } from '../js/chat/session-chat.js';

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
