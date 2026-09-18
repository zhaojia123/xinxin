// visible 控制是否在底部导航展示，隐藏入口不影响页面和接口继续使用。
const ALL_TABS = [{ key: 'ledger', title: '台账', url: '/pages/ledger/index', icon: 'book' }, { key: 'purchases', title: '采购', url: '/pages/purchases/index', icon: 'cart', visible: false }, { key: 'employees', title: '员工管理', url: '/pages/employees/index', icon: 'people' }]
function hasPermission(user, key) {
  if (Array.isArray(user.permissions)) return user.permissions.indexOf(key) >= 0
  return user['can_' + key] !== false
}

Component({
  data: { selected: 0, tabs: [] },
  lifetimes: { attached() { this.refreshTabs() } },
  pageLifetimes: { show() { this.refreshTabs() } },
  methods: {
    refreshTabs() {
      const user = wx.getStorageSync('xinxin_mini_user')
      const tabs = user && Object.keys(user).length ? ALL_TABS.filter(tab => tab.visible !== false && hasPermission(user, tab.key)) : ALL_TABS.filter(tab => tab.visible !== false)
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
