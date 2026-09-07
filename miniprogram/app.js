const config = require('./config')

App({
  globalData: { name: config.name },
  onLaunch() {
    // 仅保存登录令牌，不将员工档案或账目写入本地缓存。
    this.globalData.name = config.name
  }
})
