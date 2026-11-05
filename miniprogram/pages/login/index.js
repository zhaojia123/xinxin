const api = require('../../utils/api')
const config = require('../../config')
function hasPermission(user, key) {
  if (user.actions && user.actions[key] && user.actions[key].view !== undefined) return user.actions[key].view === true
  if (Array.isArray(user.permissions)) return user.permissions.indexOf(key) >= 0
  return user['can_' + key] === true
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
    // 小程序仅提供员工档案和请假/调休登记，入口与后台共用同一权限数据。
    const target = hasPermission(user, 'employees') ? '/pages/employees/index' : hasPermission(user, 'attendance') ? '/pages/attendance/index' : ''
    if (!target) {
      this.setData({ error: '当前账号尚未分配员工档案或请假管理权限，请联系管理员' })
      return
    }
    wx.switchTab({ url: target, fail: () => this.setData({ error: '登录成功，但首页跳转失败，请重新打开小程序后再试' }) })
  },
  async login() {
    if (this.data.loading) return
    this.setData({ loading: true, error: '' })
    try { const result = await api.login(); this.openHome(result.user || {}) }
    catch (error) { this.setData({ error: error.message }) }
    finally { this.setData({ loading: false }) }
  }
})
