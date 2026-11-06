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
      employee.initial = employee.name ? employee.name.slice(0, 1) : '人'
      employee.status_label = { active: '在职', probation: '试用期', left: '已离职' }[employee.employment_status] || '未填写'
      employee.employment_type_label = { full_time: '正式员工', part_time: '兼职', intern: '实习' }[employee.employment_type] || '未填写'
      employee.gender_label = { male: '男', female: '女', unknown: '未填写' }[employee.gender] || '未填写'
      if (!this.destroyed) this.setData({ employee, ready: true })
    } catch (error) { if (!this.destroyed) this.setData({ error: error.message }) }
  },
  edit() { wx.navigateTo({ url: '/pages/employee-edit/index?id=' + this.data.id }) },
  back() { if (getCurrentPages().length > 1) wx.navigateBack(); else wx.switchTab({ url: '/pages/employees/index' }) }
})
