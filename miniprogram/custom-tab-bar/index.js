// visible 控制是否在底部导航展示，隐藏入口不影响页面和接口继续使用。
const ALL_TABS = [{ key: 'employees', title: '员工档案', url: '/pages/employees/index', icon: 'people' }, { key: 'attendance', title: '请假管理', url: '/pages/attendance/index', icon: 'calendar' }]
function hasPermission(user, key) {
  if (user.actions && user.actions[key] && user.actions[key].view !== undefined) return user.actions[key].view === true
  if (Array.isArray(user.permissions)) return user.permissions.indexOf(key) >= 0
  // 权限数据没有返回时不展示入口，不能以旧缓存默认放行。
  return user['can_' + key] === true
}

Component({
  data: { selected: 0, tabs: [] },
  lifetimes: { attached() { this.refreshTabs() } },
  pageLifetimes: { show() { this.refreshTabs() } },
  methods: {
    refreshTabs() {
      const user = wx.getStorageSync('xinxin_mini_user')
      const tabs = user && Object.keys(user).length ? ALL_TABS.filter(tab => tab.visible !== false && hasPermission(user, tab.key)) : []
      const selected = Math.max(0, Math.min(this.data.selected, tabs.length - 1))
      this.setData({ tabs, selected })
    },
    switchTab(e) {
      const index = Number(e.currentTarget.dataset.index)
      if (index === this.data.selected) return
      wx.switchTab({ url: this.data.tabs[index].url })
    }
  }
})
