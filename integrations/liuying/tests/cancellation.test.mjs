import test from 'node:test';
import assert from 'node:assert/strict';
import { mkdtemp, mkdir, readFile, writeFile, rm } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import path from 'node:path';
import { setTimeout as delay } from 'node:timers/promises';
import { Jobs } from '../server/jobs.mjs';

test('cancelling a worker also stops its encoder subprocess', { skip: process.platform === 'win32' }, async () => {
  const root = await mkdtemp(path.join(tmpdir(), 'liuying-cancel-'));
  const marker = path.join(root, 'encoder-ticks');
  await mkdir(path.join(root, 'worker'));
  const encoder = `const fs=require('node:fs');setInterval(()=>fs.appendFileSync(${JSON.stringify(marker)},'x'),20);`;
  await writeFile(path.join(root, 'worker/resolve.py'), `require('node:child_process').spawn(process.execPath,['-e',${JSON.stringify(encoder)}],{stdio:'inherit'});setInterval(()=>{},1000);`);
  const jobs = new Jobs({ root, dataDir: path.join(root, 'data'), python: process.execPath, ttl: 3600, concurrency: 1, maxMb: 1, maxDuration: 60, workerTimeout: 10 });
  try {
    const job = jobs.create('owner', { url: 'https://example.com/video', platform: 'direct' }, 'compatible');
    for (let i = 0; i < 100; i++) {
      try { if ((await readFile(marker)).length) break; } catch { /* Wait for encoder startup. */ }
      await delay(20);
    }
    assert.ok((await readFile(marker)).length);
    jobs.cancel(job.id, 'owner');
    for (let i = 0; i < 100 && jobs.running; i++) await delay(20);
    assert.equal(jobs.running, 0);
    const stopped = await readFile(marker, 'utf8');
    await delay(100);
    assert.equal(await readFile(marker, 'utf8'), stopped);
  } finally { await jobs.close(); await rm(root, { recursive: true, force: true }); }
});
