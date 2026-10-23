import assert from 'node:assert/strict'
import test from 'node:test'

import { AnnotationChannel, ANNOTATION_MSG_ADD } from '../js/ui/annotationChannel.js'
import { AnnotationModel, Stroke } from '../js/ui/annotationModel.js'

class FakeBroadcastChannel {
    static channels = new Map()

    constructor(name) {
        this.name = name
        this.onmessage = null
        this.closed = false
        const peers = FakeBroadcastChannel.channels.get(name) || new Set()
        peers.add(this)
        FakeBroadcastChannel.channels.set(name, peers)
    }

    postMessage(data) {
        const peers = FakeBroadcastChannel.channels.get(this.name) || new Set()
        for (const peer of peers) {
            if (peer !== this && !peer.closed) {
                queueMicrotask(() => peer.onmessage?.({ data }))
            }
        }
    }

    close() {
        this.closed = true
        FakeBroadcastChannel.channels.get(this.name)?.delete(this)
    }
}

function resetChannels() {
    FakeBroadcastChannel.channels.clear()
}

function makeViewer(deviceId, viewerId, handlers = {}) {
    return new AnnotationChannel({
        deviceId,
        viewerId,
        now: () => 1700000000000,
        BroadcastChannelImpl: FakeBroadcastChannel,
        ...handlers,
    })
}

// The core of "shared": one viewer's finished stroke appears in another's model.
test('a stroke published by one viewer reaches the other', async () => {
    resetChannels()
    const remote = new AnnotationModel()
    const remoteChannel = makeViewer('device-a', 'viewer-b', {
        onStroke: (stroke) => remote.add(stroke),
    })
    remoteChannel.start()

    const local = new AnnotationModel()
    const localChannel = makeViewer('device-a', 'viewer-a')
    localChannel.start()

    const stroke = local.begin({
        id: localChannel.nextStrokeId(),
        authorId: 'viewer-a',
        x: 10,
        y: 20,
    })
    local.extend(30, 40)
    local.end()
    localChannel.publishAdd(stroke)

    await new Promise((resolve) => setTimeout(resolve, 10))
    assert.equal(remote.strokes.length, 1, 'the remote viewer never received the stroke')
    assert.deepEqual(remote.strokes[0].points, [10, 20, 30, 40])
    assert.equal(remote.strokes[0].authorId, 'viewer-a')
})

// The sender already has its own stroke; echoing it back would double-count it.
test('a viewer does not receive its own stroke back', async () => {
    resetChannels()
    const model = new AnnotationModel()
    const channel = makeViewer('device-a', 'viewer-a', {
        onStroke: (stroke) => model.add(stroke),
    })
    channel.start()

    const stroke = model.begin({ id: channel.nextStrokeId(), authorId: 'viewer-a', x: 1, y: 1 })
    model.extend(2, 2)
    model.end()
    channel.publishAdd(stroke)

    await new Promise((resolve) => setTimeout(resolve, 10))
    assert.equal(model.strokes.length, 1, 'the echo duplicated the stroke')
})

// Devices are separate rooms: a stroke on one must not appear on another.
test('a stroke does not cross devices', async () => {
    resetChannels()
    const other = new AnnotationModel()
    const otherChannel = makeViewer('device-b', 'viewer-b', {
        onStroke: (stroke) => other.add(stroke),
    })
    otherChannel.start()

    const local = new AnnotationModel()
    const localChannel = makeViewer('device-a', 'viewer-a')
    localChannel.start()
    const stroke = local.begin({ id: 's1', authorId: 'viewer-a', x: 1, y: 1 })
    local.extend(2, 2)
    local.end()
    localChannel.publishAdd(stroke)

    await new Promise((resolve) => setTimeout(resolve, 10))
    assert.equal(other.strokes.length, 0, 'a stroke crossed to another device')
})

// Clearing is scoped by author. A clear that broadcast "empty yourself" would
// let one viewer wipe everybody's annotations, which is the failure mode this
// whole design avoids.
test('a clear only removes the clearing viewer\'s strokes', async () => {
    resetChannels()
    const remote = new AnnotationModel()
    const remoteChannel = makeViewer('device-a', 'viewer-b', {
        onStroke: (stroke) => remote.add(stroke),
        onAuthorCleared: (authorId) => remote.clearAuthor(authorId),
    })
    remoteChannel.start()

    const local = new AnnotationModel()
    const localChannel = makeViewer('device-a', 'viewer-a')
    localChannel.start()

    // Viewer A draws two strokes; viewer B draws one of its own.
    for (const id of ['a1', 'a2']) {
        const stroke = local.begin({ id, authorId: 'viewer-a', x: 1, y: 1 })
        local.extend(2, 2)
        local.end()
        localChannel.publishAdd(stroke)
    }
    await new Promise((resolve) => setTimeout(resolve, 10))
    remote.add(new Stroke({ id: 'b1', authorId: 'viewer-b', points: [5, 6, 7, 8] }))
    assert.equal(remote.strokes.length, 3)

    localChannel.publishClearAuthor()
    await new Promise((resolve) => setTimeout(resolve, 10))

    assert.equal(remote.strokes.length, 1, 'a clear wiped another viewer\'s strokes')
    assert.equal(remote.strokes[0].authorId, 'viewer-b')
})

test('a remove is propagated by stroke id', async () => {
    resetChannels()
    const remote = new AnnotationModel()
    const remoteChannel = makeViewer('device-a', 'viewer-b', {
        onStroke: (stroke) => remote.add(stroke),
        onRemoved: (id) => remote.remove(id),
    })
    remoteChannel.start()

    const local = new AnnotationModel()
    const localChannel = makeViewer('device-a', 'viewer-a')
    localChannel.start()
    const stroke = local.begin({ id: 's1', authorId: 'viewer-a', x: 1, y: 1 })
    local.extend(2, 2)
    local.end()
    localChannel.publishAdd(stroke)
    await new Promise((resolve) => setTimeout(resolve, 10))
    assert.equal(remote.strokes.length, 1)

    localChannel.publishRemove('s1')
    await new Promise((resolve) => setTimeout(resolve, 10))
    assert.equal(remote.strokes.length, 0, 'the remove never reached the remote viewer')
})

// A viewer never receives its own stroke back, even through a real
// BroadcastChannel - which is what Node provides by default, so this test is
// also the one that proves the sender filter is not just an artefact of the fake.
test('a real BroadcastChannel does not echo a stroke to its sender', async () => {
    const received = []
    const channel = makeViewer('device-a', 'viewer-a', {
        onStroke: (stroke) => received.push(stroke),
    })
    channel.start()
    channel.publishAdd({ toJSON: () => ({ id: 's', points: [1, 2, 3, 4] }) })
    await new Promise((resolve) => setTimeout(resolve, 20))
    assert.equal(received.length, 0)
    // A live BroadcastChannel keeps the event loop alive, so the runner would
    // wait on a channel nobody will message again.
    channel.destroy()
})

test('a malformed message is ignored rather than throwing', async () => {
    resetChannels()
    const received = []
    const channel = makeViewer('device-a', 'viewer-b', {
        onStroke: (stroke) => received.push(stroke),
    })
    channel.start()
    // A peer that is not one of ours, reached through the same channel name.
    const peer = new FakeBroadcastChannel('quickdesk:annotations:device-a')
    peer.postMessage({ type: ANNOTATION_MSG_ADD, deviceId: 'device-a', viewerId: 'viewer-c', stroke: 'nonsense' })
    peer.postMessage(null)
    peer.postMessage({ type: 'unknown', deviceId: 'device-a', viewerId: 'viewer-c' })
    await new Promise((resolve) => setTimeout(resolve, 10))
    assert.equal(received.length, 0)
})

test('a closed channel stops delivering', async () => {
    resetChannels()
    FakeBroadcastChannel.channels.set('quickdesk:annotations:device-a', new Set())
    const received = []
    const channel = makeViewer('device-a', 'viewer-b', {
        onStroke: (stroke) => received.push(stroke),
    })
    channel.start()
    channel.destroy()

    const peer = new FakeBroadcastChannel('quickdesk:annotations:device-a')
    peer.postMessage({
        type: ANNOTATION_MSG_ADD,
        deviceId: 'device-a',
        viewerId: 'viewer-c',
        stroke: { id: 's', points: [1, 2, 3, 4] },
    })
    await new Promise((resolve) => setTimeout(resolve, 10))
    assert.equal(received.length, 0)
})

// Every channel this file opened is closed here. Without it the runner waits on
// a registry that still holds live peers, and the file reports a cancellation
// rather than a result.
test.after(() => {
    for (const peers of FakeBroadcastChannel.channels.values()) {
        for (const peer of peers) peer.close()
    }
    FakeBroadcastChannel.channels.clear()
})
