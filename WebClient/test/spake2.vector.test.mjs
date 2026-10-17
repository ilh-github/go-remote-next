import assert from 'node:assert/strict';
import test from 'node:test';

import {
  SPAKE2_ROLE_ALICE,
  SPAKE2_ROLE_BOB,
  Spake2Context,
} from '../js/auth/spake2.js';
import { bytesToHex, getSharedSecretHash } from '../js/auth/auth-util.js';

const EXPECTED = {
  clientMessage:
    '8d57e26fd5c4bc842ebaa235e39ebf1aadceea1d4dc436d91362beeb9a8934c9',
  hostMessage:
    'caf5b3dc942dc21a66cefbab23fe9f684a42fd2dfa06fb35d59f29019c7ff7c6',
  authKey:
    'e6cf084f085b5d13552e564130c70136d4af595f3d09f099d7f0583fc02e9c6c' +
    'fac4bbb5d4818aa0bac3fa249caa7bd1d028db2e48e49772a56fb0d0282db72f',
  clientVerificationHash:
    '5592d1521d3cdbb7a2caf217309585f5bfa516b3b72883862613e9959093c78f',
  hostVerificationHash:
    '30f60ffad1342128f95dd8bbbd48e0096e8df610fe634f9246ac34f12f214297',
};

function scalarToRandomBytes(hex) {
  let value = BigInt(`0x${hex}`);
  const bytes = new Uint8Array(64);
  for (let index = 0; index < bytes.length; index += 1) {
    bytes[index] = Number(value & 0xffn);
    value >>= 8n;
  }
  return bytes;
}

function installDeterministicRandom(bytes) {
  Object.defineProperty(globalThis.crypto, 'getRandomValues', {
    configurable: true,
    value(array) {
      array.set(bytes);
      return array;
    },
  });
}

test('matches Chromium 140 BoringSSL fixed byte vector', async () => {
  const originalRandom = Object.getOwnPropertyDescriptor(
    globalThis.crypto,
    'getRandomValues',
  );
  const password = await getSharedSecretHash(
    '123456789',
    '123456789424243',
  );
  const clientJid =
    'client_01234567-89ab-4cde-8f01-23456789abcd@quickdesk.local/' +
    'chromoting_ftl_quickdesk_host';
  const hostJid =
    '123456789@quickdesk.local/chromoting_ftl_quickdesk_host';

  try {
    installDeterministicRandom(
      scalarToRandomBytes('123456789abcdef123456789abcdef'),
    );
    const client = new Spake2Context(
      SPAKE2_ROLE_ALICE,
      clientJid,
      hostJid,
    );
    const clientMessage = await client.generateMessage(password);

    installDeterministicRandom(
      scalarToRandomBytes('fedcba9876543210fedcba987654321'),
    );
    const host = new Spake2Context(SPAKE2_ROLE_BOB, hostJid, clientJid);
    const hostMessage = await host.generateMessage(password);

    const clientKey = await client.processMessage(hostMessage);
    const hostKey = await host.processMessage(clientMessage);
    const clientVerificationHash =
      await client.getOutgoingVerificationHash();
    const hostVerificationHash = await host.getOutgoingVerificationHash();

    assert.equal(bytesToHex(clientMessage), EXPECTED.clientMessage);
    assert.equal(bytesToHex(hostMessage), EXPECTED.hostMessage);
    assert.equal(bytesToHex(clientKey), EXPECTED.authKey);
    assert.equal(bytesToHex(hostKey), EXPECTED.authKey);
    assert.equal(
      bytesToHex(clientVerificationHash),
      EXPECTED.clientVerificationHash,
    );
    assert.equal(
      bytesToHex(hostVerificationHash),
      EXPECTED.hostVerificationHash,
    );
  } finally {
    if (originalRandom) {
      Object.defineProperty(
        globalThis.crypto,
        'getRandomValues',
        originalRandom,
      );
    } else {
      delete globalThis.crypto.getRandomValues;
    }
  }
});
