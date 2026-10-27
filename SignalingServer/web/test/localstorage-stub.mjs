// Minimal browser surface so the real API modules can load under node:test.
// The admin API touches localStorage at import time (legacy token cleanup), so
// importing any src/api module without this fails outright.
const store = new Map()
globalThis.localStorage = {
  getItem: (k) => (store.has(k) ? store.get(k) : null),
  setItem: (k, v) => store.set(k, String(v)),
  removeItem: (k) => store.delete(k),
}
