import { access, chmod, readFile, writeFile } from 'node:fs/promises';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { parseEnv } from 'node:util';

export function clientConfig(environment) {
  const mode = environment.DEMO_MODE || 'true';
  if (!['true', 'false'].includes(mode)) throw new Error('DEMO_MODE must be true or false');
  const demoMode = mode === 'true';
  const appid = environment.WX_APP_ID || 'touristappid';
  if (appid !== 'touristappid' && !/^wx[0-9a-f]{16}$/.test(appid)) throw new Error('Invalid WX_APP_ID');
  const url = new URL(environment.PUBLIC_BASE_URL || 'http://127.0.0.1:8787');
  if (!['http:', 'https:'].includes(url.protocol) || url.username || url.password || url.search || url.hash)
    throw new Error('PUBLIC_BASE_URL must be an HTTP(S) URL without credentials, query or fragment');
  if (!demoMode && (url.protocol !== 'https:' || appid === 'touristappid'))
    throw new Error('Real WeChat login requires HTTPS and WX_APP_ID');
  return { appid, apiBaseUrl: url.href.replace(/\/+$/, ''), demoMode };
}

export async function configure(root, environment, force = false) {
  const files = [path.join(root, 'project.config.json'), path.join(root, 'miniprogram/config.js')];
  if (!force) for (const file of files) {
    try { await access(file); } catch (error) { if (error.code === 'ENOENT') continue; throw error; }
    throw new Error('Local client configuration already exists. Use --force to replace it.');
  }
  const config = clientConfig(environment);
  const project = JSON.parse(await readFile(path.join(root, 'project.config.example.json'), 'utf8'));
  project.appid = config.appid;
  await writeFile(files[0], JSON.stringify(project, null, 2) + '\n', { mode: 0o600 });
  await writeFile(files[1], 'module.exports = ' + JSON.stringify({ apiBaseUrl: config.apiBaseUrl, demoMode: config.demoMode }, null, 2) + ';\n', { mode: 0o600 });
  for (const file of files) await chmod(file, 0o600);
}

if (process.argv[1] && path.resolve(process.argv[1]) === fileURLToPath(import.meta.url)) {
  const root = path.resolve(fileURLToPath(new URL('..', import.meta.url)));
  let local = {};
  try { local = parseEnv(await readFile(path.join(root, '.env'), 'utf8')); }
  catch (error) { if (error.code !== 'ENOENT') throw error; }
  await configure(root, { ...local, ...process.env }, process.argv.includes('--force'));
  console.log('Generated ignored local client configuration. Server secrets are never copied to the client.');
}
