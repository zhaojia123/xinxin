const listPage = require('../../utils/list')
const { today } = require('../../utils/form')
Page({
  ...listPage({
    endpoint: '/ledger', tab: 0,
    initial: { month: today().slice(0, 7), direction: '', summary: null },
    params: d => ({ month: d.month, direction: d.direction, q: d.q }),
    decorate: v => ({ ...v, sign: v.direction === 'income' ? '+' : '−', badge: v.direction === 'income' ? '收' : '支' })
  }),
  monthChanged(e) { this.setData({ month: e.detail.value, summary: null }); this.load(true) },
  filter(e) { this.setData({ direction: e.currentTarget.dataset.value }); this.load(true) },
  add() { wx.navigateTo({ url: '/pages/ledger-edit/index' }) },
  edit(e) { wx.navigateTo({ url: '/pages/ledger-edit/index?id=' + e.currentTarget.dataset.id }) }
})
