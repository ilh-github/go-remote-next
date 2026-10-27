#!/usr/bin/env node
// Runs every browser E2E file and reports one aggregate verdict.
//
// The reason this exists: the suite used to be a list of per-file npm scripts,
// and files that had no script were simply never run. Three audit E2E files went
// red that way and stayed red for days - the unit tests were updated, the browser
// tests were not, and nothing ran them as a set.
//
// So the contract is: a new file in test/e2e/ is picked up without editing this
// script, and a single failing file fails the run. A file that cannot run in this
// environment (a long soak, or one needing a live server) is skipped explicitly
// and reported as skipped rather than silently passing.

import { readdir } from 'node:fs/promises'
import { pathToFileURL } from 'node:url'
import path from 'node:path'
import { spawn } from 'node:child_process'

const here = path.dirname(new URL(import.meta.url).pathname)
const e2eDir = path.join(here, '..', 'test', 'e2e')

// Files that need an environment this runner does not provide. Listed rather than
// guessed at from the filename, so the reason travels with the entry.
const SKIP = new Map([
    // The multi-viewer soak is a scheduled nightly run measured in minutes, not
    // something a pre-push gate should start.
    ['wall-soak', 'long-running soak; run explicitly with npm run test:e2e:wall-soak'],
])

// A file that needs a production build before it can load the app.
const NEEDS_BUILD = new Set(['selfhost'])

function parseArgs(argv) {
    const args = { filter: '', list: false, skipSoak: true }
    for (let i = 0; i < argv.length; i += 1) {
        const arg = argv[i]
        if (arg === '--list') args.list = true
        else if (arg === '--filter') { args.filter = argv[i + 1] || ''; i += 1 }
        else if (arg === '--include-soak') args.skipSoak = false
    }
    return args
}

async function discover() {
    const entries = await readdir(e2eDir)
    return entries
        .filter((name) => name.endsWith('.e2e.mjs'))
        .map((name) => name.replace(/\.e2e\.mjs$/, ''))
        .sort()
}

function runOne(name) {
    const file = path.join(e2eDir, `${name}.e2e.mjs`)
    return new Promise((resolve) => {
        // Each file gets its own process: they launch browsers and start servers,
        // and one crashed process must not take the rest of the run with it.
        const child = spawn(process.execPath, ['--test', file], {
            cwd: path.join(here, '..'),
            stdio: ['ignore', 'pipe', 'pipe'],
        })
        // node --test writes its TAP report to stdout, not stderr. Reading the
        // wrong stream yields an empty string and a report that says "exit code 1"
        // without naming the test that failed - which is the one thing the runner
        // exists to say.
        let stderr = ''
        child.stdout.on('data', (chunk) => { stderr += String(chunk) })
        child.stderr.on('data', (chunk) => { stderr += String(chunk) })
        child.on('error', (error) => {
            resolve({ name, status: 'error', detail: String(error) })
        })
        child.on('close', (code) => {
            // node --test exits non-zero when any test fails, so the exit code is
            // the verdict; the TAP summary on stderr says which test.
            const failed = /^# fail (\d+)/m.exec(stderr)
            const passed = /^# pass (\d+)/m.exec(stderr)
            if (code === 0) {
                resolve({
                    name,
                    status: 'pass',
                    pass: passed ? Number(passed[1]) : 0,
                })
                return
            }
            const firstFailure = /^not ok \d+ - (.+)$/m.exec(stderr)
            resolve({
                name,
                status: 'fail',
                fail: failed ? Number(failed[1]) : 0,
                detail: firstFailure ? firstFailure[1].trim() : `exit code ${code}`,
            })
        })
    })
}

const args = parseArgs(process.argv.slice(2))
const all = await discover()
const selected = args.filter
    ? all.filter((name) => name.includes(args.filter))
    : all

if (args.list) {
    for (const name of selected) console.log(name)
    process.exit(0)
}

if (selected.length === 0) {
    console.error(`no E2E file matches --filter ${args.filter}`)
    process.exit(2)
}

if (NEEDS_BUILD.has('selfhost') && selected.includes('selfhost')) {
    console.log('building the app first: selfhost.e2e.mjs loads the production bundle')
    await new Promise((resolve, reject) => {
        const build = spawn('npm', ['run', 'build'], {
            cwd: path.join(here, '..'),
            stdio: 'inherit',
        })
        build.on('error', reject)
        build.on('close', (code) => (code === 0 ? resolve() : reject(new Error(`build exited ${code}`))))
    })
}

const results = []
for (const name of selected) {
    if (SKIP.has(name) && args.skipSoak) {
        results.push({ name, status: 'skip', detail: SKIP.get(name) })
        console.log(`SKIP ${name}: ${SKIP.get(name)}`)
        continue
    }
    process.stdout.write(`RUN  ${name} ... `)
    const result = await runOne(name)
    results.push(result)
    if (result.status === 'pass') console.log(`ok (${result.pass} passed)`)
    else if (result.status === 'fail') console.log(`FAIL (${result.fail} failed) ${result.detail}`)
    else console.log(`ERROR ${result.detail}`)
}

const counts = results.reduce((acc, result) => {
    acc[result.status] = (acc[result.status] || 0) + 1
    return acc
}, {})

console.log('')
console.log(
    `E2E: ${counts.pass || 0} passed, ${counts.fail || 0} failed, ` +
    `${counts.error || 0} errored, ${counts.skip || 0} skipped, ${results.length} total`,
)

// A skipped file is not a passing one, so the summary has to say so rather than
// letting "all green" cover a suite that mostly did not run.
if (counts.skip) {
    console.log(`skipped: ${results.filter((r) => r.status === 'skip').map((r) => r.name).join(', ')}`)
}

if (counts.fail || counts.error) {
    console.log('')
    for (const result of results) {
        if (result.status === 'fail' || result.status === 'error') {
            console.log(`  ${result.name}: ${result.detail}`)
        }
    }
    process.exit(1)
}
