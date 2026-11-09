const api = require('../../utils/api')
const config = require('../../config')

Page({
  data: { hasLogin: false, filingNumber: config.filingNumber },
  onShow() { this.setData({ hasLogin: Boolean(api.token()) }) },
  browseEmployees() { wx.navigateTo({ url: '/pages/login/index?from=employees' }) },
  browseAttendance() { wx.navigateTo({ url: '/pages/login/index?from=attendance' }) },
  enterWorkspace() {
    if (api.token()) {
      wx.switchTab({ url: '/pages/employees/index' })
      return
    }
    wx.navigateTo({ url: '/pages/login/index' })
  }
})
