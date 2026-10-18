import test from 'node:test';
import assert from 'node:assert/strict';
import { mkdtemp, mkdir, writeFile, readFile, rm } from 'node:fs/promises';
import os from 'node:os';
import path from 'node:path';
import { clientConfig, configure } from '../scripts/configure.mjs';

test('client config validates login settings and omits server credentials', () => {
  assert.deepEqual(clientConfig({ WX_APP_SECRET: 'server-only', SESSION_SECRET: 'server-only' }), {
    appid: 'touristappid', apiBaseUrl: 'http://127.0.0.1:8787', demoMode: true
  });
  for (const environment of [
    { DEMO_MODE: 'false' },
    { DEMO_MODE: 'false', WX_APP_ID: 'wx0000000000000000', PUBLIC_BASE_URL: 'http://example.com' },
    { PUBLIC_BASE_URL: 'https://user:password@example.com' },
    { PUBLIC_BASE_URL: 'https://example.com/?token=private' }
  ]) assert.throws(() => clientConfig(environment));
});

test('configuration generation preserves existing files and never exports server secrets', async () => {
  const root = await mkdtemp(path.join(os.tmpdir(), 'liuying-config-'));
  try {
    await mkdir(path.join(root, 'miniprogram'));
    await writeFile(path.join(root, 'project.config.example.json'), '{"appid":"touristappid"}');
    const env = { DEMO_MODE: 'false', WX_APP_ID: 'wx0000000000000000', PUBLIC_BASE_URL: 'https://video.example.com/liuying', WX_APP_SECRET: 'private-server-secret' };
    await configure(root, env);
    const project = await readFile(path.join(root, 'project.config.json'), 'utf8');
    const client = await readFile(path.join(root, 'miniprogram/config.js'), 'utf8');
    assert.equal(JSON.parse(project).appid, env.WX_APP_ID);
    assert.equal(client.includes(env.WX_APP_SECRET), false);
    await assert.rejects(configure(root, {}), /already exists/);
    assert.equal(await readFile(path.join(root, 'miniprogram/config.js'), 'utf8'), client);
  } finally { await rm(root, { recursive: true, force: true }); }
});
