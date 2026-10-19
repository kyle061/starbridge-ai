import { createHmac, randomBytes, timingSafeEqual } from 'node:crypto';

export class AppError extends Error {
  constructor(code, message, status = 400) { super(message); this.code = code; this.status = status; }
}

export const PLATFORMS = [
  { id: 'douyin', name: '抖音', hosts: ['douyin.com', 'iesdouyin.com'], mark: '抖', color: '#141717' },
  { id: 'tiktok', name: 'TikTok', hosts: ['tiktok.com'], mark: '♪', color: '#141717' },
  { id: 'youtube', name: 'YouTube', hosts: ['youtube.com', 'youtu.be'], mark: '▶', color: '#ef4047' },
  { id: 'bilibili', name: '哔哩哔哩', hosts: ['bilibili.com', 'b23.tv'], mark: '哔', color: '#ed83a6' },
  { id: 'instagram', name: 'Instagram', hosts: ['instagram.com'], mark: '◎', color: '#b64e88' },
  { id: 'twitter', name: 'X / Twitter', hosts: ['twitter.com', 'x.com', 't.co'], mark: 'X', color: '#141717' },
  { id: 'vimeo', name: 'Vimeo', hosts: ['vimeo.com'], mark: 'v', color: '#248faa' },
  { id: 'facebook', name: 'Facebook', hosts: ['facebook.com', 'fb.watch'], mark: 'f', color: '#3478df' }
];

export function parseLink(text, directHosts = []) {
  if (typeof text !== 'string' || !text.trim() || text.length > 4096)
    throw new AppError('INVALID_LINK', '请粘贴有效的视频分享链接');
  const matches = text.match(/https?:\/\/[^\s<>"，。！？；（）【】]+/gi) || [];
  for (let raw of matches) {
    raw = raw.replace(/[)\],.!?;]+$/, '');
    let url;
    try { url = new URL(raw); } catch { continue; }
    if (url.protocol !== 'https:' || url.username || url.password || (url.port && url.port !== '443')) continue;
    const host = url.hostname.toLowerCase();
    const platform = PLATFORMS.find(p => p.hosts.some(h => host === h || host.endsWith('.' + h)));
    if (platform) return { url: url.href, platform: platform.id, platformName: platform.name };
    if (directHosts.includes(host) && /\.(mp4|webm|mov)$/i.test(url.pathname))
      return { url: url.href, platform: 'direct', platformName: '视频直链' };
  }
  throw new AppError('UNSUPPORTED_LINK', '未找到支持的平台链接，请使用 HTTPS 视频分享链接');
}

export class Signer {
  constructor(secret) { this.secret = secret; }
  sign(data) { return createHmac('sha256', this.secret).update(data).digest('base64url'); }
  verify(data, signature = '') {
    const a = Buffer.from(this.sign(data)); const b = Buffer.from(signature);
    return a.length === b.length && timingSafeEqual(a, b);
  }
  session(owner, now = Date.now()) {
    const exp = now + 7 * 86400000;
    const payload = Buffer.from(JSON.stringify({ owner, exp })).toString('base64url');
    return { token: `${payload}.${this.sign('session:' + payload)}`, expiresAt: exp };
  }
  authenticate(header, now = Date.now()) {
    const [payload, sig, extra] = String(header || '').replace(/^Bearer /, '').split('.');
    if (!payload || extra || !this.verify('session:' + payload, sig)) throw new AppError('UNAUTHORIZED', '登录已过期，请重新登录', 401);
    let value;
    try { value = JSON.parse(Buffer.from(payload, 'base64url')); } catch { throw new AppError('UNAUTHORIZED', '登录无效', 401); }
    if (!value.owner || !Number.isFinite(value.exp) || value.exp <= now) throw new AppError('UNAUTHORIZED', '登录已过期，请重新登录', 401);
    return value.owner;
  }
  media(id, exp) { return this.sign(`media:${id}:${exp}`); }
  newOwner() { return randomBytes(18).toString('hex'); }
}

export function parseRange(header, size) {
  if (!header) return null;
  const match = /^bytes=(\d*)-(\d*)$/.exec(header);
  if (!match || (!match[1] && !match[2])) throw new AppError('RANGE', '无效的文件范围', 416);
  const start = match[1] ? Number(match[1]) : Math.max(0, size - Number(match[2]));
  const end = match[1] && match[2] ? Math.min(Number(match[2]), size - 1) : size - 1;
  if (!Number.isSafeInteger(start) || !Number.isSafeInteger(end) || start < 0 || start >= size || start > end)
    throw new AppError('RANGE', '无效的文件范围', 416);
  return { start, end };
}

export function publicJob(job, signer, baseUrl) {
  const { id, status, phase, platform, platformName, mode, progress, title, duration, bytes, ext, compatible, transcoded, error, createdAt, expiresAt, isDemo } = job;
  const output = { id, status, phase, platform, platformName, mode, progress, title, duration, bytes, ext, compatible, transcoded, error, createdAt, expiresAt, isDemo };
  if (status === 'ready') {
    const exp = Math.min(expiresAt, Date.now() + 900000);
    output.mediaUrl = `${baseUrl}/media/${id}?exp=${exp}&sig=${signer.media(id, exp)}`;
    output.downloadUrl = output.mediaUrl + '&download=1';
  }
  return output;
}

export class RateLimiter {
  constructor() { this.buckets = new Map(); }
  take(key, limit, windowMs, now = Date.now()) {
    if (this.buckets.size > 10000) for (const [k, v] of this.buckets) if (v.until <= now) this.buckets.delete(k);
    const bucket = this.buckets.get(key);
    if (!bucket || bucket.until <= now) { this.buckets.set(key, { count: 1, until: now + windowMs }); return; }
    if (++bucket.count > limit) throw new AppError('RATE_LIMIT', '操作太频繁，请稍后重试', 429);
  }
}
