const config = require('../config');
let loginPromise;
function baseUrl() { return config.demoMode ? (wx.getStorageSync('apiBaseUrl') || config.apiBaseUrl) : config.apiBaseUrl; }
function raw(path, method, data, token) {
  return new Promise((resolve, reject) => wx.request({
    url: baseUrl() + path, method, data, timeout: 20000,
    header: { 'Content-Type': 'application/json', ...(token ? { Authorization: 'Bearer ' + token } : {}) },
    success(res) {
      if (res.statusCode >= 200 && res.statusCode < 300) resolve(res.data);
      else { const err = new Error(res.data && res.data.error ? res.data.error.message : '服务连接失败'); err.status = res.statusCode; reject(err); }
    },
    fail() { reject(new Error('无法连接服务，请检查网络及服务器地址')); }
  }));
}
function login() {
  const saved = wx.getStorageSync('session');
  if (saved && saved.expiresAt > Date.now() + 60000) return Promise.resolve(saved.token);
  if (loginPromise) return loginPromise;
  loginPromise = (config.demoMode ? Promise.resolve({ demo: true }) : new Promise((resolve, reject) => wx.login({
    success: res => res.code ? resolve({ code: res.code }) : reject(new Error('微信登录失败')), fail: reject
  }))).then(data => raw('/api/session', 'POST', data)).then(session => {
    wx.setStorageSync('session', session); return session.token;
  }).finally(() => { loginPromise = null; });
  return loginPromise;
}
async function request(path, method = 'GET', data, retry = true) {
  try { return await raw(path, method, data, await login()); }
  catch (error) {
    if (error.status === 401 && retry) { wx.removeStorageSync('session'); return request(path, method, data, false); }
    throw error;
  }
}
function privacy() {
  return new Promise((resolve, reject) => {
    if (!wx.requirePrivacyAuthorize) { resolve(); return; }
    wx.requirePrivacyAuthorize({ success: resolve, fail: () => reject(new Error('请先同意隐私保护指引')) });
  });
}
function display(job) {
  return { ...job, sizeText: job.bytes ? (job.bytes / 1048576).toFixed(1) + ' MB' : '',
    durationText: job.duration ? Math.floor(job.duration / 60) + ':' + String(Math.floor(job.duration % 60)).padStart(2, '0') : '',
    timeText: new Date(job.createdAt).toLocaleString(),
    statusText: ({ ready: '可保存', queued: '排队中', processing: '处理中', failed: '解析失败', cancelled: '已取消' })[job.status] || job.status };
}
module.exports = { request, raw, baseUrl, privacy, display };
