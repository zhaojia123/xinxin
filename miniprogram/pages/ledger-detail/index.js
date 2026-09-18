const api = require('../../utils/api')

Page({
  data: { id: 0, ready: false, uploading: false, error: '', record: null, canEdit: false },
  onLoad(query) {
    if (!api.guard()) return
    const id = Number(query.id) || 0
    this.setData({ id, canEdit: api.hasPermission('ledger', 'edit'), canDelete: api.hasPermission('ledger', 'delete') })
    wx.setNavigationBarTitle({ title: '台账详情' })
    if (id) this.load()
    else this.setData({ error: '记录ID不正确' })
  },
  onUnload() { this.destroyed = true },
  async load() {
    this.setData({ error: '', ready: false })
    try {
      const record = await api.request('/ledger?id=' + this.data.id)
      record.attachments = (record.attachments || []).map(item => ({ ...item, file_url: item.file_url.startsWith('http') ? item.file_url : api.baseURL() + item.file_url }))
      if (!this.destroyed) this.setData({ record, ready: true })
    } catch (error) { if (!this.destroyed) this.setData({ error: error.message }) }
  },
  edit() { if (!this.data.record.payroll_batch_id) wx.navigateTo({ url: '/pages/ledger-edit/index?id=' + this.data.id }) },
  uploadProof() {
    if (this.data.uploading) return
    const options = {
      count: 9, sourceType: ['album', 'camera'],
      success: result => this.uploadFiles((result.tempFiles || result.tempFilePaths || []).map(v => typeof v === 'string' ? v : v.tempFilePath)),
      fail: error => { if (error.errMsg && !error.errMsg.includes('cancel')) this.setData({ error: '打开相册失败，请重试' }) }
    }
    if (wx.chooseMedia) wx.chooseMedia({ ...options, mediaType: ['image'] })
    else wx.chooseImage(options)
  },
  uploadFiles(paths) {
    if (!paths.length) return
    this.setData({ uploading: true, error: '' })
    Promise.all(paths.map(path => api.uploadFile('/ledger-proof', path, { ledger_id: String(this.data.id) })))
      .then(() => { wx.showToast({ title: '上传成功', icon: 'success' }); this.load() })
      .catch(error => this.setData({ error: error.message }))
      .finally(() => this.setData({ uploading: false }))
  },
  back() { if (getCurrentPages().length > 1) wx.navigateBack(); else wx.switchTab({ url: '/pages/ledger/index' }) }
})
