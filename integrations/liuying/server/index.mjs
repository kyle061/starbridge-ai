import http from 'node:http';
import { readFile, stat, mkdir, rm, readdir } from 'node:fs/promises';
import { createReadStream } from 'node:fs';
import { pipeline } from 'node:stream/promises';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { randomBytes, createHash } from 'node:crypto';
import { execFile } from 'node:child_process';
import { promisify } from 'node:util';
import { AppError, PLATFORMS, parseLink, Signer, publicJob, RateLimiter, parseRange } from './core.mjs';
import { Jobs } from './jobs.mjs';

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const production = process.env.NODE_ENV === 'production';
const demoMode = process.env.DEMO_MODE !== 'false' && !production;
const secret = process.env.SESSION_SECRET || (production ? '' : randomBytes(32).toString('hex'));
if (!secret || (production && secret.length < 32)) throw new Error('Production requires SESSION_SECRET with at least 32 characters');
if (production && (!process.env.WX_APP_ID || !process.env.WX_APP_SECRET)) throw new Error('Production requires WX_APP_ID and WX_APP_SECRET');
const baseUrl = process.env.PUBLIC_BASE_URL || 'http://127.0.0.1:8787';
if (production && new URL(baseUrl).protocol !== 'https:') throw new Error('Production PUBLIC_BASE_URL must use HTTPS');
function numberEnv(name, fallback, max) {
  const value = Number(process.env[name] || fallback);
  if (!Number.isInteger(value) || value < 1 || value > max) throw new Error(`Invalid ${name}`);
  return value;
}
const config = {
  root, dataDir: path.resolve(root, process.env.DATA_DIR || 'data'), python: process.env.PYTHON_BIN || 'python3',
  concurrency: numberEnv('MAX_CONCURRENT_JOBS', 2, 8), ttl: numberEnv('JOB_TTL_SECONDS', 3600, 86400),
  maxMb: numberEnv('MAX_FILE_MB', 200, 1024), maxDuration: numberEnv('MAX_DURATION_SECONDS', 1200, 7200),
  workerTimeout: numberEnv('WORKER_TIMEOUT_SECONDS', 600, 3600), cookiesFile: process.env.COOKIES_FILE
};
await mkdir(config.dataDir, { recursive: true });
// Jobs are intentionally ephemeral; files left by a previous process have no valid signed URLs.
for (const name of await readdir(config.dataDir)) if (/^[a-f0-9-]{36}$/.test(name)) await rm(path.join(config.dataDir, name), { recursive: true, force: true });
const jobs = new Jobs(config); const signer = new Signer(secret); const limiter = new RateLimiter();
let extractor = false;
try { await promisify(execFile)(config.python, ['-c', 'import yt_dlp'], { timeout: 8000 }); extractor = true; } catch { /* Report readiness in health. */ }
const directHosts = (process.env.DIRECT_MEDIA_HOSTS || '').split(',').map(s => s.trim().toLowerCase()).filter(Boolean);
const types = { '.html': 'text/html; charset=utf-8', '.css': 'text/css; charset=utf-8', '.js': 'text/javascript; charset=utf-8', '.svg': 'image/svg+xml', '.png': 'image/png', '.jpg': 'image/jpeg', '.mp4': 'video/mp4', '.webm': 'video/webm', '.mov': 'video/quicktime', '.mkv': 'video/x-matroska' };
const local = req => ['127.0.0.1', '::1', '::ffff:127.0.0.1'].includes(req.socket.remoteAddress);
const send = (res, value, status = 200) => { res.writeHead(status, { 'Content-Type': 'application/json; charset=utf-8', 'Cache-Control': 'no-store' }); res.end(JSON.stringify(value)); };
async function body(req) {
  const chunks = []; let size = 0;
  for await (const chunk of req) {
    size += chunk.length;
    if (size > 16384) throw new AppError('BODY_TOO_LARGE', '请求内容过长', 413);
    chunks.push(chunk);
  }
  try {
    const value = JSON.parse(Buffer.concat(chunks).toString());
    if (!value || typeof value !== 'object' || Array.isArray(value)) throw new Error('object required');
    return value;
  } catch { throw new AppError('INVALID_JSON', '请求格式错误'); }
}
async function fileResponse(req, res, file, attachment) {
  const { size } = await stat(file); let range;
  try { range = parseRange(req.headers.range, size); } catch (error) { res.setHeader('Content-Range', `bytes */${size}`); throw error; }
  const headers = { 'Content-Type': types[path.extname(file)] || 'application/octet-stream', 'Accept-Ranges': 'bytes', 'Cache-Control': 'private, no-store', 'Content-Length': range ? range.end - range.start + 1 : size };
  if (range) headers['Content-Range'] = `bytes ${range.start}-${range.end}/${size}`;
  if (attachment) headers['Content-Disposition'] = `attachment; filename="liuying-${path.basename(path.dirname(file))}${path.extname(file)}"`;
  res.writeHead(range ? 206 : 200, headers);
  if (req.method === 'HEAD') { res.end(); return; }
  await pipeline(createReadStream(file, range || undefined), res);
}

export const server = http.createServer(async (req, res) => {
  res.setHeader('X-Content-Type-Options', 'nosniff'); res.setHeader('Referrer-Policy', 'no-referrer');
  res.setHeader('X-Frame-Options', 'DENY');
  res.setHeader('Content-Security-Policy', "default-src 'self'; script-src 'self'; style-src 'self'; img-src 'self' data:; media-src 'self'; connect-src 'self'; frame-ancestors 'none'; base-uri 'none'; form-action 'self'");
  try {
    const url = new URL(req.url, 'http://localhost');
    if (!['GET', 'HEAD', 'POST', 'DELETE'].includes(req.method)) throw new AppError('METHOD', '请求方式不支持', 405);
    if (req.method === 'POST' || req.method === 'DELETE') {
      const origin = req.headers.origin;
      if (origin && origin !== new URL(baseUrl).origin) throw new AppError('ORIGIN', '请求来源无效', 403);
    }
    if (url.pathname.startsWith('/api/')) {
      limiter.take('ip:' + req.socket.remoteAddress, 500, 60000);
      if (url.pathname === '/api/health' && req.method === 'GET') return send(res, { status: 'ok', extractor, demoMode, maxMb: config.maxMb, maxDuration: config.maxDuration, ttl: config.ttl });
      if (url.pathname === '/api/platforms' && req.method === 'GET') return send(res, { platforms: PLATFORMS.map(({ hosts, ...rest }) => rest), extractor });
      if (url.pathname === '/api/session' && req.method === 'POST') {
        limiter.take('login:' + req.socket.remoteAddress, 20, 60000);
        const input = await body(req);
        if (typeof input.previewKey === 'string') {
          if (!process.env.PREVIEW_ACCESS_KEY || input.previewKey.length > 256 || !signer.verify('preview:' + input.previewKey, signer.sign('preview:' + process.env.PREVIEW_ACCESS_KEY)))
            throw new AppError('PREVIEW_AUTH', '访问码不正确', 401);
          return send(res, signer.session(signer.newOwner()));
        }
        if (input.demo === true) {
          if (!demoMode || !local(req)) throw new AppError('DEMO_DISABLED', '演示登录仅允许本机开发环境使用', 403);
          return send(res, signer.session(signer.newOwner()));
        }
        if (!process.env.WX_APP_ID || !process.env.WX_APP_SECRET) throw new AppError('WX_NOT_CONFIGURED', '请先配置微信 AppID 和 AppSecret；本地可使用演示登录', 503);
        if (typeof input.code !== 'string' || input.code.length > 256) throw new AppError('INVALID_CODE', '微信登录凭证无效');
        const endpoint = new URL('https://api.weixin.qq.com/sns/jscode2session');
        endpoint.search = new URLSearchParams({ appid: process.env.WX_APP_ID, secret: process.env.WX_APP_SECRET, js_code: input.code, grant_type: 'authorization_code' });
        const response = await fetch(endpoint, { signal: AbortSignal.timeout(10000) });
        const wx = await response.json();
        if (!wx.openid) throw new AppError('WX_LOGIN_FAILED', '微信登录失败，请重新进入小程序', 401);
        return send(res, signer.session(createHash('sha256').update(wx.openid + secret).digest('hex')));
      }
      const owner = signer.authenticate(req.headers.authorization);
      if (url.pathname === '/api/jobs' && req.method === 'GET') {
        const values = [...jobs.items.values()].filter(j => j.owner === owner && j.expiresAt > Date.now()).sort((a, b) => b.createdAt - a.createdAt);
        return send(res, { jobs: values.map(j => publicJob(j, signer, baseUrl)) });
      }
      if (url.pathname === '/api/demo' && req.method === 'POST') {
        if (!demoMode || !local(req)) throw new AppError('DEMO_DISABLED', '当前环境未开放示例视频', 403);
        limiter.take('create:' + owner, 10, 60000);
        return send(res, publicJob(await jobs.demo(owner), signer, baseUrl), 201);
      }
      if (url.pathname === '/api/jobs' && req.method === 'POST') {
        limiter.take('create:' + owner, 10, 60000);
        const input = await body(req);
        if (input.authorized !== true) throw new AppError('AUTHORIZATION_REQUIRED', '请确认你拥有该视频的保存权限');
        if (!['original', 'compatible'].includes(input.mode)) throw new AppError('INVALID_MODE', '请选择有效的保存模式');
        const link = parseLink(input.text, directHosts);
        if (!extractor) throw new AppError('DEPENDENCY_MISSING', '视频解析引擎尚未安装，请先完成后端部署', 503);
        return send(res, publicJob(jobs.create(owner, link, input.mode), signer, baseUrl), 202);
      }
      const match = /^\/api\/jobs\/([a-f0-9-]{36})$/.exec(url.pathname);
      if (match && req.method === 'GET') return send(res, publicJob(jobs.get(match[1], owner), signer, baseUrl));
      if (match && req.method === 'DELETE') { jobs.cancel(match[1], owner); return send(res, { ok: true }); }
      throw new AppError('NOT_FOUND', '接口不存在', 404);
    }
    const mediaMatch = /^\/media\/([a-f0-9-]{36})$/.exec(url.pathname);
    if (mediaMatch && ['GET', 'HEAD'].includes(req.method)) {
      const expText = url.searchParams.get('exp'); const exp = Number(expText);
      if (!Number.isSafeInteger(exp) || exp < Date.now() || !signer.verify(`media:${mediaMatch[1]}:${expText}`, url.searchParams.get('sig') || '')) throw new AppError('MEDIA_EXPIRED', '下载地址已过期，请从记录重新打开', 403);
      const job = jobs.get(mediaMatch[1]);
      if (job.status !== 'ready') throw new AppError('NOT_READY', '视频尚未就绪', 409);
      return await fileResponse(req, res, job.file, url.searchParams.get('download') === '1');
    }
    if (!['GET', 'HEAD'].includes(req.method)) throw new AppError('NOT_FOUND', '页面不存在', 404);
    const relative = url.pathname === '/' ? 'index.html' : decodeURIComponent(url.pathname).replace(/^\//, '');
    const file = path.resolve(root, 'web', relative);
    if (!file.startsWith(path.join(root, 'web') + path.sep)) throw new AppError('NOT_FOUND', '文件不存在', 404);
    try { await fileResponse(req, res, file, false); } catch (error) { if (error.code === 'ENOENT' || error.code === 'EISDIR') throw new AppError('NOT_FOUND', '文件不存在', 404); throw error; }
  } catch (error) {
    if (res.headersSent) { res.destroy(); return; }
    const known = error instanceof AppError;
    if (!known) console.error('request_failed', error.code || error.name);
    send(res, { error: { code: known ? error.code : 'INTERNAL_ERROR', message: known ? error.message : '服务暂时不可用，请稍后重试' } }, known ? error.status : 500);
  }
});
server.requestTimeout = 20000; server.headersTimeout = 15000;
const port = Number(process.env.PORT || 8787);
server.listen(port, process.env.HOST || '127.0.0.1', () => console.log(`留影: ${baseUrl} | extractor=${extractor} | demo=${demoMode}`));
for (const signal of ['SIGTERM', 'SIGINT']) process.once(signal, async () => { server.close(); await jobs.close(); server.closeAllConnections(); });
