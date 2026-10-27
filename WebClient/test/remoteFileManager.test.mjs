import assert from 'node:assert/strict';
import test from 'node:test';

import { RemoteFileManager } from '../js/files/remote-file-manager.js';

function fakeFile(name, bytes) {
    return {
        name,
        size: bytes.length,
        slice(start, end) {
            return {
                async arrayBuffer() {
                    return Uint8Array.from(bytes.slice(start, end)).buffer;
                },
            };
        },
    };
}

test('upload writes bounded chunks and truncates only the first chunk', async () => {
    const requests = [];
    const progress = [];
    const bridge = {
        async request(tool, args) {
            requests.push({ tool, args });
            return { bytes_written: args.offset === 0 ? 3 : 2 };
        },
    };
    const manager = new RemoteFileManager(bridge, {
        chunkBytes: 3,
        refreshAfterUpload: false,
    });
    manager.currentPath = '/tmp';
    manager.addEventListener('uploadProgress', (event) => progress.push(event.detail));

    await manager.upload(fakeFile('sample.bin', [1, 2, 3, 4, 5]), '/incoming');

    assert.deepEqual(requests.map(item => item.tool), [
        'write_file_chunk',
        'write_file_chunk',
    ]);
    assert.equal(requests[0].args.path, '/incoming/sample.bin');
    assert.equal(requests[0].args.offset, 0);
    assert.equal(requests[0].args.truncate, true);
    assert.equal(requests[1].args.offset, 3);
    assert.equal(requests[1].args.truncate, false);
    assert.deepEqual(progress.map(item => item.written), [3, 5]);
});

test('download assembles base64 chunks in server order', async () => {
    const bridge = {
        async request(tool, args) {
            assert.equal(tool, 'read_file_chunk');
            if (args.offset === 0) {
                return {
                    offset: 0,
                    next_offset: 2,
                    size: 3,
                    eof: false,
                    data_base64: 'AQI=',
                };
            }
            return {
                offset: 2,
                next_offset: 3,
                size: 3,
                eof: true,
                data_base64: 'Aw==',
            };
        },
    };
    const manager = new RemoteFileManager(bridge);

    const result = await manager.download({ name: 'sample.bin', isDirectory: false });

    assert.equal(result.filename, 'sample.bin');
    assert.deepEqual([...result.bytes], [1, 2, 3]);
});

test('rename validates the target and moves within the same directory', async () => {
    const requests = [];
    const bridge = {
        async request(tool, args) {
            requests.push({ tool, args });
            return {};
        },
    };
    const manager = new RemoteFileManager(bridge, { refreshAfterMutation: false });
    manager.currentPath = 'C:\\Users\\tim';

    await manager.rename({ name: 'old.txt' }, 'new.txt');
    assert.deepEqual(requests[0], {
        tool: 'move_file',
        args: {
            source: 'C:\\Users\\tim\\old.txt',
            destination: 'C:\\Users\\tim\\new.txt',
        },
    });
    await assert.rejects(manager.rename({ name: 'old.txt' }, 'bad/name'), /separator/);
});

// The host's file-ops skill rejects any chunk above 256 KiB and silently clamps
// read requests down to it. A manager configured above the limit would fail every
// upload with an error naming a size the caller never chose.
test('a chunk size above the host limit is clamped', () => {
    const manager = new RemoteFileManager({ async request() { return {} } }, {
        chunkBytes: 4 * 1024 * 1024,
    });

    assert.equal(manager.chunkBytes, 256 * 1024);
});

test('a chunk size within the host limit is left alone', () => {
    const manager = new RemoteFileManager({ async request() { return {} } }, {
        chunkBytes: 3,
    });

    assert.equal(manager.chunkBytes, 3);
});

// A zero or negative chunk size would make the upload loop never advance, and the
// host would reject a zero-length write as an empty chunk.
test('a zero or negative chunk size falls back to the default', () => {
    for (const value of [0, -1, NaN, 'not a number']) {
        const manager = new RemoteFileManager({ async request() { return {} } }, {
            chunkBytes: value,
        });
        assert.equal(manager.chunkBytes, 64 * 1024, `chunk size ${value} was kept`);
    }
});
