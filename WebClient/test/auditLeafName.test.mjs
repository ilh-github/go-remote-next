import assert from 'node:assert/strict'
import test from 'node:test'

// QD-909: the file-transfer audit reports a filename, and a filename that
// carries a path is information about the controlled machine rather than about
// the transfer. The server now refuses one; this is the client's half, which
// drops it before it is ever a candidate.
const { auditLeafName } = await import('../js/audit/audit-leaf-name.js')

test('a plain leaf is reported unchanged', () => {
    assert.equal(auditLeafName('report.pdf'), 'report.pdf')
    assert.equal(auditLeafName('Q3 report final.pdf'), 'Q3 report final.pdf')
    assert.equal(auditLeafName('archive.tar.gz'), 'archive.tar.gz')
    assert.equal(auditLeafName('.gitignore'), '.gitignore')
})

test('a posix path reduces to its leaf', () => {
    assert.equal(auditLeafName('/Users/someone/.ssh/id_rsa'), 'id_rsa')
    assert.equal(auditLeafName('docs/report.pdf'), 'report.pdf')
    assert.equal(auditLeafName('secrets/'), '')
})

test('a windows path reduces to its leaf', () => {
    assert.equal(auditLeafName(String.raw`C:\Users\someone\.ssh\id_rsa`), 'id_rsa')
    assert.equal(auditLeafName(String.raw`\\server\share\keys\id_rsa`), 'id_rsa')
})

test('a bare drive letter names nothing', () => {
    assert.equal(auditLeafName('C:'), '')
    assert.equal(auditLeafName('z:'), '')
})

test('a traversal is reduced to its leaf, not resolved', () => {
    // The point is not to make the path safe - it never reaches a filesystem -
    // but to keep the directory layout out of the audit row.
    assert.equal(auditLeafName('docs/../../../etc/passwd'), 'passwd')
})

test('anything that is not a string is an empty name', () => {
    assert.equal(auditLeafName(''), '')
    assert.equal(auditLeafName(undefined), '')
    assert.equal(auditLeafName(null), '')
    assert.equal(auditLeafName(42), '')
    assert.equal(auditLeafName({}), '')
})

test('surrounding whitespace does not survive', () => {
    assert.equal(auditLeafName('  report.pdf  '), 'report.pdf')
})
