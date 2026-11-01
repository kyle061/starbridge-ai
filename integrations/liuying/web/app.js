const $ = selector => document.querySelector(selector);
const base = new URL('./', location.href);
const state = { token: null, job: null, health: null, platforms: [], polling: null, view: 'workspace', busy: false };
const labels = { ready: '可保存', queued: '排队中', processing: '处理中', failed: '解析失败' };
const toast = message => { $('#toast').textContent = message; $('#toast').hidden = false; clearTimeout(state.toastTimer); state.toastTimer = setTimeout(() => { $('#toast').hidden = true; }, 3200); };
const showError = message => { $('#error').textContent = message || ''; $('#error').hidden = !message; };
const sizeText = size => size ? (size < 1048576 ? Math.ceil(size / 1024) + ' KB' : (size / 1048576).toFixed(1) + ' MB') : '';
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
  $('#detected-platform').textContent = found ? found.name : (text ? '提交后自动识别内容' : '等待分享链接');
  $('#clear').hidden = !text;
}
function setBusy(value) {
  state.busy = value; $('#resolve').disabled = value;
  $('#resolve span').textContent = value ? '正在获取原文件' : '开始解析';
  $('#task-progress').hidden = !value;
}
function resetPreview() {
  $('.preview-frame').classList.remove('has-gallery');
  $('#video').removeAttribute('src'); $('#video').removeAttribute('poster'); $('#video').controls = false; $('#video').load();
  $('#preview-overlay').hidden = false; $('#video').hidden = true; $('#gallery').hidden = true; $('#gallery').replaceChildren();
  $('#video-title').textContent = '原图与视频，在这里相遇';
  $('#video-details').textContent = '粘贴链接后，自动展示图片或视频';
  $('#preview-status').textContent = '等待解析';
  $('#save-note').textContent = '';
  $('#save').disabled = $('#copy').disabled = true;
}
function renderJob(job) {
  state.job = job;
  const busy = ['queued', 'processing'].includes(job.status); setBusy(busy);
  const transcoding = busy && (job.phase === 'transcoding' || (!job.phase && job.progress >= 96));
  $('#preview-status').textContent = transcoding ? '生成相册兼容版' : (labels[job.status] || job.status);
  if (busy) {
    if (transcoding) $('#progress').removeAttribute('value'); else $('#progress').value = job.progress;
    $('#task-status').textContent = transcoding ? '生成相册兼容版' : `${labels[job.status]} · ${Math.floor(job.progress)}%`;
    $('#resolve span').textContent = transcoding ? '生成相册兼容版' : '正在获取原文件';
  }
  $('#save').disabled = $('#copy').disabled = job.status !== 'ready';
  if (job.status === 'ready') {
    const images = job.kind === 'images';
    $('#video').hidden = images; $('#gallery').hidden = !images; $('#preview-overlay').hidden = true;
    $('.preview-frame').classList.toggle('has-gallery', images);
    if (images) {
      $('#video').removeAttribute('src'); $('#video').load(); $('#gallery').replaceChildren();
      $('#gallery').classList.toggle('single-image', job.assets.length === 1);
      for (const [index, asset] of job.assets.entries()) {
        const tile = document.createElement('div'); tile.className = 'image-tile';
        const open = document.createElement('button'); open.className = 'image-open'; open.setAttribute('aria-label', '查看第 ' + (index + 1) + ' 张原图');
        const picture = document.createElement('img'); picture.src = asset.mediaUrl; picture.alt = '原图 ' + (index + 1); picture.loading = 'lazy';
        open.append(picture); open.onclick = () => { $('#full-image').src = asset.mediaUrl; $('#image-dialog').showModal(); };
        const caption = document.createElement('div'); caption.className = 'image-caption';
        const number = document.createElement('span'); number.textContent = String(index + 1).padStart(2, '0') + ' · ' + sizeText(asset.bytes);
        const save = document.createElement('button'); save.className = 'text-button green'; save.textContent = '下载原图'; save.onclick = () => download(asset.id);
        caption.append(number, save); tile.append(open, caption); $('#gallery').append(tile);
      }
    } else { $('#video').controls = true; $('#video').src = job.mediaUrl; }
    $('#video-title').textContent = job.title;
    $('#video-details').textContent = [job.platformName, images ? job.assets.length + ' 张原图' : '原视频', durationText(job.duration), sizeText(job.bytes), job.isDemo ? 'CC0 示例' : ''].filter(Boolean).join(' · ');
    $('#save span').textContent = images && job.assets.length > 1 ? '下载整组原图 · ZIP' : '下载原文件';
    $('.source-note strong').textContent = images ? '每一张，都是原图' : '保留原有画质';
    $('.source-note p').textContent = images ? '按分享顺序保存，不压缩图片。' : '保留平台可获取的原文件，不添加水印。';
    $('#format-label').textContent = images && job.assets.length > 1 ? job.assets.length + ' IMAGES' : job.ext.toUpperCase();
    $('#save-note').textContent = `临时文件保留至 ${new Date(job.expiresAt).toLocaleTimeString()}`;

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
    const job = await api('api/jobs', 'POST', { text: $('#share-text').value, authorized: $('#authorized').checked });
    resetPreview(); renderJob(job); poll();
  } catch (error) { setBusy(false); showError(error.message); }
});
for (const [selector, kind] of [['#demo', 'video'], ['#demo-images', 'images']]) $(selector).addEventListener('click', async () => {
  if (state.busy) return;
  showError('');
  try { resetPreview(); renderJob(await api('api/demo', 'POST', { kind })); toast('CC0 示例已就绪'); }
  catch (error) { showError(error.message); }
});
$('#paste').addEventListener('click', async () => {
  try { $('#share-text').value = (await navigator.clipboard.readText()).slice(0, 4096); updateInput(); }
  catch { toast('请在链接框中手动粘贴'); $('#share-text').focus(); }
});
$('#clear').addEventListener('click', () => { $('#share-text').value = ''; updateInput(); showError(''); });
$('#share-text').addEventListener('input', updateInput);
$('#cancel').addEventListener('click', async () => {
  if (!state.job) return;
  try { await api('api/jobs/' + state.job.id, 'DELETE'); clearTimeout(state.polling); state.job = null; setBusy(false); $('#preview-status').textContent = '已取消'; toast('已取消任务'); }
  catch (error) { showError(error.message); }
});
async function download(assetId) {
  if (!state.job) return;
  try {
    const job = await api('api/jobs/' + state.job.id);
    const asset = typeof assetId === 'string' ? job.assets.find(a => a.id === assetId) : null;
    const anchor = document.createElement('a'); anchor.href = asset ? asset.downloadUrl : job.downloadUrl; anchor.download = ''; anchor.click();
  } catch (error) { showError(error.message); }
}
$('#save').addEventListener('click', () => download());
$('#close-image').onclick = () => $('#image-dialog').close();
$('#copy').addEventListener('click', async () => {
  try { const job = await api('api/jobs/' + state.job.id); await navigator.clipboard.writeText(job.downloadUrl); toast('下载地址已复制，有效期 15 分钟'); }
  catch (error) { showError(error.message || '复制失败'); }
});
$('#video').addEventListener('error', () => { if (state.job?.status === 'ready' && state.job.kind !== 'images') showError('浏览器无法预览此格式，可直接下载原文件'); });
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
    const details = document.createElement('p'); details.textContent = `${new Date(job.createdAt).toLocaleString()} · ${job.kind === 'images' ? job.assets.length + ' 张原图' : '原视频'} · ${sizeText(job.bytes) || '等待获取'}`;
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
  $('#view-label').textContent = ({ workspace: '链接解析', history: '解析记录', settings: '服务设置' })[name];
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
    $('.demo-actions').hidden = !health.demoMode;
    const entries = [['解析引擎', health.extractor ? '已就绪' : '待安装'], ['当前环境', health.demoMode ? '本地开发' : '服务器'], ['文件大小上限', health.maxMb + ' MB'], ['视频时长上限', health.maxDuration / 60 + ' 分钟'], ['临时文件保留', health.ttl / 60 + ' 分钟'], ['可用解析器', result.platforms.length + ' 个平台']];
    for (const [key, value] of entries) { const row = document.createElement('div'); const dt = document.createElement('dt'); dt.textContent = key; const dd = document.createElement('dd'); dd.textContent = value; row.append(dt, dd); $('#settings-list').append(row); }
    if (health.demoMode) { await authenticate(); await refreshHistory(false); }
  } catch (error) { $('#connection').textContent = '服务未连接'; showError(error.message); }
  updateInput();
}
init();
