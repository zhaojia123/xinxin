const api = require('../../utils/api')

const TYPES = [
  { name: '公休', category: 'public_rest', recordType: '公休' },
  { name: '请假', category: 'leave', recordType: '请假' },
  { name: '法定节假日', category: 'holiday', recordType: '节假日' },
  { name: '特殊请假补贴', category: 'leave', recordType: '特殊请假' }
]

function pad(value) { return String(value).padStart(2, '0') }
function today() {
  const date = new Date()
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}`
}
function currentMonth() { return today().slice(0, 7) }

Page({
  data: {
    month: currentMonth(), records: [], employees: [], employeeNames: [], selectedEmployees: [], dates: [], formDate: today(),
    types: TYPES, typeIndex: 0, duration: '1', subsidy: '', reason: '', showForm: false, editingID: 0,
    q: '', typeFilter: '', loading: false, saving: false, loaded: false, error: '', canCreate: false, canEdit: false, canDelete: false
  },
  onShow() {
    if (!api.guard()) return
    this.setData({ canCreate: api.hasPermission('attendance', 'create'), canEdit: api.hasPermission('attendance', 'edit'), canDelete: api.hasPermission('attendance', 'delete') })
    const bar = this.getTabBar && this.getTabBar()
    if (bar) { const index = bar.data.tabs ? bar.data.tabs.findIndex(item => item.key === 'attendance') : 1; bar.setData({ selected: index < 0 ? 0 : index }) }
    this.load()
  },
  onPullDownRefresh() { this.load().finally(() => wx.stopPullDownRefresh()) },
  async load(overrides = {}) {
    this.setData({ loading: true, error: '' })
    try {
      const state = { ...this.data, ...overrides }
      const [result, employees] = await Promise.all([
        api.request('/attendance?' + api.query({ month: state.month, q: state.q, type: state.typeFilter })),
        // 临时工按每日工时结算，不走请假扣薪；请假管理只选择在职正式工。
        this.data.employees.length ? Promise.resolve({ records: this.data.employees }) : api.request('/employees?page=1&status=active&employment_type=full_time')
      ])
      const list = (employees.records || []).map(item => ({ ...item, selected: state.selectedEmployees.indexOf(Number(item.id)) >= 0 }))
      this.setData({ records: result.records || [], employees: list, loading: false, loaded: true })
    } catch (error) { this.setData({ loading: false, loaded: true, error: error.message }) }
  },
  monthChange(e) { const month = e.detail.value.slice(0, 7); this.setData({ month }); this.load({ month }) },
  search(e) { const q = e.detail.value; clearTimeout(this.searchTimer); this.setData({ q }); this.searchTimer = setTimeout(() => this.load({ q }), 350) },
  filterType(e) { const typeFilter = e.currentTarget.dataset.value; this.setData({ typeFilter }); this.load({ typeFilter }) },
  add() {
    if (!this.data.canCreate) { api.errorModal(new Error('当前账号没有新增请假记录权限')); return }
    this.setData({ showForm: true, editingID: 0, selectedEmployees: [], employeeNames: [], employees: this.data.employees.map(item => ({ ...item, selected: false })), dates: [], formDate: today(), typeIndex: 0, duration: '1', subsidy: '', reason: '', error: '' })
  },
  cancel() { this.setData({ showForm: false, editingID: 0 }) },
  employeeChange(e) {
    const ids = (e.detail.value || []).map(Number)
    const names = ids.map(id => { const item = this.data.employees.find(employee => Number(employee.id) === id); return item ? item.name : '' }).filter(Boolean)
    this.setData({ selectedEmployees: ids, employeeNames: names, employees: this.data.employees.map(item => ({ ...item, selected: ids.indexOf(Number(item.id)) >= 0 })) })
  },
  dateChange(e) { this.setData({ formDate: e.detail.value }) },
  addDate() {
    if (!this.data.formDate) return
    if (this.data.dates.indexOf(this.data.formDate) >= 0) { wx.showToast({ title: '该日期已添加', icon: 'none' }); return }
    this.setData({ dates: this.data.dates.concat(this.data.formDate).sort() })
  },
  removeDate(e) { this.setData({ dates: this.data.dates.filter(value => value !== e.currentTarget.dataset.date) }) },
  typeChange(e) { this.setData({ typeIndex: Number(e.detail.value), subsidy: Number(e.detail.value) === 3 ? this.data.subsidy : '' }) },
  durationChange(e) { this.setData({ duration: e.detail.value }) },
  subsidyChange(e) { this.setData({ subsidy: e.detail.value }) },
  reasonChange(e) { this.setData({ reason: e.detail.value }) },
  async edit(e) {
    if (!this.data.canEdit) { api.errorModal(new Error('当前账号没有编辑请假记录权限')); return }
    try {
      const value = await api.request('/attendance?id=' + e.currentTarget.dataset.id)
      const index = Math.max(0, TYPES.findIndex(item => item.recordType === value.record_type))
      const employee = this.data.employees.find(item => Number(item.id) === Number(value.employee_id))
      const selectedID = Number(value.employee_id)
      this.setData({ showForm: true, editingID: value.id, selectedEmployees: [selectedID], employeeNames: employee ? [employee.name] : [], employees: this.data.employees.map(item => ({ ...item, selected: Number(item.id) === selectedID })), dates: [value.occurred_on], formDate: value.occurred_on, typeIndex: index, duration: value.duration_days || '1', subsidy: value.subsidy_amount === '0' ? '' : value.subsidy_amount, reason: value.reason || '', error: '' })
    } catch (error) { api.errorModal(error) }
  },
  async save() {
    if (this.data.saving) return
    const type = TYPES[this.data.typeIndex]
    if (!this.data.selectedEmployees.length) { api.errorModal(new Error('请选择员工')); return }
    if (!this.data.dates.length) { api.errorModal(new Error('请至少添加一个日期')); return }
    const days = Number(this.data.duration)
    if (!Number.isFinite(days) || days <= 0 || Math.round(days * 2) !== days * 2) { api.errorModal(new Error('天数必须是大于0的0.5倍数')); return }
    const subsidy = type.recordType === '特殊请假' ? (this.data.subsidy || '0') : '0'
    const salaryEffect = type.recordType === '公休' || type.recordType === '节假日' ? 'none' : (Number(subsidy) > 0 ? 'deduct_and_subsidy' : 'deduct')
    const body = employeeID => ({ employee_id: employeeID, category: type.category, record_type: type.recordType, occurred_on: this.data.dates[0], duration_days: String(this.data.duration), salary_effect: salaryEffect, subsidy_amount: subsidy, reason: this.data.reason.trim() })
    this.setData({ saving: true })
    try {
      if (this.data.editingID) {
        await api.request('/attendance?id=' + this.data.editingID, { method: 'PUT', data: body(this.data.selectedEmployees[0]) })
      } else {
        const requests = []
        this.data.selectedEmployees.forEach(employeeID => this.data.dates.forEach(date => requests.push(api.request('/attendance', { method: 'POST', data: { ...body(employeeID), occurred_on: date } }))))
        await Promise.all(requests)
      }
      wx.showToast({ title: this.data.editingID ? '已更新' : '已保存' }); this.setData({ showForm: false, editingID: 0 }); this.load()
    } catch (error) { api.errorModal(error) }
    finally { this.setData({ saving: false }) }
  },
  remove(e) {
    if (!this.data.canDelete) { api.errorModal(new Error('当前账号没有删除请假记录权限')); return }
    const id = e.currentTarget.dataset.id
    wx.showModal({ title: '删除记录', content: '删除后该记录不再参与工资计算，确定继续吗？', success: async result => {
      if (!result.confirm) return
      try { await api.request('/attendance?id=' + id, { method: 'DELETE' }); wx.showToast({ title: '已删除' }); this.load() } catch (error) { api.errorModal(error) }
    } })
  }
})
