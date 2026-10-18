import test from 'node:test';
import assert from 'node:assert/strict';
import { spawn } from 'node:child_process';
import { mkdtemp, rm } from 'node:fs/promises';
import os from 'node:os';
import path from 'node:path';
import net from 'node:net';

test('API isolates users, signs downloads, supports seeking and deletes files', async t => {
  const temporary = await mkdtemp(path.join(os.tmpdir(), 'liuying-test-'));
  const probe = net.createServer(); await new Promise(resolve => probe.listen(0, '127.0.0.1', resolve));
  const port = probe.address().port; await new Promise(resolve => probe.close(resolve));
  const base = `http://127.0.0.1:${port}`;
  const child = spawn(process.execPath, ['server/index.mjs'], { cwd: path.resolve(import.meta.dirname, '..'), env: { ...process.env, NODE_ENV: 'test', PORT: String(port), PUBLIC_BASE_URL: base, DATA_DIR: temporary, DEMO_MODE: 'true', SESSION_SECRET: 'integration-secret', WX_APP_ID: '', WX_APP_SECRET: '', PYTHON_BIN: 'nonexistent-liuying-python' }, stdio: ['ignore', 'pipe', 'pipe'] });
  const exited = new Promise(resolve => child.once('exit', resolve));
  t.after(async () => { child.kill('SIGTERM'); await exited; await rm(temporary, { recursive: true, force: true }); });
  for (let i = 0; i < 100; i++) {
    try { if ((await fetch(base + '/api/health')).ok) break; } catch { /* Wait for listening. */ }
    await new Promise(resolve => setTimeout(resolve, 25));
  }
  async function request(route, method = 'GET', data, token) {
    return fetch(base + route, { method, headers: { 'Content-Type': 'application/json', ...(token ? { Authorization: 'Bearer ' + token } : {}) }, ...(data !== undefined ? { body: JSON.stringify(data) } : {}) });
  }
  assert.equal((await request('/api/jobs')).status, 401);
  const a = await (await request('/api/session', 'POST', { demo: true })).json();
  const b = await (await request('/api/session', 'POST', { demo: true })).json();
  assert.equal((await request('/api/session', 'POST', null)).status, 400);
  const denied = await request('/api/jobs', 'POST', { authorized: false, mode: 'original', text: 'https://youtu.be/a' }, a.token);
  assert.equal(denied.status, 400);
  assert.equal((await fetch(base + '/api/demo', { method: 'POST', headers: { Origin: 'https://evil.test', Authorization: 'Bearer ' + a.token } })).status, 403);
  const created = await request('/api/demo', 'POST', {}, a.token); assert.equal(created.status, 201);
  const job = await created.json(); assert.equal(job.status, 'ready'); assert.equal(job.isDemo, true);
  assert.equal((await request('/api/jobs/' + job.id, 'GET', undefined, b.token)).status, 404);
  assert.deepEqual((await (await request('/api/jobs', 'GET', undefined, b.token)).json()).jobs, []);
  const range = await fetch(job.mediaUrl, { headers: { Range: 'bytes=0-15' } }); assert.equal(range.status, 206);
  assert.equal((await range.arrayBuffer()).byteLength, 16); assert.equal(range.headers.get('content-type'), 'video/mp4');
  assert.equal((await fetch(job.mediaUrl, { headers: { Range: 'bytes=99999999-' } })).status, 416);
  assert.equal((await fetch(job.mediaUrl.replace('sig=', 'sig=x'))).status, 403);
  const full = await fetch(job.downloadUrl); assert.match(full.headers.get('content-disposition'), /attachment/); assert.equal((await full.arrayBuffer()).byteLength, job.bytes);
  assert.equal((await request('/api/jobs/' + job.id, 'DELETE', undefined, b.token)).status, 404);
  assert.equal((await request('/api/jobs/' + job.id, 'DELETE', undefined, a.token)).status, 200);
  assert.equal((await fetch(job.mediaUrl)).status, 404);
  assert.equal((await fetch(base + '/assets/../../.env')).status, 404);
  assert.equal((await fetch(base + '/')).status, 200);
});
