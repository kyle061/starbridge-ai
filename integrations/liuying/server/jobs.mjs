import { spawn } from 'node:child_process';
import { mkdir, rm, stat, statfs, copyFile } from 'node:fs/promises';
import path from 'node:path';
import { randomUUID } from 'node:crypto';
import { AppError } from './core.mjs';

function stopWorker(child, signal) {
  try { if (process.platform === 'win32') child.kill(signal); else process.kill(-child.pid, signal); }
  catch (error) { if (error.code !== 'ESRCH') child.kill(signal); }
}

export class Jobs {
  constructor(config) {
    this.config = config; this.items = new Map(); this.queue = []; this.running = 0; this.closed = false;
    this.timer = setInterval(() => this.cleanup().catch(() => {}), 60000).unref();
  }
  create(owner, link, mode = 'original') {
    if (this.closed) throw new AppError('UNAVAILABLE', '服务正在关闭', 503);
    if ([...this.items.values()].filter(j => j.owner === owner && (['queued', 'processing'].includes(j.status) || ['queued', 'processing'].includes(j.albumStatus))).length >= 2)
      throw new AppError('BUSY', '你已有两个进行中的任务，请完成后重试', 429);
    if (this.queue.length >= 30 || this.items.size >= 500) throw new AppError('BUSY', '当前任务较多，请稍后重试', 503);
    const createdAt = Date.now();
    const job = { id: randomUUID(), owner, ...link, mode, status: 'queued', progress: 0, title: '正在获取内容', createdAt, expiresAt: createdAt + this.config.ttl * 1000 };
    this.items.set(job.id, job); this.queue.push({ job, album: false }); this.pump(); return job;
  }
  get(id, owner) {
    const job = this.items.get(id);
    if (!job || (owner && job.owner !== owner)) throw new AppError('NOT_FOUND', '记录不存在或已过期', 404);
    if (job.expiresAt <= Date.now()) throw new AppError('EXPIRED', '文件已过期，请重新解析', 410);
    return job;
  }
  async demo(owner, kind = 'video') {
    const source = path.join(this.config.root, 'web/assets/demo.mp4');
    let size;
    try { size = (await stat(source)).size; } catch { throw new AppError('NO_DEMO', '示例视频尚未安装，请先完成资源准备', 503); }
    if (this.items.size >= 500) throw new AppError('BUSY', '当前任务较多，请稍后重试', 503);
    const id = randomUUID(); const dir = path.join(this.config.dataDir, id);
    await mkdir(dir, { recursive: true }); await copyFile(source, path.join(dir, 'video.mp4'));
    const job = { id, owner, platform: 'demo', platformName: '公开示例', title: '风吹花开', mode: 'compatible', status: 'ready', progress: 100, bytes: size, ext: 'mp4', compatible: true, duration: 5, isDemo: true, createdAt: Date.now(), expiresAt: Date.now() + this.config.ttl * 1000, file: path.join(dir, 'video.mp4') };
    job.kind = 'video'; job.assets = [{ id: '0', kind: 'video', file: job.file, bytes: size, ext: 'mp4', compatible: true }];
    if (kind === 'images') {
      const sources = ['demo.jpg', 'demo-detail.jpg']; job.assets = [];
      for (const [index, source] of sources.entries()) {
        const file = path.join(dir, `image-00${index}.jpg`);
        await copyFile(path.join(this.config.root, 'web/assets', source), file);
        job.assets.push({ id: String(index), kind: 'image', file, bytes: (await stat(file)).size, ext: 'jpg', compatible: true });
      }
      await rm(job.file); job.file = job.assets[0].file;
      job.kind = 'images'; job.title = '花开时刻 · CC0 示例原图'; job.duration = undefined;
      job.bytes = job.assets.reduce((n, a) => n + a.bytes, 0); job.ext = 'jpg'; job.mode = 'original';
      const archive = path.join(dir, 'originals.zip');
      await copyFile(path.join(this.config.root, 'web/assets/demo-images.zip'), archive);
      job.archive = { file: archive, bytes: (await stat(archive)).size };
    }
    this.items.set(id, job); return job;
  }
  cancel(id, owner) {
    const job = this.items.get(id);
    if (!job || job.owner !== owner) throw new AppError('NOT_FOUND', '记录不存在或已过期', 404);
    job.status = 'cancelled'; if (job.child) stopWorker(job.child, 'SIGTERM');
    if (!job.child) rm(path.join(this.config.dataDir, job.id), { recursive: true, force: true }).catch(() => {});
    this.queue = this.queue.filter(task => task.job.id !== id); this.items.delete(id);
  }
  prepareAlbum(job) {
    if (job.status !== 'ready') throw new AppError('NOT_READY', '文件尚未就绪', 409);
    if (job.assets.every(a => a.compatible && a.bytes <= 190 * 1048576)) { job.albumStatus = 'ready'; return; }
    if (['queued', 'processing', 'ready'].includes(job.albumStatus)) return;
    if (this.closed || this.queue.length >= 30) throw new AppError('BUSY', '当前任务较多，请稍后重试', 503);
    job.albumStatus = 'queued'; delete job.albumError;
    this.queue.push({ job, album: true }); this.pump();
  }
  pump() {
    while (!this.closed && this.running < this.config.concurrency && this.queue.length) {
      const { job, album } = this.queue.shift();
      this.running++;
      this.run(job, album).catch(async () => {
        if (job.status === 'cancelled') return;
        const error = { code: 'WORKER_ERROR', message: '解析任务启动失败，请稍后重试' };
        if (album) { job.albumStatus = 'failed'; job.albumError = error; }
        else { job.status = 'failed'; job.error = error; await rm(path.join(this.config.dataDir, job.id), { recursive: true, force: true }); }
      }).finally(() => { this.running--; this.pump(); });
    }
  }
  async run(job, album = false) {
    const dir = path.join(this.config.dataDir, job.id);
    await mkdir(dir, { recursive: true });
    const disk = await statfs(this.config.dataDir);
    if (disk.bavail * disk.bsize < (this.config.maxMb * this.config.concurrency * 2.2 + 512) * 1048576) {
      const error = { code: 'STORAGE_LOW', message: '服务器临时空间不足，请稍后重试' };
      if (album) { job.albumStatus = 'failed'; job.albumError = error; }
      else { job.status = 'failed'; job.error = error; await rm(dir, { recursive: true, force: true }); }
      return;
    }
    if (album) job.albumStatus = 'processing';
    else { job.status = 'processing'; job.phase = 'extracting'; }
    const args = [path.join(this.config.root, 'worker/resolve.py'), '--url', job.url, '--platform', job.platform, '--mode', job.mode, '--directory', dir, '--max-mb', String(this.config.maxMb), '--max-duration', String(this.config.maxDuration)];
    if (album) args.splice(0, args.length, path.join(this.config.root, 'worker/album.py'), '--files', JSON.stringify(job.assets.map(a => ({ file: a.file, compatible: a.compatible, kind: a.kind }))), '--max-mb', String(this.config.maxMb));
    if (!album && this.config.cookiesFile) args.push('--cookies', this.config.cookiesFile);
    const child = spawn(this.config.python, args, { detached: process.platform !== 'win32', stdio: ['ignore', 'pipe', 'pipe'], env: { ...process.env, HTTP_PROXY: '', HTTPS_PROXY: '', ALL_PROXY: '', NO_PROXY: '*' } });
    job.child = child;
    let buffer = ''; let result; let failure;
    const timeout = setTimeout(() => { failure = { code: 'TIMEOUT', message: '解析超时，请换一个链接或稍后重试' }; stopWorker(child, 'SIGKILL'); }, this.config.workerTimeout * 1000);
    const watchdog = setInterval(async () => {
      try {
        const { readdir } = await import('node:fs/promises');
        const sizes = await Promise.all((await readdir(dir)).map(f => stat(path.join(dir, f)).then(s => s.size).catch(() => 0)));
        if (sizes.reduce((a, b) => a + b, 0) > this.config.maxMb * 1024 * 1024 * 2.2) {
          failure = { code: 'FILE_TOO_LARGE', message: '文件超出大小限制' }; stopWorker(child, 'SIGKILL');
        }
      } catch { /* Directory may be removed during cancellation. */ }
    }, 1500);
    child.stdout.on('data', data => {
      buffer += data.toString();
      if (buffer.length > 262144) { failure = { code: 'WORKER_ERROR', message: '解析返回异常' }; stopWorker(child, 'SIGKILL'); return; }
      let index;
      while ((index = buffer.indexOf('\n')) >= 0) {
        const line = buffer.slice(0, index); buffer = buffer.slice(index + 1);
        try {
          const event = JSON.parse(line);
          if (!album && event.type === 'progress') job.progress = Math.min(96, Math.max(job.progress, Number(event.value) || 0));
          if (!album && event.type === 'phase' && ['downloading', 'transcoding'].includes(event.value)) job.phase = event.value;
          if (!album && event.type === 'metadata') { job.title = String(event.title || '视频').slice(0, 150); job.duration = event.duration; }
          if (event.type === 'complete') result = event;
          if (event.type === 'error') failure = event;
        } catch { /* Ignore extractor diagnostic lines. */ }
      }
    });
    child.stderr.on('data', () => {});
    const exitCode = await new Promise(resolve => {
      child.on('error', () => { failure = { code: 'DEPENDENCY_MISSING', message: '解析环境未就绪，请联系管理员安装 Python 和 yt-dlp' }; resolve(-1); });
      child.on('close', resolve);
    });
    clearTimeout(timeout); clearInterval(watchdog); delete job.child;
    if (job.status === 'cancelled') { await rm(dir, { recursive: true, force: true }); return; }
    try {
      if (exitCode !== 0 || !result) throw new Error('worker');
      const maxBytes = this.config.maxMb * 1048576;
      const validate = async (name, pattern, cap = maxBytes) => {
        if (typeof name !== 'string' || !pattern.test(name)) throw new Error('path');
        const file = path.resolve(dir, name);
        if (path.dirname(file) !== dir) throw new Error('path');
        const info = await stat(file);
        if (!info.isFile() || info.size <= 0 || info.size > cap) throw new Error('size');
        return { file, bytes: info.size, ext: path.extname(file).slice(1) };
      };
      if (album) {
        if (!Array.isArray(result.assets) || result.assets.length !== job.assets.length) throw new Error('manifest');
        const copies = await Promise.all(result.assets.map((a, i) => validate(a.filename, /^(?:compatible\.mp4|album-\d{3}\.jpg|image-\d{3}\.(?:jpg|png|gif)|video\.(?:mp4|mov|mkv|webm))$/, 190 * 1048576)));
        copies.forEach((copy, i) => { job.assets[i].album = copy; });
        job.albumStatus = 'ready';
        return;
      }
      const manifest = result.assets || [{ filename: result.filename, kind: 'video', compatible: result.compatible }];
      if (!Array.isArray(manifest) || !manifest.length || manifest.length > 100) throw new Error('manifest');
      const assets = await Promise.all(manifest.map(async (a, index) => {
        const allowed = a.kind === 'image' ? /^image-\d{3}\.(?:jpg|png|webp|gif|avif)$/ : /^video\.(?:mp4|mov|mkv|webm)$/;
        return { ...await validate(a.filename, allowed), id: String(index), kind: a.kind, compatible: a.compatible === true };
      }));
      if (assets.reduce((n, a) => n + a.bytes, 0) > maxBytes) throw new Error('size');
      let archive;
      if (result.archive) archive = await validate(result.archive, /^originals\.zip$/, maxBytes + 65536);
      Object.assign(job, { assets, archive, kind: assets.every(a => a.kind === 'image') ? 'images' : 'video', file: assets[0].file,
        bytes: assets.reduce((n, a) => n + a.bytes, 0), ext: assets[0].ext, compatible: assets.every(a => a.compatible),
        transcoded: result.transcoded === true, status: 'ready', progress: 100 });
    } catch {
      const error = { code: failure?.code || 'RESOLVE_FAILED', message: failure?.message || '暂时无法获取该内容，请稍后重试' };
      if (album) { job.albumStatus = 'failed'; job.albumError = error; }
      else { job.status = 'failed'; job.error = error; await rm(dir, { recursive: true, force: true }); }
    }
  }

  async cleanup() {
    for (const [id, job] of this.items) if (job.expiresAt <= Date.now()) this.cancel(id, job.owner);
  }
  async close() {
    this.closed = true; clearInterval(this.timer); this.queue = [];
    const children = [...this.items.values()].map(j => j.child).filter(Boolean);
    const exits = children.map(c => new Promise(resolve => c.once('close', resolve)));
    children.forEach(c => stopWorker(c, 'SIGKILL')); await Promise.all(exits);
  }
}
