import { readFile, readdir, access } from 'node:fs/promises';
import { execFileSync } from 'node:child_process';
import path from 'node:path';
const root = process.cwd(); let files = 0;
async function walk(dir) {
  for (const entry of await readdir(dir, { withFileTypes: true })) {
    if (['node_modules', 'data', '.venv', '__pycache__'].includes(entry.name)) continue;
    const file = path.join(dir, entry.name);
    if (entry.isDirectory()) { await walk(file); continue; }
    if (/\.(js|mjs|cjs)$/.test(file)) execFileSync(process.execPath, ['--check', file]);
    if (file.endsWith('.json')) JSON.parse(await readFile(file, 'utf8'));
    if (/\.(html|wxml)$/.test(file)) {
      const source = await readFile(file, 'utf8');
      for (const [, asset] of source.matchAll(/(?:src|href)="([^"{}]+)"/g)) {
        if (asset.startsWith('https:') || asset.startsWith('#') || asset === './') continue;
        const location = asset.startsWith('/assets/') ? path.join(root, 'miniprogram', asset.slice(1)) : path.resolve(path.dirname(file), asset);
        await access(location);
      }
    }
    files++;
  }
}
await walk(path.join(root, 'server')); await walk(path.join(root, 'miniprogram')); await walk(path.join(root, 'web')); await walk(path.join(root, 'scripts'));
const app = JSON.parse(await readFile('miniprogram/app.json', 'utf8'));
for (const page of app.pages) for (const ext of ['js', 'json', 'wxml', 'wxss']) await access(path.join('miniprogram', page + '.' + ext));
for (const item of app.tabBar.list) for (const key of ['iconPath', 'selectedIconPath']) await access(path.join('miniprogram', item[key]));
console.log(`Checked ${files} files: JavaScript syntax, JSON, page routes and asset references`);
