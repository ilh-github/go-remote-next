import { cpSync, existsSync } from 'node:fs';
import { dirname, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';

const root = resolve(dirname(fileURLToPath(import.meta.url)), '..');
const dist = resolve(root, 'dist');

if (!existsSync(dist)) {
  console.error('dist/ does not exist — run vite build first');
  process.exit(1);
}

cpSync(resolve(root, 'remote.html'), resolve(dist, 'remote.html'));
cpSync(resolve(root, 'js'), resolve(dist, 'js'), { recursive: true });

console.log('Copied remote.html + js/ to dist/');
