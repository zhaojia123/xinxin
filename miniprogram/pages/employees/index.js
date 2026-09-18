const listPage = require('../../utils/list')
const api = require('../../utils/api')
Page({
  ...listPage({ endpoint: '/employees', tab: 'employees', module: 'employees', initial: { status: '', summary: null }, params: d => ({ status: d.status, q: d.q }), decorate: v => ({ ...v, initial: v.name.slice(0, 1) }) }),
  filter(e) { this.setData({ status: e.currentTarget.dataset.value }); this.load(true) },
  add() { wx.navigateTo({ url: '/pages/employee-edit/index' }) },
  detail(e) { wx.navigateTo({ url: '/pages/employee-detail/index?id=' + e.currentTarget.dataset.id }) },
  edit(e) { wx.navigateTo({ url: '/pages/employee-edit/index?id=' + e.currentTarget.dataset.id }) },
  remove(e) {
    const id = e.currentTarget.dataset.id
    wx.showModal({ title: '删除员工', content: '删除后将从列表隐藏，但不会物理删除档案。确定继续吗？', success: async res => {
      if (!res.confirm) return
      try { await api.request('/employees?id=' + id, { method: 'DELETE' }); wx.showToast({ title: '已删除' }); this.load(true) }
      catch (error) { api.errorModal(error) }
    } })
  }
})
