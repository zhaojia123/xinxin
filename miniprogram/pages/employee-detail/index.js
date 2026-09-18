const api = require('../../utils/api')

Page({
  data: { id: 0, ready: false, error: '', employee: null, canEdit: false },
  onLoad(query) {
    if (!api.guard()) return
    const id = Number(query.id) || 0
    this.setData({ id, canEdit: api.hasPermission('employees', 'edit'), canDelete: api.hasPermission('employees', 'delete') })
    wx.setNavigationBarTitle({ title: '员工详情' })
    if (id) this.load()
    else this.setData({ error: '员工ID不正确' })
  },
  onUnload() { this.destroyed = true },
  async load() {
    this.setData({ error: '', ready: false })
    try {
      const employee = await api.request('/employees?id=' + this.data.id)
      employee.attachments = (employee.attachments || []).map(item => ({ ...item, file_url: item.file_url.startsWith('http') ? item.file_url : api.baseURL() + item.file_url }))
      employee.initial = employee.name ? employee.name.slice(0, 1) : '人'
      employee.status_label = { active: '在职', probation: '试用期', left: '已离职' }[employee.employment_status] || '未填写'
      employee.employment_type_label = { full_time: '正式员工', part_time: '兼职', intern: '实习' }[employee.employment_type] || '未填写'
      employee.gender_label = { male: '男', female: '女', unknown: '未填写' }[employee.gender] || '未填写'
      if (!this.destroyed) this.setData({ employee, ready: true })
    } catch (error) { if (!this.destroyed) this.setData({ error: error.message }) }
  },
  uploadDocument(e) {
    const kind = e.currentTarget.dataset.kind
    if (this.data.uploadingType) return
    const options = {
      count: 1, sourceType: ['album', 'camera'],
      success: result => {
        const path = (result.tempFiles || result.tempFilePaths || []).map(v => typeof v === 'string' ? v : v.tempFilePath)[0]
        if (!path) return
        this.setData({ uploadingType: kind, error: '' })
        api.uploadFile('/employee-document', path, { employee_id: String(this.data.id), attachment_type: kind }, 'document')
          .then(() => { wx.showToast({ title: '上传成功', icon: 'success' }); this.load() })
          .catch(error => this.setData({ error: error.message }))
          .finally(() => this.setData({ uploadingType: '' }))
      },
      fail: error => { if (error.errMsg && !error.errMsg.includes('cancel')) this.setData({ error: '打开相册失败，请重试' }) }
    }
    if (wx.chooseMedia) wx.chooseMedia({ ...options, mediaType: ['image'] })
    else wx.chooseImage(options)
  },
  edit() { wx.navigateTo({ url: '/pages/employee-edit/index?id=' + this.data.id }) },
  back() { if (getCurrentPages().length > 1) wx.navigateBack(); else wx.switchTab({ url: '/pages/employees/index' }) }
})
