const api = require('../../utils/api')
Page({
  data: { loading: false, error: '', dev: false, settings: false, host: '' },
  onLoad() { this.setData({ dev: api.environment() === 'develop', host: api.baseURL() }) },
  onShow() { if (api.token()) wx.switchTab({ url: '/pages/ledger/index' }) },
  toggleSettings() { this.setData({ settings: !this.data.settings }) },
  changeHost(e) { this.setData({ host: e.detail.value }) },
  saveHost() {
    try { api.setDevelopHost(this.data.host); this.setData({ error: '', settings: false }); wx.showToast({ title: '连接地址已保存' }) }
    catch (error) { this.setData({ error: error.message }) }
  },
  async login() {
    if (this.data.loading) return
    this.setData({ loading: true, error: '' })
    try { await api.login(); wx.switchTab({ url: '/pages/ledger/index' }) }
    catch (error) { this.setData({ error: error.message }) }
    finally { this.setData({ loading: false }) }
  }
})
