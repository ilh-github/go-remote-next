import assert from 'node:assert/strict';
import test from 'node:test';

test('file transfer error type 2 explains host save failure', async () => {
    const { formatFileTransferError } = await import('../js/protocol/file-transfer-errors.js');
    const message = formatFileTransferError(2);

    assert.match(message, /type=2/);
    assert.match(message, /disk space/i);
    assert.match(message, /permission/i);
});

test('file transfer error type 1 is reported as cancelled', async () => {
    const { formatFileTransferError } = await import('../js/protocol/file-transfer-errors.js');
    assert.match(formatFileTransferError(1), /cancel/i);
});

test('unknown file transfer errors preserve the raw type', async () => {
    const { formatFileTransferError } = await import('../js/protocol/file-transfer-errors.js');
    assert.match(formatFileTransferError(99), /type=99/);
});

test('upload cancellation uses the protocol canceled value', async () => {
    const { FILE_TRANSFER_ERROR_CANCELED } = await import('../js/protocol/file-transfer-errors.js');
    assert.equal(FILE_TRANSFER_ERROR_CANCELED, 1);
});
