const listPage = require('../../utils/list')
Page({
  ...listPage({ endpoint: '/employees', tab: 1, initial: { status: '', summary: null }, params: d => ({ status: d.status, q: d.q }), decorate: v => ({ ...v, initial: v.name.slice(0, 1) }) }),
  filter(e) { this.setData({ status: e.currentTarget.dataset.value }); this.load(true) },
  add() { wx.navigateTo({ url: '/pages/employee-edit/index' }) },
  edit(e) { wx.navigateTo({ url: '/pages/employee-edit/index?id=' + e.currentTarget.dataset.id }) }
})
