const api = require('./api')

// 切换筛选时仅接收最新请求，避免旧响应覆盖新结果。
function listPage({ endpoint, tab, module, initial, params, decorate = v => v }) {
  return {
    data: { records: [], loading: false, error: '', more: false, page: 1, q: '', loaded: false, ...initial },
    onShow() {
      if (!api.guard()) return
      const bar = this.getTabBar && this.getTabBar()
      if (module) this.setData({ canCreate: api.hasPermission(module, 'create'), canEdit: api.hasPermission(module, 'edit'), canDelete: api.hasPermission(module, 'delete') })
      if (bar) {
        const selected = typeof tab === 'string' && bar.data.tabs ? bar.data.tabs.findIndex(item => item.key === tab) : tab
        bar.setData({ selected: selected < 0 ? 0 : selected })
      }
      this.load(true)
    },
    onHide() { clearTimeout(this.searchTimer); this.serial = (this.serial || 0) + 1; this.setData({ loading: false }) },
    onUnload() { clearTimeout(this.searchTimer); this.serial = (this.serial || 0) + 1 },
    onPullDownRefresh() { this.load(true).finally(() => wx.stopPullDownRefresh()) },
    onReachBottom() { if (this.data.more && !this.data.loading) this.load(false) },
    async load(reset = true) {
      if (!api.guard()) return
      if (!reset && this.data.loading) return
      const serial = this.serial = (this.serial || 0) + 1
      const page = reset ? 1 : this.data.page + 1
      this.setData({ loading: true, error: '', ...(reset ? { records: [], more: false, loaded: false } : {}) })
      try {
        const result = await api.request(endpoint + '?' + api.query({ ...params(this.data), page }))
        if (serial !== this.serial) return
        const records = (result.records || []).map(decorate)
        this.setData({ records: reset ? records : this.data.records.concat(records), summary: result.summary, totals: result.totals || this.data.totals || [], reminders: result.reminders || this.data.reminders || [], more: result.has_more, page, loaded: true })
      } catch (error) { if (serial === this.serial) this.setData({ error: error.message }) }
      finally { if (serial === this.serial) this.setData({ loading: false }) }
    },
    search(e) {
      this.setData({ q: e.detail.value })
      clearTimeout(this.searchTimer)
      // 立即失效旧请求，输入结束后再查询。
      this.serial = (this.serial || 0) + 1
      this.searchTimer = setTimeout(() => this.load(true), 350)
    },
    retry() { this.load(!this.data.loaded) },
    logout() {
      wx.showModal({ title: '退出登录', content: '账目和员工档案仍保存在服务器，重新登录即可查看。', success: res => { if (res.confirm) api.toLogin() } })
    }
  }
}
module.exports = listPage
