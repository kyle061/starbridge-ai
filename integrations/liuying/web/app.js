const $ = selector => document.querySelector(selector);
const base = new URL('./', location.href);
const state = { token: null, mode: 'compatible', job: null, health: null, platforms: [], polling: null, view: 'workspace', busy: false };
const labels = { ready: '可保存', queued: '排队中', processing: '处理中', failed: '解析失败' };
const toast = message => { $('#toast').textContent = message; $('#toast').hidden = false; clearTimeout(state.toastTimer); state.toastTimer = setTimeout(() => { $('#toast').hidden = true; }, 3200); };
const showError = message => { $('#error').textContent = message || ''; $('#error').hidden = !message; };
const sizeText = size => size ? (size / 1048576).toFixed(1) + ' MB' : '';
const durationText = seconds => seconds ? `${Math.floor(seconds / 60)}:${String(Math.floor(seconds % 60)).padStart(2, '0')}` : '';
const sessionKey = 'liuying-session:' + base.pathname;
async function raw(route, method = 'GET', data, token) {
  const response = await fetch(new URL(route.replace(/^\//, ''), base), { method, headers: { 'Content-Type': 'application/json', ...(token ? { Authorization: `Bearer ${token}` } : {}) }, ...(data ? { body: JSON.stringify(data) } : {}), signal: AbortSignal.timeout(25000) });
  const result = await response.json();
  if (!response.ok) { const error = new Error(result.error?.message || '服务连接失败'); error.status = response.status; throw error; }
  return result;
}
async function authenticate() {
  let saved;
  try { saved = JSON.parse(localStorage.getItem(sessionKey)); } catch { /* Ignore invalid local cache. */ }
  if (saved?.expiresAt > Date.now() + 60000) { state.token = saved.token; return; }
  if (state.health?.demoMode) {
    saved = await raw('api/session', 'POST', { demo: true });
  } else {
    const key = await accessKey();
    saved = await raw('api/session', 'POST', { previewKey: key });
  }
  localStorage.setItem(sessionKey, JSON.stringify(saved)); state.token = saved.token;
}
function accessKey() {
  $('#access-dialog').showModal();
  return new Promise((resolve, reject) => {
    $('#access-form').onsubmit = event => {
      event.preventDefault(); const value = $('#access-key').value; $('#access-key').value = ''; $('#access-dialog').close(); resolve(value);
    };
    $('#access-cancel').onclick = () => { $('#access-dialog').close(); reject(new Error('工作台尚未验证')); };
    $('#access-dialog').oncancel = () => reject(new Error('工作台尚未验证'));
  });
}
async function api(route, method = 'GET', data, retry = true) {
  if (!state.token) await authenticate();
  try { return await raw(route, method, data, state.token); }
  catch (error) {
    if (error.status === 401 && retry) { state.token = null; localStorage.removeItem(sessionKey); return api(route, method, data, false); }
    throw error;
  }
}
function updateInput() {
  const text = $('#share-text').value;
  $('#text-count').textContent = `${text.length} / 4096`;
  const found = state.platforms.find(p => ({ douyin: /douyin\.com/, tiktok: /tiktok\.com/, youtube: /youtu(be\.com|\.be)/, bilibili: /bilibili\.com|b23\.tv/, instagram: /instagram\.com/, twitter: /twitter\.com|x\.com/, vimeo: /vimeo\.com/, facebook: /facebook\.com|fb\.watch/ })[p.id]?.test(text));
  $('#detected-platform').textContent = found ? found.name : (text ? '提交后识别视频源' : '等待视频链接');
  $('#clear').hidden = !text;
}
function setBusy(value) {
  state.busy = value; $('#resolve').disabled = value;
  $('#resolve span').textContent = value ? '正在解析视频' : '解析视频';
  $('#task-progress').hidden = !value;
}
function resetPreview() {
  $('#video').removeAttribute('src'); $('#video').removeAttribute('poster'); $('#video').controls = false; $('#video').load();
  $('#preview-overlay').hidden = true;
  $('#video-title').textContent = '你的下一个好片段';
  $('#video-details').textContent = '解析完成后，视频将在这里出现';
  $('#preview-status').textContent = '等待解析';
  $('#save-note').textContent = '';
  $('#save').disabled = $('#save-original').disabled = $('#copy').disabled = true;
}
function renderJob(job) {
  state.job = job;
  const busy = ['queued', 'processing'].includes(job.status); setBusy(busy);
  const transcoding = busy && (job.phase === 'transcoding' || (!job.phase && job.progress >= 96));
  $('#preview-status').textContent = transcoding ? '生成相册兼容版' : (labels[job.status] || job.status);
  if (busy) {
    if (transcoding) $('#progress').removeAttribute('value'); else $('#progress').value = job.progress;
    $('#task-status').textContent = transcoding ? '生成相册兼容版' : `${labels[job.status]} · ${Math.floor(job.progress)}%`;
    $('#resolve span').textContent = transcoding ? '生成相册兼容版' : '正在解析视频';
  }
  $('#save').disabled = $('#save-original').disabled = $('#copy').disabled = job.status !== 'ready';
  if (job.status === 'ready') {
    $('#video').controls = true; $('#video').src = job.mediaUrl; $('#preview-overlay').hidden = true;
    $('#video-title').textContent = job.title;
    $('#video-details').textContent = [job.platformName, durationText(job.duration), sizeText(job.bytes), job.transcoded ? '相册兼容版' : '', job.isDemo ? '示例视频' : ''].filter(Boolean).join(' · ');
    $('#save-original').lastChild.textContent = job.transcoded ? '下载视频文件' : '下载原文件';
    $('.source-note strong').textContent = job.transcoded ? '相册兼容版' : '保留原有画质';
    $('.source-note p').textContent = job.transcoded ? 'H.264 MP4 · 已压缩' : '仅获取平台可用的视频源，不添加水印。';
    $('#format-label').textContent = job.ext.toUpperCase();
    $('#save-note').textContent = `临时文件保留至 ${new Date(job.expiresAt).toLocaleTimeString()}${job.compatible ? '' : ' · 当前格式请使用原文件下载'}`;
    $('#save').disabled = !job.compatible;
  }
  if (job.error) showError(job.error.message);
  refreshHistory(false).catch(() => {});
}
function poll() {
  clearTimeout(state.polling);
  if (!state.job || !['queued', 'processing'].includes(state.job.status)) return;
  state.polling = setTimeout(async () => {
    try { renderJob(await api('api/jobs/' + state.job.id)); poll(); }
    catch (error) { setBusy(false); showError(error.message); }
  }, 1800);
}
$('#resolve-form').addEventListener('submit', async event => {
  event.preventDefault(); if (state.busy) return;
  showError(''); setBusy(true); clearTimeout(state.polling);
  try {
    const job = await api('api/jobs', 'POST', { text: $('#share-text').value, mode: state.mode, authorized: $('#authorized').checked });
    resetPreview(); renderJob(job); poll();
  } catch (error) { setBusy(false); showError(error.message); }
});
$('#demo').addEventListener('click', async () => {
  if (state.busy) return;
  showError('');
  try { renderJob(await api('api/demo', 'POST', {})); toast('示例视频已就绪'); }
  catch (error) { showError(error.message); }
});
$('#paste').addEventListener('click', async () => {
  try { $('#share-text').value = (await navigator.clipboard.readText()).slice(0, 4096); updateInput(); }
  catch { toast('请在链接框中手动粘贴'); $('#share-text').focus(); }
});
$('#clear').addEventListener('click', () => { $('#share-text').value = ''; updateInput(); showError(''); });
$('#share-text').addEventListener('input', updateInput);
document.querySelectorAll('[data-mode]').forEach(button => button.addEventListener('click', () => {
  state.mode = button.dataset.mode; document.querySelectorAll('[data-mode]').forEach(b => b.classList.toggle('selected', b === button));
}));
$('#cancel').addEventListener('click', async () => {
  if (!state.job) return;
  try { await api('api/jobs/' + state.job.id, 'DELETE'); clearTimeout(state.polling); state.job = null; setBusy(false); $('#preview-status').textContent = '已取消'; toast('已取消任务'); }
  catch (error) { showError(error.message); }
});
async function download() {
  if (!state.job) return;
  try {
    const job = await api('api/jobs/' + state.job.id);
    const anchor = document.createElement('a'); anchor.href = job.downloadUrl; anchor.download = `留影-${job.id}.${job.ext}`; anchor.click();
  } catch (error) { showError(error.message); }
}
$('#save').addEventListener('click', download); $('#save-original').addEventListener('click', download);
$('#copy').addEventListener('click', async () => {
  try { const job = await api('api/jobs/' + state.job.id); await navigator.clipboard.writeText(job.downloadUrl); toast('下载地址已复制，有效期 15 分钟'); }
  catch (error) { showError(error.message || '复制失败'); }
});
$('#video').addEventListener('error', () => { if (state.job?.status === 'ready') showError('浏览器无法预览此格式，可直接下载原文件'); });
async function refreshHistory(render = true) {
  const { jobs } = await api('api/jobs');
  $('#history-count').textContent = jobs.length; $('#record-count').textContent = `${jobs.length} 条记录`;
  if (!render) return;
  const list = $('#history-list'); list.replaceChildren();
  if (!jobs.length) { const empty = document.createElement('p'); empty.className = 'empty-history'; empty.textContent = '还没有解析记录'; list.append(empty); }
  for (const job of jobs) {
    const row = document.createElement('div'); row.className = 'record';
    const open = document.createElement('button'); open.className = 'record-open';
    const badge = document.createElement('span'); badge.className = 'badge'; badge.textContent = job.platformName;
    const title = document.createElement('h3'); title.textContent = job.title;
    const details = document.createElement('p'); details.textContent = `${new Date(job.createdAt).toLocaleString()} · ${sizeText(job.bytes) || job.mode}`;
    open.append(badge, title, details); open.onclick = async () => { switchView('workspace'); showError(''); try { renderJob(await api('api/jobs/' + job.id)); poll(); } catch (error) { showError(error.message); } };
    const status = document.createElement('span'); status.className = 'record-status'; status.textContent = labels[job.status] || job.status;
    const remove = document.createElement('button'); remove.className = 'icon-button'; remove.title = '删除记录'; remove.setAttribute('aria-label', '删除 ' + job.title);
    const icon = document.createElement('img'); icon.src = new URL('assets/trash-2.svg', base).href; icon.alt = ''; remove.append(icon);
    remove.onclick = () => { state.deleteId = job.id; $('#delete-dialog').showModal(); };
    row.append(open, status, remove); list.append(row);
  }
}
function switchView(name) {
  state.view = name;
  document.querySelectorAll('.view').forEach(v => { v.hidden = v.id !== name; });
  document.querySelectorAll('[data-view]').forEach(b => b.classList.toggle('active', b.dataset.view === name));
  $('#view-label').textContent = ({ workspace: '视频解析', history: '解析记录', settings: '服务设置' })[name];
  if (name === 'history') refreshHistory().catch(error => toast(error.message));
}
document.querySelectorAll('[data-view]').forEach(button => button.addEventListener('click', () => switchView(button.dataset.view)));
$('#refresh-history').onclick = () => refreshHistory().catch(error => toast(error.message));
$('#delete-cancel').onclick = () => $('#delete-dialog').close();
$('#delete-confirm').onclick = async () => {
  $('#delete-dialog').close();
  try { await api('api/jobs/' + state.deleteId, 'DELETE'); if (state.job?.id === state.deleteId) { state.job = null; clearTimeout(state.polling); setBusy(false); resetPreview(); } await refreshHistory(); toast('记录已删除'); }
  catch (error) { toast(error.message); }
};
async function init() {
  try {
    const health = await raw('api/health'); state.health = health;
    const result = await raw('api/platforms'); state.platforms = result.platforms;
    $('#connection').textContent = health.extractor ? '服务已连接' : '引擎待安装';
    $('#demo').hidden = !health.demoMode;
    const entries = [['解析引擎', health.extractor ? '已就绪' : '待安装'], ['当前环境', health.demoMode ? '本地开发' : '服务器'], ['文件大小上限', health.maxMb + ' MB'], ['视频时长上限', health.maxDuration / 60 + ' 分钟'], ['临时文件保留', health.ttl / 60 + ' 分钟'], ['可用解析器', result.platforms.length + ' 个平台']];
    for (const [key, value] of entries) { const row = document.createElement('div'); const dt = document.createElement('dt'); dt.textContent = key; const dd = document.createElement('dd'); dd.textContent = value; row.append(dt, dd); $('#settings-list').append(row); }
    if (health.demoMode) { await authenticate(); await refreshHistory(false); }
  } catch (error) { $('#connection').textContent = '服务未连接'; showError(error.message); }
  updateInput();
}
init();
