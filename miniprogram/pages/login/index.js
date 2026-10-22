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
    // 仅能用 switchTab 跳转到当前底部导航中的页面，默认进入员工档案。
    const target = hasPermission(user, 'employees') ? '/pages/employees/index' : hasPermission(user, 'attendance') ? '/pages/attendance/index' : hasPermission(user, 'work_logs') ? '/pages/work-logs/index' : ''
    if (!target) {
      this.setData({ error: '当前账号尚未分配员工档案、请假管理或临时工工时权限，请联系管理员' })
      return
    }
    if (target === '/pages/work-logs/index') {
      wx.reLaunch({ url: target, fail: () => this.setData({ error: '登录成功，但首页跳转失败，请重新打开小程序后再试' }) })
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
