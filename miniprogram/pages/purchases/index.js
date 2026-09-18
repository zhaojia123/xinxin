const listPage = require('../../utils/list')
const api = require('../../utils/api')
const { today } = require('../../utils/form')
Page({
  ...listPage({
    endpoint: '/purchases', tab: 'purchases', module: 'purchases',
    initial: { start_date: today(), end_date: today(), totals: [] },
    params: d => ({ start_date: d.start_date, end_date: d.end_date }),
    decorate: v => v
  }),
  dateChanged(e) {
    const field = e.currentTarget.dataset.field
    const value = e.detail.value
    const data = { [field]: value, totals: [] }
    if (field === 'start_date' && value > this.data.end_date) data.end_date = value
    if (field === 'end_date' && value < this.data.start_date) data.start_date = value
    this.setData(data)
    this.load(true)
  },
  today() { const value = today(); this.setData({ start_date: value, end_date: value }); this.load(true) },
  add() { wx.navigateTo({ url: '/pages/purchases-edit/index' }) },
  edit(e) { wx.navigateTo({ url: '/pages/purchases-edit/index?id=' + e.currentTarget.dataset.id }) },
  remove(e) {
    const id = e.currentTarget.dataset.id
    wx.showModal({ title: '删除采购明细', content: '删除后只会从列表隐藏，不会物理删除。确定继续吗？', success: async result => {
      if (!result.confirm) return
      try { await api.request('/purchases?id=' + id, { method: 'DELETE' }); wx.showToast({ title: '已删除' }); this.load(true) }
      catch (error) { api.errorModal(error) }
    } })
  },
  async exportExcel() {
    try {
      wx.showLoading({ title: '正在生成' })
      const path = await api.downloadFile('/purchases/export?' + api.query({ start_date: this.data.start_date, end_date: this.data.end_date }))
      wx.hideLoading()
      wx.openDocument({ filePath: path, fileType: 'xls', showMenu: true, fail: () => api.errorModal(new Error('文件已下载，但当前微信无法打开，请在文件菜单中查看')) })
    } catch (error) { wx.hideLoading(); api.errorModal(error) }
  }
})
