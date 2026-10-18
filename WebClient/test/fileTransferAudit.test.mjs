import assert from 'node:assert/strict';
import test from 'node:test';

function installStorage() {
  const values = new Map();
  globalThis.localStorage = {
    getItem(key) { return values.has(key) ? values.get(key) : null; },
    setItem(key, value) { values.set(key, String(value)); },
    removeItem(key) { values.delete(key); },
  };
  return values;
}

test('file transfer reports post the documented audit payload', async () => {
  const values = installStorage();
  values.set('quickdesk_user_access_token', 'access-token');
  const calls = [];
  globalThis.fetch = async (url, init) => {
    calls.push({ url, init });
    return {
      ok: true,
      status: 204,
      headers: new Map(),
      json: async () => null,
    };
  };

  try {
    const { UserApi } = await import('../js/api/user-api.js');
    const api = new UserApi('ws://127.0.0.1:8000');
    const result = await api.recordFileTransferAudit({
      deviceId: '123456789',
      transferId: 'transfer-1',
      direction: 'upload',
      status: 'completed',
      filename: 'report.pdf',
      bytesTransferred: 25,
      totalBytes: 25,
    });

    assert.equal(result.ok, true);
    assert.equal(calls.length, 1);
    assert.equal(calls[0].url, 'http://127.0.0.1:8000/v1/me/file-transfer-audits');
    assert.equal(calls[0].init.method, 'POST');
    assert.equal(calls[0].init.headers.Authorization, 'Bearer access-token');
    assert.deepEqual(JSON.parse(calls[0].init.body), {
      device_id: '123456789',
      transfer_id: 'transfer-1',
      direction: 'upload',
      status: 'completed',
      filename: 'report.pdf',
      bytes_transferred: 25,
      total_bytes: 25,
      error: '',
    });
  } finally {
    delete globalThis.fetch;
    delete globalThis.localStorage;
  }
});
