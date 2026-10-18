const api = require('../../utils/api');
Page({
  data: { jobs: [], error: '', loading: false },
  onShow() { this.active = true; this.load(); },
  onHide() { this.active = false; clearTimeout(this.timer); },
  onUnload() { this.onHide(); },
  onPullDownRefresh() { this.load().finally(() => wx.stopPullDownRefresh()); },
  async load() {
    clearTimeout(this.timer); this.setData({ loading: true, error: '' });
    try {
      const result = await api.request('/api/jobs');
      this.setData({ jobs: result.jobs.map(api.display) });
      if (this.active && result.jobs.some(j => ['processing', 'queued'].includes(j.status))) this.timer = setTimeout(() => this.load(), 3000);
    } catch (error) { this.setData({ error: error.message }); }
    finally { this.setData({ loading: false }); }
  },
  open(e) { wx.setStorageSync('openJob', e.currentTarget.dataset.id); wx.switchTab({ url: '/pages/index/index' }); },
  remove(e) {
    wx.showModal({ title: '删除这条记录？', content: '临时视频将同时删除，仍可使用原链接重新解析。', success: async res => {
      if (!res.confirm) return;
      try { await api.request('/api/jobs/' + e.currentTarget.dataset.id, 'DELETE'); this.load(); }
      catch (error) { this.setData({ error: error.message }); }
    } });
  }
});
