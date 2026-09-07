Component({
  data: { selected: 0, tabs: [{ title: '台账', url: '/pages/ledger/index', icon: 'book' }, { title: '员工', url: '/pages/employees/index', icon: 'people' }] },
  methods: {
    switchTab(e) {
      const index = Number(e.currentTarget.dataset.index)
      if (index === this.data.selected) return
      wx.switchTab({ url: this.data.tabs[index].url })
    }
  }
})
