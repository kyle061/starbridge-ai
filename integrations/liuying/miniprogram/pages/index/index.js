const api = require('../../utils/api');
const config = require('../../config');
Page({
  data: { text: '', mode: 'compatible', authorized: false, busy: false, saving: false, downloadProgress: 0, error: '', job: null, privacyVisible: false, demoMode: config.demoMode, platforms: [
    { id: 'douyin', name: '抖音', mark: '抖' }, { id: 'tiktok', name: 'TikTok', mark: '♪' }, { id: 'youtube', name: 'YouTube', mark: '▶' }, { id: 'bilibili', name: '哔哩哔哩', mark: '哔' }, { id: 'more', name: '更多平台', mark: '+' }
  ] },
  onShow() {
    this.active = true;
    const id = wx.getStorageSync('openJob');
    if (id) { wx.removeStorageSync('openJob'); this.loadJob(id); }
    else if (this.data.job && ['queued', 'processing'].includes(this.data.job.status)) this.poll();
  },
  onHide() { this.active = false; clearTimeout(this.pollTimer); },
  onUnload() { this.onHide(); if (this.downloadTask) this.downloadTask.abort(); this.deleteTemp(); },
  input(e) { this.setData({ text: e.detail.value, error: '' }); },
  clear() { this.setData({ text: '', error: '' }); },
  setMode(e) { this.setData({ mode: e.currentTarget.dataset.mode }); },
  setAuthorized(e) { this.setData({ authorized: e.detail.value.includes('authorized') }); },
  async paste() {
    try { await api.privacy(); wx.getClipboardData({ success: res => this.setData({ text: res.data.slice(0, 4096), error: '' }), fail: () => this.setData({ error: '无法读取剪贴板，请手动粘贴' }) }); }
    catch (error) { this.setData({ error: error.message }); }
  },
  async resolve() {
    if (this.data.busy) return;
    if (!this.data.text || !this.data.authorized) { this.setData({ error: '请填写视频链接并确认保存权限' }); return; }
    clearTimeout(this.pollTimer); this.setData({ busy: true, error: '', job: null });
    try {
      const job = await api.request('/api/jobs', 'POST', { text: this.data.text, mode: this.data.mode, authorized: true });
      this.setJob(job); this.poll();
    } catch (error) { this.setData({ error: error.message, busy: false }); }
  },
  async demo() {
    this.setData({ error: '' });
    try { this.setJob(await api.request('/api/demo', 'POST', {})); }
    catch (error) { this.setData({ error: error.message }); }
  },
  async loadJob(id) {
    clearTimeout(this.pollTimer);
    try { this.setJob(await api.request('/api/jobs/' + id)); if (this.data.busy) this.poll(); }
    catch (error) { this.setData({ error: error.message, busy: false }); }
  },
  setJob(job) {
    this.setData({ job: { ...api.display(job), timeExpiry: new Date(job.expiresAt).toLocaleTimeString() }, busy: ['queued', 'processing'].includes(job.status), error: job.error ? job.error.message : '' });
  },
  poll() {
    if (!this.active || !this.data.job) return;
    this.pollTimer = setTimeout(async () => {
      try { this.setJob(await api.request('/api/jobs/' + this.data.job.id)); if (this.data.busy) this.poll(); }
      catch (error) { this.setData({ error: error.message, busy: false }); }
    }, 1800);
  },
  async cancel() {
    if (!this.data.job) return;
    try { await api.request('/api/jobs/' + this.data.job.id, 'DELETE'); clearTimeout(this.pollTimer); this.setData({ busy: false, job: null }); }
    catch (error) { this.setData({ error: error.message }); }
  },
  async download() {
    await api.privacy();
    const job = await api.request('/api/jobs/' + this.data.job.id); this.setJob(job);
    if (job.bytes > 200000000) throw new Error('原文件超过微信单次下载大小限制，请复制地址到浏览器保存，或切换相册兼容模式');
    this.setData({ saving: true, downloadProgress: 0 }); this.cancelledDownload = false;
    this.deleteTemp();
    return new Promise((resolve, reject) => {
      this.downloadTask = wx.downloadFile({
        url: job.downloadUrl, timeout: 600000,
        success: res => {
          this.downloadTask = null;
          if (res.statusCode !== 200) { reject(new Error('视频下载失败，请重新解析')); return; }
          this.tempFile = res.tempFilePath; resolve(res.tempFilePath);
        }, fail: () => { this.downloadTask = null; reject(new Error(this.cancelledDownload ? '已取消下载' : '下载失败，请检查网络或存储空间')); }
      });
      this.downloadTask.onProgressUpdate(event => this.setData({ downloadProgress: event.progress }));
    });
  },
  cancelDownload() { this.cancelledDownload = true; if (this.downloadTask) this.downloadTask.abort(); },
  deleteTemp() { if (this.tempFile) wx.getFileSystemManager().unlink({ filePath: this.tempFile, complete() {} }); this.tempFile = null; },
  async saveAlbum() {
    if (this.data.saving) return;
    try {
      const filePath = await this.download();
      await new Promise((resolve, reject) => wx.saveVideoToPhotosAlbum({ filePath, success: resolve, fail: reject }));
      wx.showToast({ title: '已保存到相册', icon: 'success' });
    } catch (error) {
      if (/privacy agreement|scope is not declared/i.test(error.errMsg || '')) this.setData({ error: '小程序尚未配置相册接口的隐私声明，请联系运营者完成配置后重试' });
      else if (/auth deny|auth denied|authorize|permission/i.test(error.errMsg || '')) wx.showModal({ title: '需要相册权限', content: '请在设置中允许保存到相册', confirmText: '打开设置', success: res => { if (res.confirm) wx.openSetting({}); } });
      else this.setData({ error: error.message || '保存失败，请检查相册权限和可用空间' });
    } finally { this.setData({ saving: false }); this.deleteTemp(); }
  },
  async saveFile() {
    if (this.data.saving) return;
    if (!wx.shareFileMessage) { this.setData({ error: '当前微信版本不支持原文件保存，请更新微信或复制下载地址' }); return; }
    try {
      const filePath = await this.download();
      await new Promise((resolve, reject) => wx.shareFileMessage({ filePath, fileName: (this.data.job.transcoded ? '视频.' : '原视频.') + this.data.job.ext, success: resolve, fail: reject }));
    } catch (error) { if (!/cancel/.test(error.errMsg || '')) this.setData({ error: error.message || '原文件保存未完成' }); }
    finally { this.setData({ saving: false }); this.deleteTemp(); }
  },
  copyDownload() { api.request('/api/jobs/' + this.data.job.id).then(job => wx.setClipboardData({ data: job.downloadUrl })).catch(error => this.setData({ error: error.message })); },
  videoError() { this.setData({ error: '视频暂时无法播放，可尝试保存原文件或重新解析' }); },
  openPrivacy() { if (wx.openPrivacyContract) wx.openPrivacyContract({}); },
  agreePrivacy() { getApp().acceptPrivacy(); this.setData({ privacyVisible: false }); },
  rejectPrivacy() { getApp().rejectPrivacy(); this.setData({ privacyVisible: false }); }
});
