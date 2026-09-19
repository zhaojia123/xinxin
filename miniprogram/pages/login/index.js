const api = require('../../utils/api')
const config = require('../../config')
function hasPermission(user, key) {
  if (Array.isArray(user.permissions)) return user.permissions.indexOf(key) >= 0
  return user['can_' + key] !== false
}
Page({
  data: { loading: false, error: '', dev: false, settings: false, host: '', filingNumber: config.filingNumber },
  onLoad() { this.setData({ dev: api.environment() === 'develop', host: api.baseURL() }) },
  onShow() { if (api.token()) this.openHome(wx.getStorageSync('xinxin_mini_user') || {}) },
  toggleSettings() { this.setData({ settings: !this.data.settings }) },
  changeHost(e) { this.setData({ host: e.detail.value }) },
  saveHost() {
    try { api.setDevelopHost(this.data.host); this.setData({ error: '', settings: false }); wx.showToast({ title: '连接地址已保存' }) }
    catch (error) { this.setData({ error: error.message }) }
  },
  openHome(user) {
    const target = hasPermission(user, 'ledger') ? '/pages/ledger/index' : hasPermission(user, 'purchases') ? '/pages/purchases/index' : hasPermission(user, 'employees') ? '/pages/employees/index' : ''
    if (target) wx.switchTab({ url: target })
    else this.setData({ error: '当前账号尚未分配可用模块，请联系管理员' })
  },
  async login() {
    if (this.data.loading) return
    this.setData({ loading: true, error: '' })
    try { const result = await api.login(); this.openHome(result.user || {}) }
    catch (error) { this.setData({ error: error.message }) }
    finally { this.setData({ loading: false }) }
  }
})
