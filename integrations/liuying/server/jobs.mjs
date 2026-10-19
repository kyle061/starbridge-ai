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
  create(owner, link, mode) {
    if (this.closed) throw new AppError('UNAVAILABLE', '服务正在关闭', 503);
    if ([...this.items.values()].filter(j => j.owner === owner && ['queued', 'processing'].includes(j.status)).length >= 2)
      throw new AppError('BUSY', '你已有两个进行中的任务，请完成后重试', 429);
    if (this.queue.length >= 30 || this.items.size >= 500) throw new AppError('BUSY', '当前任务较多，请稍后重试', 503);
    const createdAt = Date.now();
    const job = { id: randomUUID(), owner, ...link, mode, status: 'queued', progress: 0, title: '正在获取视频', createdAt, expiresAt: createdAt + this.config.ttl * 1000 };
    this.items.set(job.id, job); this.queue.push(job); this.pump(); return job;
  }
  get(id, owner) {
    const job = this.items.get(id);
    if (!job || (owner && job.owner !== owner)) throw new AppError('NOT_FOUND', '记录不存在或已过期', 404);
    if (job.expiresAt <= Date.now()) throw new AppError('EXPIRED', '视频已过期，请重新解析', 410);
    return job;
  }
  async demo(owner) {
    const source = path.join(this.config.root, 'web/assets/demo.mp4');
    let size;
    try { size = (await stat(source)).size; } catch { throw new AppError('NO_DEMO', '示例视频尚未安装，请先完成资源准备', 503); }
    if (this.items.size >= 500) throw new AppError('BUSY', '当前任务较多，请稍后重试', 503);
    const id = randomUUID(); const dir = path.join(this.config.dataDir, id);
    await mkdir(dir, { recursive: true }); await copyFile(source, path.join(dir, 'video.mp4'));
    const job = { id, owner, platform: 'demo', platformName: '公开示例', title: '风吹花开', mode: 'compatible', status: 'ready', progress: 100, bytes: size, ext: 'mp4', compatible: true, duration: 5, isDemo: true, createdAt: Date.now(), expiresAt: Date.now() + this.config.ttl * 1000, file: path.join(dir, 'video.mp4') };
    this.items.set(id, job); return job;
  }
  cancel(id, owner) {
    const job = this.items.get(id);
    if (!job || job.owner !== owner) throw new AppError('NOT_FOUND', '记录不存在或已过期', 404);
    job.status = 'cancelled'; if (job.child) stopWorker(job.child, 'SIGTERM');
    if (!job.child) rm(path.join(this.config.dataDir, job.id), { recursive: true, force: true }).catch(() => {});
    this.queue = this.queue.filter(j => j.id !== id); this.items.delete(id);
  }
  pump() {
    while (!this.closed && this.running < this.config.concurrency && this.queue.length) {
      const job = this.queue.shift();
      this.running++;
      this.run(job).catch(async () => {
        if (job.status === 'cancelled') return;
        job.status = 'failed'; job.error = { code: 'WORKER_ERROR', message: '视频任务启动失败，请稍后重试' };
        await rm(path.join(this.config.dataDir, job.id), { recursive: true, force: true });
      }).finally(() => { this.running--; this.pump(); });
    }
  }
  async run(job) {
    const dir = path.join(this.config.dataDir, job.id);
    await mkdir(dir, { recursive: true });
    const disk = await statfs(this.config.dataDir);
    if (disk.bavail * disk.bsize < (this.config.maxMb * this.config.concurrency * 2.2 + 512) * 1048576) {
      job.status = 'failed'; job.error = { code: 'STORAGE_LOW', message: '服务器临时空间不足，请稍后重试' };
      await rm(dir, { recursive: true, force: true }); return;
    }
    job.status = 'processing'; job.phase = 'extracting';
    const args = [path.join(this.config.root, 'worker/resolve.py'), '--url', job.url, '--platform', job.platform, '--mode', job.mode, '--directory', dir, '--max-mb', String(this.config.maxMb), '--max-duration', String(this.config.maxDuration)];
    if (this.config.cookiesFile) args.push('--cookies', this.config.cookiesFile);
    const child = spawn(this.config.python, args, { detached: process.platform !== 'win32', stdio: ['ignore', 'pipe', 'pipe'], env: { ...process.env, HTTP_PROXY: '', HTTPS_PROXY: '', ALL_PROXY: '', NO_PROXY: '*' } });
    job.child = child;
    let buffer = ''; let result; let failure;
    const timeout = setTimeout(() => { failure = { code: 'TIMEOUT', message: '解析超时，请换一个链接或稍后重试' }; stopWorker(child, 'SIGKILL'); }, this.config.workerTimeout * 1000);
    const watchdog = setInterval(async () => {
      try {
        const { readdir } = await import('node:fs/promises');
        const sizes = await Promise.all((await readdir(dir)).map(f => stat(path.join(dir, f)).then(s => s.size).catch(() => 0)));
        if (sizes.reduce((a, b) => a + b, 0) > this.config.maxMb * 1024 * 1024 * 2.2) {
          failure = { code: 'FILE_TOO_LARGE', message: '视频超出大小限制' }; stopWorker(child, 'SIGKILL');
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
          if (event.type === 'progress') job.progress = Math.min(96, Math.max(job.progress, Number(event.value) || 0));
          if (event.type === 'phase' && ['downloading', 'transcoding'].includes(event.value)) job.phase = event.value;
          if (event.type === 'metadata') { job.title = String(event.title || '视频').slice(0, 150); job.duration = event.duration; }
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
      const file = path.resolve(dir, result.filename);
      if (path.dirname(file) !== dir || !/^video\.[a-z0-9]+$/.test(path.basename(file))) throw new Error('path');
      const bytes = (await stat(file)).size;
      if (bytes <= 0 || bytes > this.config.maxMb * 1024 * 1024) { failure = { code: 'FILE_TOO_LARGE', message: '视频超出大小限制' }; throw new Error('size'); }
      Object.assign(job, { file, bytes, ext: result.ext, compatible: result.compatible === true, transcoded: result.transcoded === true, status: 'ready', progress: 100 });
    } catch {
      job.status = 'failed'; job.error = { code: failure?.code || 'RESOLVE_FAILED', message: failure?.message || '暂时无法获取该视频，请稍后重试' };
      await rm(dir, { recursive: true, force: true });
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
