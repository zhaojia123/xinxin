const listPage = require('../../utils/list')
const api = require('../../utils/api')
const { today } = require('../../utils/form')
Page({
  ...listPage({
    endpoint: '/ledger', tab: 'ledger', module: 'ledger',
    initial: { start_date: today().slice(0, 8) + '01', end_date: today(), direction: '', summary: null },
    params: d => ({ start_date: d.start_date, end_date: d.end_date, direction: d.direction, q: d.q }),
    decorate: v => ({ ...v, sign: v.direction === 'income' ? '+' : '−', badge: v.direction === 'income' ? '收' : '支' })
  }),
  dateChanged(e) {
    const field = e.currentTarget.dataset.field
    const value = e.detail.value
    const data = { [field]: value, summary: null }
    if (field === 'start_date' && value > this.data.end_date) data.end_date = value
    if (field === 'end_date' && value < this.data.start_date) data.start_date = value
    this.setData(data)
    this.load(true)
  },
  currentMonth() {
    const date = new Date()
    const start = `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}-01`
    const endDate = new Date(date.getFullYear(), date.getMonth() + 1, 0)
    const end = `${endDate.getFullYear()}-${String(endDate.getMonth() + 1).padStart(2, '0')}-${String(endDate.getDate()).padStart(2, '0')}`
    this.setData({ start_date: start, end_date: end, summary: null })
    this.load(true)
  },
  filter(e) { this.setData({ direction: e.currentTarget.dataset.value }); this.load(true) },
  add() { wx.navigateTo({ url: '/pages/ledger-edit/index' }) },
  detail(e) { wx.navigateTo({ url: '/pages/ledger-detail/index?id=' + e.currentTarget.dataset.id }) },
  edit(e) { wx.navigateTo({ url: '/pages/ledger-edit/index?id=' + e.currentTarget.dataset.id }) },
  remove(e) {
    const id = e.currentTarget.dataset.id
    wx.showModal({ title: '删除台账', content: '删除后将从列表隐藏，但不会物理删除数据。确定继续吗？', success: async res => {
      if (!res.confirm) return
      try { await api.request('/ledger?id=' + id, { method: 'DELETE' }); wx.showToast({ title: '已删除' }); this.load(true) }
      catch (error) { api.errorModal(error) }
    } })
  }
})
