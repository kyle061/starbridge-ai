import test from 'node:test';
import assert from 'node:assert/strict';
import upload from '../scripts/upload.cjs';

test('upload rejects the observed login failure even when the official CLI exits zero', () => {
  assert.throws(() => upload.verifyUploadResult({ status: 0, stdout: '', stderr: '- preparing\n✖ preparing\n[error] { code: 10, message: "需要重新登录" }' }));
});

test('upload requires a positive CLI acknowledgement and rejects timeouts or incomplete output', () => {
  assert.doesNotThrow(() => upload.verifyUploadResult({ status: 0, stdout: '', stderr: '\u001b[32m✔ upload\u001b[0m\n' }));
  for (const result of [
    { status: 0, stdout: '- Upload\n' },
    { status: 1, stdout: '✔ upload\n' },
    { status: null, error: new Error('timed out'), stdout: '' },
    { status: 0, stdout: '✔ upload\n', stderr: '[error] failure' }
  ]) assert.throws(() => upload.verifyUploadResult(result));
});
