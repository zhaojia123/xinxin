const api = require('../../utils/api')

function pad(value) { return String(value).padStart(2, '0') }
function today() {
  const value = new Date()
  return `${value.getFullYear()}-${pad(value.getMonth() + 1)}-${pad(value.getDate())}`
}
function currentMonth() { return today().slice(0, 7) }
function workerHint(worker) {
  if (!worker) return '请先在员工档案中新增临时工'
  const rate = worker.current_salary || worker.salary || '0'
  const label = worker.pay_basis === 'daily' ? '按天' : worker.pay_basis === 'half_daily' ? '按半天' : '按小时'
  return `${label} · 计薪标准 ¥${rate}`
}

Page({
  data: {
    month: currentMonth(), q: '', records: [], workers: [], workerIndex: 0,
    dates: [], formDate: today(), hours: '8', remark: '', workerHint: '请先在员工档案中新增临时工', showForm: false, editingID: 0,
    loading: false, saving: false, loaded: false, error: '', canCreate: false, canEdit: false, canDelete: false
  },
  onLoad() { wx.setNavigationBarTitle({ title: '临时工工时' }) },
  onHide() { clearTimeout(this.searchTimer) },
  onUnload() { clearTimeout(this.searchTimer) },
  onShow() {
    if (!api.guard()) return
    this.setData({
      canCreate: api.hasPermission('work_logs', 'create'),
      canEdit: api.hasPermission('work_logs', 'edit'),
      canDelete: api.hasPermission('work_logs', 'delete')
    })
    this.load()
  },
  onPullDownRefresh() { this.load().finally(() => wx.stopPullDownRefresh()) },
  async load(overrides = {}) {
    this.setData({ loading: true, error: '' })
    try {
      const state = { ...this.data, ...overrides }
      const [result, employees] = await Promise.all([
        api.request('/work-logs?' + api.query({ month: state.month, q: state.q })),
        api.request('/employees?' + api.query({ page: 1, status: 'active', employment_type: 'part_time' }))
      ])
      const workers = (employees.records || []).filter(item => ['hourly', 'daily', 'half_daily'].indexOf(item.pay_basis) >= 0)
      const workerIndex = Math.min(this.data.workerIndex, Math.max(0, workers.length - 1))
      this.setData({ records: result.records || [], workers, workerIndex, workerHint: workerHint(workers[workerIndex]), loading: false, loaded: true })
    } catch (error) { this.setData({ loading: false, loaded: true, error: error.message }) }
  },
  monthChange(e) { const month = e.detail.value.slice(0, 7); this.setData({ month }); this.load({ month }) },
  search(e) { const q = e.detail.value; clearTimeout(this.searchTimer); this.setData({ q }); this.searchTimer = setTimeout(() => this.load({ q }), 350) },
  add() {
    if (!this.data.canCreate) { api.errorModal(new Error('当前账号没有登记临时工工时权限')); return }
    this.setData({ showForm: true, editingID: 0, workerIndex: 0, dates: [], formDate: today(), hours: '8', remark: '', error: '' })
  },
  cancel() { this.setData({ showForm: false, editingID: 0 }) },
  workerChange(e) { const workerIndex = Number(e.detail.value); this.setData({ workerIndex, workerHint: workerHint(this.data.workers[workerIndex]) }) },
  dateChange(e) { this.setData({ formDate: e.detail.value }) },
  addDate() {
    const date = this.data.formDate
    if (!date) return
    if (this.data.dates.indexOf(date) >= 0) { wx.showToast({ title: '该日期已添加', icon: 'none' }); return }
    this.setData({ dates: this.data.dates.concat(date).sort() })
  },
  removeDate(e) { this.setData({ dates: this.data.dates.filter(value => value !== e.currentTarget.dataset.date) }) },
  hoursChange(e) { this.setData({ hours: e.detail.value }) },
  preset(e) { this.setData({ hours: String(e.currentTarget.dataset.hours) }) },
  remarkChange(e) { this.setData({ remark: e.detail.value }) },
  async edit(e) {
    if (!this.data.canEdit) { api.errorModal(new Error('当前账号没有编辑临时工工时权限')); return }
    try {
      const value = await api.request('/work-logs?id=' + e.currentTarget.dataset.id)
      const workerIndex = Math.max(0, this.data.workers.findIndex(item => Number(item.id) === Number(value.employee_id)))
      this.setData({ showForm: true, editingID: value.id, workerIndex, workerHint: workerHint(this.data.workers[workerIndex]), dates: [value.work_date], formDate: value.work_date, hours: value.hours_value || value.hours, remark: value.remark || '', error: '' })
    } catch (error) { api.errorModal(error) }
  },
  async save() {
    if (this.data.saving) return
    const worker = this.data.workers[this.data.workerIndex]
    const hours = Number(this.data.hours)
    if (!worker) { api.errorModal(new Error('请选择临时工')); return }
    if (!this.data.dates.length) { api.errorModal(new Error('请至少添加一个工作日期')); return }
    if (!Number.isFinite(hours) || hours <= 0 || hours > 24) { api.errorModal(new Error('当天工时必须大于0且不超过24小时')); return }
    if (this.data.editingID && this.data.dates.length !== 1) { api.errorModal(new Error('编辑单条记录时只能保留一个工作日期')); return }
    const body = { employee_id: Number(worker.id), work_dates: this.data.dates, work_date: this.data.dates[0], hours: String(this.data.hours), remark: this.data.remark.trim() }
    this.setData({ saving: true })
    try {
      await api.request('/work-logs' + (this.data.editingID ? '?id=' + this.data.editingID : ''), { method: this.data.editingID ? 'PUT' : 'POST', data: body })
      wx.showToast({ title: this.data.editingID ? '已更新' : '已保存' })
      this.setData({ showForm: false, editingID: 0 })
      this.load()
    } catch (error) { api.errorModal(error) }
    finally { this.setData({ saving: false }) }
  },
  remove(e) {
    if (!this.data.canDelete) { api.errorModal(new Error('当前账号没有删除临时工工时权限')); return }
    const id = e.currentTarget.dataset.id
    wx.showModal({ title: '删除工时', content: '删除后该日工时不再参与本月工资，确定继续吗？', success: async result => {
      if (!result.confirm) return
      try { await api.request('/work-logs?id=' + id, { method: 'DELETE' }); wx.showToast({ title: '已删除' }); this.load() }
      catch (error) { api.errorModal(error) }
    } })
  }
})
