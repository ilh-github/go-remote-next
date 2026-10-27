// QD-909: the leaf, and only the leaf, of a name a host handed back.
//
// The server refuses a filename that carries a path, because a path is
// information about the controlled machine rather than about the transfer:
// `/Users/someone/.ssh/id_rsa` tells every later reader of the audit table where
// that user keeps their keys, and that table outlives the session.
//
// Reducing it here as well is not belt-and-braces. The host is where the full
// path exists, the client is the first place it can be dropped, and the audit
// trail's whole property is that the path never got written down at all. A
// server rejection turns a leak into a 400; this keeps it from ever being a
// candidate.

/**
 * @param {string} name  whatever the host reported
 * @returns {string} the final path segment, or '' when there is no leaf
 */
export function auditLeafName(name) {
    if (typeof name !== 'string' || name === '') return ''
    // Both separator families: the host may be either kind of machine, and the
    // concern is identical on both.
    const lastSeparator = Math.max(name.lastIndexOf('/'), name.lastIndexOf('\\'))
    const leaf = lastSeparator === -1 ? name : name.slice(lastSeparator + 1)
    // `C:` is a path with no separator in it, so the slice above leaves it
    // whole. A bare drive letter names nothing.
    if (/^[A-Za-z]:$/.test(leaf)) return ''
    return leaf.trim()
}
