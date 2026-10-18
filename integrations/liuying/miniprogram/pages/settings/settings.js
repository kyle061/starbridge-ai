const api = require('../../utils/api'); const config = require('../../config');
Page({
  data: { baseUrl: '', demoMode: config.demoMode, health: null, error: '', platforms: [] },
  onShow() { this.setData({ baseUrl: api.baseUrl() }); this.check(); },
  input(e) { this.setData({ baseUrl: e.detail.value }); },
  async check() {
    try {
      this.setData({ error: '' });
      const health = await api.raw('/api/health', 'GET');
      const result = await api.raw('/api/platforms', 'GET');
      this.setData({ health, platforms: result.platforms });
    } catch (error) { this.setData({ error: error.message, health: null }); }
  },
  save() {
    if (!config.demoMode) return;
    const value = this.data.baseUrl.trim().replace(/\/$/, '');
    if (!/^https:\/\/[a-zA-Z0-9.-]+(?::\d+)?(?:\/[a-zA-Z0-9/_-]*)?$/.test(value) && !/^http:\/\/(127\.0\.0\.1|localhost)(:\d+)?$/.test(value)) { this.setData({ error: '请输入 HTTPS 服务地址，本机开发可使用 http://127.0.0.1:8787' }); return; }
    wx.setStorageSync('apiBaseUrl', value); wx.removeStorageSync('session'); this.check(); wx.showToast({ title: '已保存' });
  },
  albumSettings() { wx.openSetting({}); },
  privacy() { if (wx.openPrivacyContract) wx.openPrivacyContract({}); }
});
