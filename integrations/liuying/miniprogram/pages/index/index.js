const api = require('../../utils/api');
const config = require('../../config');
Page({
  data: { text: '', authorized: false, busy: false, saving: false, downloadProgress: 0, saveLabel: '', savedCount: 0, error: '', job: null, privacyVisible: false, demoMode: config.demoMode, platforms: [
    { id: 'douyin', name: '抖音', mark: '抖' }, { id: 'tiktok', name: 'TikTok', mark: '♪' }, { id: 'youtube', name: 'YouTube', mark: '▶' }, { id: 'bilibili', name: '哔哩哔哩', mark: '哔' }, { id: 'more', name: '更多平台', mark: '+' }
  ] },
  onShow() {
    this.active = true;
    const id = wx.getStorageSync('openJob');
    if (id) { wx.removeStorageSync('openJob'); this.loadJob(id); }
    else if (this.data.job && ['queued', 'processing'].includes(this.data.job.status)) this.poll();
  },
  onHide() { this.active = false; clearTimeout(this.pollTimer); },
  onUnload() { this.cancelledDownload = true; this.onHide(); if (this.downloadTask) this.downloadTask.abort(); this.deleteTemp(); },
  input(e) { this.setData({ text: e.detail.value, error: '' }); },
  clear() { this.setData({ text: '', error: '' }); },
  setAuthorized(e) { this.setData({ authorized: e.detail.value.includes('authorized') }); },
  async paste() {
    try { await api.privacy(); wx.getClipboardData({ success: res => this.setData({ text: res.data.slice(0, 4096), error: '' }), fail: () => this.setData({ error: '无法读取剪贴板，请手动粘贴' }) }); }
    catch (error) { this.setData({ error: error.message }); }
  },
  async resolve() {
    if (this.data.busy || this.data.saving) return;
    if (!this.data.text || !this.data.authorized) { this.setData({ error: '请填写分享链接并确认保存权限' }); return; }
    clearTimeout(this.pollTimer); this.savedIds = new Set(); this.setData({ savedCount: 0, busy: true, error: '', job: null });
    try {
      const job = await api.request('/api/jobs', 'POST', { text: this.data.text, authorized: true });
      this.setJob(job); this.poll();
    } catch (error) { this.setData({ error: error.message, busy: false }); }
  },
  async demo(e) {
    this.setData({ error: '' });
    try { this.setJob(await api.request('/api/demo', 'POST', { kind: e.currentTarget.dataset.kind || 'video' })); }
    catch (error) { this.setData({ error: error.message }); }
  },
  async loadJob(id) {
    clearTimeout(this.pollTimer);
    try { this.setJob(await api.request('/api/jobs/' + id)); if (this.data.busy) this.poll(); }
    catch (error) { this.setData({ error: error.message, busy: false }); }
  },
  setJob(job) {
    if (this.savedJobId !== job.id) { this.savedJobId = job.id; this.savedIds = new Set(); this.setData({ savedCount: 0 }); }
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
  previewImage(e) {
    const images = this.data.job.assets.filter(a => a.kind === 'image');
    wx.previewImage({ current: images[Number(e.currentTarget.dataset.index)].mediaUrl, urls: images.map(a => a.mediaUrl) });
  },
  async download(url, bytes) {
    if (this.cancelledDownload) throw new Error('已取消保存');
    if (bytes > 200000000) throw new Error('原文件超过微信单次下载限制，请复制地址到浏览器下载原文件');
    this.setData({ downloadProgress: 0 }); this.deleteTemp();
    return new Promise((resolve, reject) => {
      this.downloadTask = wx.downloadFile({ url, timeout: 600000,
        success: res => {
          this.downloadTask = null;
          if (res.statusCode !== 200) { this.tempFile = res.tempFilePath; this.deleteTemp(); reject(new Error('文件下载失败，请刷新记录后重试')); return; }
          this.tempFile = res.tempFilePath; resolve(res.tempFilePath);
        }, fail: () => { this.downloadTask = null; reject(new Error(this.cancelledDownload ? '已取消保存' : '下载失败，请检查网络和存储空间')); }
      });
      this.downloadTask.onProgressUpdate(event => this.setData({ downloadProgress: event.progress }));
    });
  },
  cancelDownload() { this.cancelledDownload = true; if (this.downloadTask) this.downloadTask.abort(); },
  deleteTemp() { if (this.tempFile) wx.getFileSystemManager().unlink({ filePath: this.tempFile, complete() {} }); this.tempFile = null; },
  async albumReady(job) {
    if (job.assets.every(a => a.albumUrl)) return job;
    this.setData({ saveLabel: '正在准备相册副本' });
    job = await api.request('/api/jobs/' + job.id + '/album', 'POST', {});
    while (['queued', 'processing'].includes(job.albumStatus)) {
      if (this.cancelledDownload) throw new Error('已取消保存');
      await new Promise(resolve => setTimeout(resolve, 1800));
      job = await api.request('/api/jobs/' + job.id);
    }
    this.setJob(job);
    if (job.albumStatus !== 'ready') throw new Error(job.albumError ? job.albumError.message : '相册副本暂未就绪，请重试');
    return job;
  },
  async saveAlbum(e) {
    if (this.data.saving) return;
    const selected = e && e.currentTarget && e.currentTarget.dataset.id;
    let completed = 0, planned = 0;
    this.cancelledDownload = false;
    this.setData({ saving: true, downloadProgress: 0, error: '', saveLabel: '准备保存' });
    try {
      await api.privacy();
      const job = await this.albumReady(await api.request('/api/jobs/' + this.data.job.id));
      this.savedIds = this.savedIds || new Set();
      const assets = job.assets.filter(a => selected !== undefined ? a.id === String(selected) : !this.savedIds.has(a.id));
      planned = assets.length;
      if (!planned) { wx.showToast({ title: '全部已保存', icon: 'success' }); return; }
      for (const asset of assets) {
        if (this.cancelledDownload) throw new Error('已取消保存');
        this.setData({ saveLabel: '保存 ' + (completed + 1) + ' / ' + planned });
        const fresh = await api.request('/api/jobs/' + job.id);
        const current = fresh.assets.find(a => a.id === asset.id);
        const filePath = await this.download(current.albumUrl, current.albumUrl === current.mediaUrl ? current.bytes : 0);
        const save = asset.kind === 'image' ? wx.saveImageToPhotosAlbum : wx.saveVideoToPhotosAlbum;
        await new Promise((resolve, reject) => save({ filePath, success: resolve, fail: reject }));
        completed++; this.savedIds.add(asset.id); this.setData({ savedCount: this.savedIds.size }); this.deleteTemp();
      }
      wx.showToast({ title: '已保存到相册', icon: 'success' });
    } catch (error) {
      let message = error.message || '保存失败，请检查相册权限和可用空间';
      if (/privacy agreement|scope is not declared/i.test(error.errMsg || '')) message = '微信相册隐私声明尚未生效，请联系运营者完成配置后重试';
      else if (/auth deny|auth denied|authorize|permission/i.test(error.errMsg || '')) {
        message = '请允许相册权限后重试';
        wx.showModal({ title: '需要相册权限', content: '请在设置中允许保存到相册', confirmText: '打开设置', success: res => { if (res.confirm) wx.openSetting({}); } });
      }
      this.setData({ error: (completed ? '本次已保存 ' + completed + ' / ' + planned + '，再次保存会继续剩余文件。' : '') + message });
    } finally { this.setData({ saving: false, saveLabel: '' }); this.deleteTemp(); }
  },
  async saveFile() {
    if (this.data.saving) return;
    if (!wx.shareFileMessage) { this.setData({ error: '请更新微信或复制下载地址保存原文件' }); return; }
    this.cancelledDownload = false; this.setData({ saving: true, downloadProgress: 0, error: '', saveLabel: '下载原文件' });
    try {
      await api.privacy();
      const job = await api.request('/api/jobs/' + this.data.job.id); this.setJob(job);
      const bundle = job.assets.length > 1;
      const filePath = await this.download(job.downloadUrl, bundle ? job.bytes + 65536 : job.bytes);
      await new Promise((resolve, reject) => wx.shareFileMessage({ filePath,
        fileName: bundle ? '留影原图.zip' : (job.kind === 'images' ? '留影原图.' : '留影原视频.') + job.ext, success: resolve, fail: reject }));
    } catch (error) { if (!/cancel/.test(error.errMsg || '')) this.setData({ error: error.message || '原文件保存未完成' }); }
    finally { this.setData({ saving: false, saveLabel: '' }); this.deleteTemp(); }
  },
  copyDownload() { api.request('/api/jobs/' + this.data.job.id).then(job => wx.setClipboardData({ data: job.downloadUrl })).catch(error => this.setData({ error: error.message })); },
  videoError() { this.setData({ error: '视频暂时无法播放，可尝试保存原文件或重新解析' }); },
  openPrivacy() { if (wx.openPrivacyContract) wx.openPrivacyContract({}); },
  agreePrivacy() { getApp().acceptPrivacy(); this.setData({ privacyVisible: false }); },
  rejectPrivacy() { getApp().rejectPrivacy(); this.setData({ privacyVisible: false }); }
});
