#!/usr/bin/env node

import { spawn } from 'node:child_process';
import { createServer } from 'node:http';
import { mkdir, readFile, rm, writeFile } from 'node:fs/promises';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const repoRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '../..');
const webRoot = path.join(repoRoot, 'WebClient');
const outDir = path.join(repoRoot, 'docs', 'e2e');
const chromePath = '/Applications/Google Chrome.app/Contents/MacOS/Google Chrome';
const debugPort = 19333;

const mimeTypes = new Map([
    ['.html', 'text/html; charset=utf-8'],
    ['.js', 'text/javascript; charset=utf-8'],
    ['.mjs', 'text/javascript; charset=utf-8'],
    ['.css', 'text/css; charset=utf-8'],
    ['.png', 'image/png'],
    ['.ico', 'image/x-icon'],
]);

function startStaticServer() {
    const server = createServer(async (request, response) => {
        try {
            const url = new URL(request.url, 'http://127.0.0.1');
            const relative = decodeURIComponent(url.pathname).replace(/^\/+/, '') || 'index.html';
            const filePath = path.resolve(webRoot, relative);
            if (!filePath.startsWith(webRoot + path.sep)) {
                response.writeHead(403).end('forbidden');
                return;
            }
            const body = await readFile(filePath);
            response.writeHead(200, {
                'content-type': mimeTypes.get(path.extname(filePath)) || 'application/octet-stream',
                'cache-control': 'no-store',
            });
            response.end(body);
        } catch {
            response.writeHead(404).end('not found');
        }
    });
    return new Promise(resolve => {
        server.listen(0, '127.0.0.1', () => {
            resolve({
                server,
                origin: `http://127.0.0.1:${server.address().port}`,
            });
        });
    });
}

async function waitForJson(url, timeoutMs = 10000) {
    const deadline = Date.now() + timeoutMs;
    while (Date.now() < deadline) {
        try {
            const response = await fetch(url);
            if (response.ok) return response.json();
        } catch {}
        await new Promise(resolve => setTimeout(resolve, 100));
    }
    throw new Error(`timed out waiting for ${url}`);
}

async function connectCdp(webSocketDebuggerUrl) {
    const socket = new WebSocket(webSocketDebuggerUrl);
    await new Promise((resolve, reject) => {
        socket.addEventListener('open', resolve, { once: true });
        socket.addEventListener('error', reject, { once: true });
    });

    let nextId = 1;
    const pending = new Map();
    socket.addEventListener('message', event => {
        const message = JSON.parse(event.data);
        if (!message.id) return;
        const waiter = pending.get(message.id);
        if (!waiter) return;
        pending.delete(message.id);
        if (message.error) waiter.reject(new Error(message.error.message));
        else waiter.resolve(message.result);
    });

    const send = (method, params = {}) => new Promise((resolve, reject) => {
        const id = nextId++;
        pending.set(id, { resolve, reject });
        socket.send(JSON.stringify({ id, method, params }));
    });

    return { socket, send };
}

async function capture(cdp, origin, width, height, screenshotPath) {
    await cdp.send('Emulation.setDeviceMetricsOverride', {
        width,
        height,
        deviceScaleFactor: 1,
        mobile: width <= 768,
    });
    await cdp.send('Page.navigate', {
        url: `${origin}/test/fixtures/session-chat.html`,
    });
    await new Promise(resolve => setTimeout(resolve, 1200));

    const evaluation = await cdp.send('Runtime.evaluate', {
        expression: `(() => {
            const root = document.documentElement;
            const metrics = JSON.parse(root.dataset.chatSmokeMetrics || '{}');
            const empty = document.querySelector('.chat-empty');
            const panel = document.querySelector('.chat-panel');
            return {
                metrics,
                messages: document.querySelectorAll('.chat-message').length,
                emptyHidden: !empty || getComputedStyle(empty).display === 'none',
                panelFits: panel ? panel.getBoundingClientRect().right <= window.innerWidth + 0.5 : false,
                noHorizontalOverflow: root.scrollWidth <= window.innerWidth,
            };
        })()`,
        returnByValue: true,
    });
    const result = evaluation.result.value;
    if (result.metrics.innerWidth !== width
        || result.messages !== 3
        || !result.emptyHidden
        || !result.panelFits
        || !result.noHorizontalOverflow
        || !result.metrics.chatMenuVisible
        || !result.metrics.chatMenuText.includes('(2)')) {
        throw new Error(`chat smoke failed at ${width}x${height}: ${JSON.stringify(result)}`);
    }

    const screenshot = await cdp.send('Page.captureScreenshot', {
        format: 'png',
        fromSurface: true,
    });
    await writeFile(screenshotPath, Buffer.from(screenshot.data, 'base64'));
    console.log(`PASS ${width}x${height}: ${path.relative(repoRoot, screenshotPath)}`);
    return result;
}

const { server, origin } = await startStaticServer();
const userDataDir = `/tmp/quickdesk-chat-cdp-${process.pid}`;
const chrome = spawn(chromePath, [
    '--headless=new',
    '--disable-gpu',
    '--hide-scrollbars',
    '--no-first-run',
    `--remote-debugging-port=${debugPort}`,
    `--user-data-dir=${userDataDir}`,
    'about:blank',
], { stdio: 'ignore' });

try {
    await mkdir(outDir, { recursive: true });
    await waitForJson(`http://127.0.0.1:${debugPort}/json/version`);
    const targetResponse = await fetch(
        `http://127.0.0.1:${debugPort}/json/new?${encodeURIComponent(origin)}`,
        { method: 'PUT' },
    );
    const target = await targetResponse.json();
    const cdp = await connectCdp(target.webSocketDebuggerUrl);
    await cdp.send('Page.enable');
    await cdp.send('Runtime.enable');

    await capture(
        cdp,
        origin,
        1280,
        720,
        path.join(outDir, 'e2e-web-session-chat.png'),
    );
    await capture(
        cdp,
        origin,
        390,
        844,
        path.join(outDir, 'e2e-web-session-chat-mobile.png'),
    );
    cdp.socket.close();
} finally {
    chrome.kill('SIGTERM');
    await new Promise(resolve => {
        if (chrome.exitCode !== null) {
            resolve();
            return;
        }
        chrome.once('exit', resolve);
        setTimeout(resolve, 3000);
    });
    await new Promise(resolve => server.close(resolve));
    try {
        await rm(userDataDir, { recursive: true, force: true, maxRetries: 5, retryDelay: 100 });
    } catch (error) {
        console.warn(`warning: could not remove temporary Chrome profile: ${error.message}`);
    }
}
