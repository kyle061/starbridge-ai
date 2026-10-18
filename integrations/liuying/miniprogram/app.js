App({
  onLaunch() {
    if (wx.onNeedPrivacyAuthorization) {
      wx.onNeedPrivacyAuthorization(resolve => {
        this.privacyResolve = resolve;
        const pages = getCurrentPages();
        const page = pages[pages.length - 1];
        if (page) page.setData({ privacyVisible: true });
      });
    }
  },
  acceptPrivacy() {
    if (this.privacyResolve) this.privacyResolve({ event: 'agree', buttonId: 'privacy-agree' });
    this.privacyResolve = null;
  },
  rejectPrivacy() {
    if (this.privacyResolve) this.privacyResolve({ event: 'disagree' });
    this.privacyResolve = null;
  }
});
