// Building the admin download links for a recording.
//
// The admin list shows a row's size and status, but the bytes are a separate
// route and a browser cannot attach a Bearer token to a plain <a href> or an
// <img src>. So these are fetched through authFetch and turned into a blob URL,
// which is the pattern the rest of the admin app already uses for downloads.
import test from 'node:test'
import assert from 'node:assert/strict'

const mod = await import('../src/views/recording-download-model.js')

test('an artifact link names the admin route, not the owner one', () => {
  // The owner route is scoped to the caller's own devices; an admin console has
  // to read any account's recording.
  assert.equal(mod.artifactUrl(42), '/v1/admin/recordings/42/artifact')
  assert.equal(mod.thumbnailUrl(42), '/v1/admin/recordings/42/thumbnail')
})

test('a link is offered only when the row actually has media', () => {
  // A recording the server is still writing has no bytes; a download button
  // there is a guaranteed failure with no explanation.
  assert.equal(mod.canDownload({ status: 'completed', size_bytes: 1024 }), true)
  assert.equal(mod.canDownload({ status: 'recording', size_bytes: 0 }), false)
  assert.equal(mod.canDownload({ status: 'failed', size_bytes: 0 }), false)
})

test('a row with no recorded size offers nothing to download', () => {
  assert.equal(mod.canDownload({ status: 'completed', size_bytes: 0 }), false)
})

test('a thumbnail is only offered when the row has one', () => {
  // Not every recording produces a thumbnail, and a broken image in a table
  // reads as a bug rather than "not captured".
  assert.equal(mod.canThumbnail({ thumbnail_path: 'thumbs/a.png' }), true)
  assert.equal(mod.canThumbnail({ thumbnail_path: '' }), false)
  assert.equal(mod.canThumbnail({}), false)
})

test('a filename carries the recording id so two exports stay distinguishable', () => {
  assert.equal(mod.artifactName(42), 'recording-42.h264')
})

test('an id that is not a number produces no link rather than a broken one', () => {
  assert.equal(mod.artifactUrl('abc'), '')
  assert.equal(mod.artifactUrl(0), '')
})

test('the row id is escaped into the path', () => {
  // A path segment is interpolated, so an id carrying a slash would otherwise
  // turn into an extra segment and hit the wrong route.
  assert.equal(mod.artifactUrl('7/8'), '')
})
