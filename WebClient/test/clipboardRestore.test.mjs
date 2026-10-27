import assert from 'node:assert/strict'
import test from 'node:test'

// QD-906, clipboard half: the visibility release tears the input handlers down
// on every way an operator stops paying attention - tab switch, application
// switch, minimise, screen lock. That is right for the keyboard, because the
// host is left holding whatever was down and has no way to notice this
// controller is gone.
//
// It is wrong for the clipboard. `_cleanupInputHandlers` destroys the
// ClipboardHandler along with everything else, and the only paths that rebuild
// it are a new media track, the control-taken overlay, and wall arming. None of
// those happen when the operator switches back to the tab. So the first blur
// leaves clipboard sync dead for the rest of the session, and nothing on screen
// says so - the operator copies something on the remote machine, nothing
// arrives locally, and there is no error anywhere.
const source = await import('node:fs/promises').then(fs =>
    fs.readFile(new URL('../js/remote-main.js', import.meta.url), 'utf8'))

test('the visibility release restores clipboard sync when the operator comes back', () => {
    // Strip imports so a bare name cannot be satisfied by the import line.
    const body = source.replace(/^import .*$/gm, '')

    // The release callback must rebuild what it tore down. Releasing the held
    // keys is the whole point of the teardown; the clipboard is collateral.
    // The call, not the definition. A method that exists and is never called is
    // exactly the defect this guard exists to catch, and matching the bare name
    // passes on the definition alone.
    assert.match(body, /watchForReturn\(window,\s*\(\)\s*=>\s*\{\s*this\._restoreInputHandlers\(\)/,
        'the visibility release never restores what it tore down, so clipboard sync stays dead after the first blur')
})

test('the restore path rebuilds the clipboard handler, not only the keyboard', () => {
    const body = source.replace(/^import .*$/gm, '')

    // The clipboard handler is created in _setupInputHandlers, so a restore that
    // calls anything narrower would leave it out.
    assert.match(body, /_restoreInputHandlers\(\)\s*\{[\s\S]*?_setupInputHandlers\(\)/,
        'the restore does not go through the path that builds the clipboard handler')
})
