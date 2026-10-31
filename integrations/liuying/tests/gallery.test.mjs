import test from 'node:test';
import assert from 'node:assert/strict';
import { mkdtemp, mkdir, writeFile, readFile, rm } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import path from 'node:path';
import { setTimeout as delay } from 'node:timers/promises';
import { Jobs } from '../server/jobs.mjs';
import { Signer, publicJob, parseLink } from '../server/core.mjs';

test('gallery defaults to originals, validates every file and binds signatures to assets', async () => {
  const root = await mkdtemp(path.join(tmpdir(), 'liuying-gallery-'));
  await mkdir(path.join(root, 'worker'));
  await writeFile(path.join(root, 'worker/resolve.py'), `const fs=require('node:fs'),path=require('node:path');
const dir=process.argv[process.argv.indexOf('--directory')+1];
if(process.argv[process.argv.indexOf('--mode')+1]!=='original')process.exit(1);
for(let i=0;i<2;i++)fs.writeFileSync(path.join(dir,'image-00'+i+'.jpg'),'original-'+i);
console.log(JSON.stringify({type:'complete',assets:[0,1].map(i=>({filename:'image-00'+i+'.jpg',kind:'image',compatible:true}))}));`);
  const jobs = new Jobs({ root, dataDir: path.join(root, 'data'), python: process.execPath, ttl: 3600, concurrency: 1, maxMb: 1, maxDuration: 60, workerTimeout: 10 });
  try {
    const job = jobs.create('owner', { url: 'https://www.douyin.com/note/123', platform: 'douyin' });
    for (let i = 0; i < 200 && job.status !== 'ready' && job.status !== 'failed'; i++) await delay(10);
    assert.equal(job.status, 'ready'); assert.equal(job.kind, 'images'); assert.equal(job.mode, 'original');
    assert.equal(job.assets.length, 2); assert.equal(job.bytes, 20);
    const signer = new Signer('test'); const output = publicJob(job, signer, 'https://example.org');
    const first = new URL(output.assets[0].mediaUrl); const second = new URL(output.assets[1].mediaUrl);
    assert.ok(signer.verify(`asset:${job.id}:0:${first.searchParams.get('exp')}`, first.searchParams.get('sig')));
    assert.equal(signer.verify(`asset:${job.id}:1:${first.searchParams.get('exp')}`, first.searchParams.get('sig')), false);
    assert.notEqual(first.searchParams.get('sig'), second.searchParams.get('sig'));
    for (const asset of output.assets) { assert.equal(asset.file, undefined); assert.equal(asset.album, undefined); assert.ok(asset.albumUrl); }
    assert.throws(() => jobs.get(job.id, 'someone-else'), { code: 'NOT_FOUND' });
    jobs.prepareAlbum(job); assert.equal(job.albumStatus, 'ready');
    job.albumStatus = undefined; job.assets[0].compatible = false;
    await writeFile(path.join(root, 'worker/album.py'), `console.log(JSON.stringify({type:'error',code:'ALBUM_FAILED',message:'test conversion failure'}));process.exit(1)`);
    jobs.prepareAlbum(job);
    for (let i = 0; i < 100 && job.albumStatus !== 'failed'; i++) await delay(10);
    assert.equal(job.albumStatus, 'failed'); assert.equal(job.status, 'ready'); assert.equal(job.bytes, 20);
    assert.equal(await readFile(job.assets[0].file, 'utf8'), 'original-0');
    assert.equal(parseLink('https://cdn.example.org/a.webp', ['cdn.example.org']).platform, 'direct');
  } finally { await jobs.close(); await rm(root, { recursive: true, force: true }); }
});
