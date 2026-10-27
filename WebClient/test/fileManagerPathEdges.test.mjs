import assert from 'node:assert/strict';
import test from 'node:test';

import {
    basename,
    joinRemotePath,
    parentPath,
    separatorForPath,
} from '../js/files/file-manager-model.js';

// The path helpers decide where a rename or a delete lands, which is where the
// nested-trash bug lived: a path whose parent is computed wrongly gets written
// one level too deep, and nothing downstream notices.
//
// The existing tests cover the representative cases. These are the edges: a
// trailing separator, a drive-relative path with no directory, and a name that
// itself looks like a separator.

test('a trailing separator does not produce an empty basename', () => {
    // The delete destination is parentPath + basename, so a basename that comes
    // back empty silently moves the item into its own parent.
    assert.equal(basename('/home/user/docs/'), 'docs');
    assert.equal(basename('C:\\Users\\tim\\docs\\'), 'docs');
});

test('a trailing separator does not change the parent', () => {
    assert.equal(parentPath('/home/user/docs/'), '/home/user');
    assert.equal(parentPath('C:\\Users\\tim\\docs\\'), 'C:\\Users\\tim');
});

test('repeated separators collapse rather than yielding an empty segment', () => {
    assert.equal(parentPath('/home//user///docs'), '/home/user');
    assert.equal(basename('/home/user/docs//'), 'docs');
});

// A path with no separator at all has no parent. Returning '/' here would put a
// rename's destination at the filesystem root instead of refusing.
test('a bare name has no parent', () => {
    assert.equal(parentPath('docs'), '');
    assert.equal(basename('docs'), 'docs');
});

test('a drive-relative path stays inside its drive', () => {
    // `C:docs` is drive-relative, not rooted at C:\. Its parent is the drive
    // root, and slicing at the last separator would give '' instead.
    assert.equal(parentPath('C:docs'), 'C:\\');
});

test('the separator is chosen from the path, not from the host default', () => {
    assert.equal(separatorForPath('/home/user'), '/');
    assert.equal(separatorForPath('C:\\Users'), '\\');
    assert.equal(separatorForPath('C:/Users'), '\\');
});

test('joining onto a directory that already ends in a separator does not double it', () => {
    assert.equal(joinRemotePath('/home/user/', 'docs'), '/home/user/docs');
    assert.equal(joinRemotePath('C:\\Users\\', 'tim'), 'C:\\Users\\tim');
});

test('the root is its own parent, so navigating up from it stays put', () => {
    assert.equal(parentPath('/'), '/');
    assert.equal(parentPath('C:\\'), 'C:\\');
    assert.equal(parentPath('C:'), 'C:\\');
});
