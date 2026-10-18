import test from 'node:test';
import assert from 'node:assert/strict';
import { parseLink, parseRange, Signer, publicJob, RateLimiter } from '../server/core.mjs';

test('extracts supported share URLs and rejects host spoofing and unsafe schemes', () => {
  assert.equal(parseLink('复制这段 https://v.douyin.com/abc/ 打开抖音').platform, 'douyin');
  assert.equal(parseLink('看看 https://youtu.be/abc123。').url, 'https://youtu.be/abc123');
  for (const value of ['http://youtube.com/watch?v=1', 'https://youtube.com.evil.test/1', 'https://youtube.com@127.0.0.1/a', 'https://127.0.0.1/a.mp4', 'https://youtube.com:444/1', 'file:///etc/passwd', 'https://evil.test/a.mp4']) assert.throws(() => parseLink(value));
  assert.equal(parseLink('https://media.example.org/v.mp4', ['media.example.org']).platform, 'direct');
  assert.throws(() => parseLink('https://child.media.example.org/v.mp4', ['media.example.org']));
});
test('session signatures reject tampering and expired tokens', () => {
  const signer = new Signer('test-secret'); const { token } = signer.session('user-a', 1000);
  assert.equal(signer.authenticate('Bearer ' + token, 1001), 'user-a');
  assert.throws(() => signer.authenticate('Bearer ' + token, 1000 + 8 * 86400000));
  assert.throws(() => signer.authenticate('Bearer ' + token + 'a', 1001));
  assert.throws(() => signer.authenticate('Bearer ' + token + '.extra', 1001));
  assert.equal(signer.verify('media:a:1000', signer.media('a', 1000)), true);
  assert.equal(signer.verify('media:b:1000', signer.media('a', 1000)), false);
});
test('byte ranges support seeking and reject unsatisfiable requests', () => {
  assert.deepEqual(parseRange('bytes=20-39', 100), { start: 20, end: 39 });
  assert.deepEqual(parseRange('bytes=-10', 100), { start: 90, end: 99 });
  assert.deepEqual(parseRange('bytes=90-', 100), { start: 90, end: 99 });
  assert.deepEqual(parseRange('bytes=0-200', 100), { start: 0, end: 99 });
  for (const value of ['bytes=100-', 'bytes=50-40', 'bytes=0-1,2-3', 'bytes=-0', 'bytes=a-b', 'bytes=-']) assert.throws(() => parseRange(value, 100));
});
test('public job responses omit server paths, source URLs and user identifiers', () => {
  const job = publicJob({ id: 'id', owner: 'secret-owner', file: '/private/video.mp4', url: 'https://example.test/secret', status: 'ready', expiresAt: Date.now() + 3600000 }, new Signer('secret'), 'https://example.test/liuying');
  assert.equal(job.owner, undefined); assert.equal(job.file, undefined); assert.equal(job.url, undefined);
  assert.match(job.mediaUrl, /^https:\/\/example.test\/liuying\/media\/id\?/);
});
test('rate limits reset at the configured boundary', () => {
  const limiter = new RateLimiter(); limiter.take('a', 2, 100, 0); limiter.take('a', 2, 100, 1);
  assert.throws(() => limiter.take('a', 2, 100, 2), { code: 'RATE_LIMIT' });
  assert.doesNotThrow(() => limiter.take('a', 2, 100, 100));
});
