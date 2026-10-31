import test from 'node:test';
import assert from 'node:assert/strict';
import vm from 'node:vm';
import { readFile } from 'node:fs/promises';

test('image batch saves originals in order and resumes remaining images after a failure', async () => {
  let page, failSecond = true; const downloaded = [], saved = [];
  const assets = [0, 1, 2].map(id => ({ id: String(id), kind: 'image', bytes: 100, ext: 'jpg', mediaUrl: 'https://example.org/' + id, albumUrl: 'https://example.org/' + id }));
  const job = { id: 'images', status: 'ready', assets, kind: 'images' };
  const wx = {
    downloadFile(options) { downloaded.push(options.url); queueMicrotask(() => options.success({ statusCode: 200, tempFilePath: options.url })); return { onProgressUpdate() {} }; },
    saveImageToPhotosAlbum(options) {
      if (options.filePath.endsWith('/1') && failSecond) { failSecond = false; options.fail({ errMsg: 'storage full' }); }
      else { saved.push(options.filePath); options.success(); }
    },
    getFileSystemManager: () => ({ unlink() {} }), showToast() {}
  };
  const api = { privacy: async () => {}, request: async () => job, display: x => x };
  const context = vm.createContext({ require: name => name.includes('api') ? api : {}, wx, Page: object => { page = object; }, setTimeout, clearTimeout });
  vm.runInContext(await readFile(new URL('../miniprogram/pages/index/index.js', import.meta.url), 'utf8'), context);
  page.data = { ...page.data, job }; page.setData = patch => Object.assign(page.data, patch);
  await page.saveAlbum();
  assert.equal(saved.length, 1); assert.match(page.data.error, /本次已保存 1 \/ 3/);
  await page.saveAlbum();
  assert.deepEqual(saved, assets.map(a => a.mediaUrl));
  assert.equal(downloaded.filter(url => url.endsWith('/0')).length, 1);
  assert.equal(page.data.saving, false); assert.equal(page.data.savedCount, 3);
});
