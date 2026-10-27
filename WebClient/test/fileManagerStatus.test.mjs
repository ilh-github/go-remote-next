import assert from 'node:assert/strict'
import test from 'node:test'

// QD-002: the file manager's status line has one rule - a message announces
// what it is about, and a later message only displaces an earlier one when it is
// about the same subject. StatusLine states that rule and its tests are
// thorough.
//
// What was never pinned is whether the panel actually routes every message
// through it. The event handlers do: uploadProgress passes 'files.uploading',
// the listing passes 'files.itemCount', the busy state passes 'files.loading'.
// `_run` does not. It is the wrapper every button action goes through, and both
// of its messages - the success confirmation and the error - are written with no
// key at all, which the panel documents as "always shown".
//
// That is right for the error: "the transfer failed" must beat "the transfer is
// at 40%", and the StatusLine tests pin it. It is wrong for the success
// confirmation. An operator uploads a file, and while the progress readout is
// still standing the refresh lands and then a rename they did in another
// directory writes "renamed to notes.txt" straight over it - and the readout
// that said the upload finished is gone.
const source = await import('node:fs/promises').then(fs =>
    fs.readFile(new URL('../js/files/file-manager-panel.js', import.meta.url), 'utf8'))

test('every _run message carries a subject so the status line can place it', () => {
    // Both _setStatus calls inside _run must pass a key. Matching the call
    // shape rather than counting occurrences, because the point is that these
    // two specific writes participate in the rule.
    //
    // The success message's key is the caller's, so it arrives as a parameter
    // rather than a literal - what matters is that the third argument exists.
    assert.match(source, /_setStatus\(successMessage,\s*false,\s*successKey\)/,
        'the success confirmation bypasses the status line rule')
    assert.match(source, /_setStatus\(error\.message \|\| String\(error\),\s*true,\s*'files\.error'\)/,
        'the error bypasses the status line rule')
})

test('the entry call sites pass their key, not an empty string', () => {
    // A third argument that is always empty would satisfy the shape above while
    // changing nothing - the same false green as a bare method name.
    for (const key of ['files.created', 'files.renamed', 'files.trashed']) {
        const escaped = key.replace('.', '\\.')
        // The rendered message and its key are consecutive arguments of the same
        // _run call. Asserting the pair rather than a count, because a key passed
        // somewhere else in the file would satisfy a count. The gap is bounded so
        // the two must be adjacent arguments rather than anywhere in the file.
        assert.match(source, new RegExp(`t\\('${escaped}'[\\s\\S]{0,60}?'${escaped}',`),
            `${key} is rendered but its key never reaches the status line`)
    }
})

test('the keys _run uses are ones subjectForKey already understands', async () => {
    const { subjectForKey, StatusSubject } = await import('../js/files/statusLine.js')

    // The three entry confirmations need their own subject: `Other` is accepted
    // over everything, so an entry message mapped there would displace a
    // transfer readout exactly as it did before it carried a key.
    for (const key of ['files.created', 'files.renamed', 'files.trashed']) {
        assert.equal(subjectForKey(key), StatusSubject.Entry,
            `${key} does not map to Entry, so it displaces a transfer readout`)
    }

    // The error stays `Other`, and that is deliberate rather than an oversight:
    // Other is accepted over everything, which is what lets "the transfer
    // failed" beat "the transfer is at 40%".
    assert.equal(subjectForKey('files.error'), StatusSubject.Other,
        'the error no longer displaces a transfer readout')
})

test('an entry confirmation does not displace a transfer readout', async () => {
    const { StatusLine } = await import('../js/files/statusLine.js')
    const line = new StatusLine()

    line.set('report.bin 100%', 'files.uploaded')

    // The rule the entry subject exists for: a rename the operator did elsewhere
    // must not overwrite the confirmation that their upload finished.
    assert.equal(line.accepts('files.renamed'), false,
        'an entry confirmation displaced the transfer readout')
})

test('a second entry confirmation replaces the first', async () => {
    // The subject has to move on eventually, or the line is stuck on the first
    // rename forever. Two entry messages are about different entries, so the
    // newer one is the news.
    const { StatusLine } = await import('../js/files/statusLine.js')
    const line = new StatusLine()

    line.set('renamed to notes.txt', 'files.renamed')
    assert.equal(line.accepts('files.trashed'), true,
        'a second entry confirmation was held back')
})
