const api = require('../../utils/api')
const formUtil = require('../../utils/form')
const basicFields = [
  { key: 'name', label: '姓名', placeholder: '填写员工姓名', required: true, max: 64 },
  { key: 'employee_no', label: '员工编号', placeholder: '例如：YG001，不可重复', required: true, max: 32 },
  { key: 'mobile', label: '手机号', placeholder: '选填，联系电话', max: 32 },
  { key: 'id_card', label: '身份证号', placeholder: '选填，完整身份证号', max: 32 }
]
const selections = {
  gender: [{ value: 'unknown', name: '未填写' }, { value: 'female', name: '女' }, { value: 'male', name: '男' }],
  employment_status: [{ value: 'probation', name: '试用期' }, { value: 'active', name: '在职' }, { value: 'left', name: '已离职' }],
  employment_type: [{ value: 'full_time', name: '正式员工' }, { value: 'part_time', name: '兼职' }, { value: 'intern', name: '实习' }]
}
Page({
  data: {
    id: 0, ready: false, saving: false, error: '', basicFields, selections,
    choiceFields: [{ key: 'gender', label: '性别' }, { key: 'employment_status', label: '在职状态' }, { key: 'employment_type', label: '用工类型' }],
    dates: [{ key: 'joined_on', label: '入职日期' }, { key: 'regularized_on', label: '转正日期' }, { key: 'left_on', label: '离职日期（离职时必填）' }],
    form: { employee_no: '', name: '', gender: 'unknown', id_card: '', mobile: '', department_id: 0, position_id: 0, employment_status: 'probation', employment_type: 'full_time', joined_on: '', regularized_on: '', left_on: '', current_salary: '0.00', education: '', hometown: '', remark: '' },
    departments: [], positions: [], labels: {}
  },
  onLoad(query) {
    if (!api.guard()) return
    this.setData({ id: Number(query.id) || 0, 'form.joined_on': formUtil.today() })
    wx.setNavigationBarTitle({ title: this.data.id ? '编辑员工' : '新增员工' })
    this.load()
  },
  onUnload() { this.destroyed = true },
  async load() {
    this.setData({ error: '', ready: false })
    try {
      const [options, employee] = await Promise.all([api.request('/options'), this.data.id ? api.request('/employees?id=' + this.data.id) : Promise.resolve(null)])
      if (this.destroyed) return
      this.options = options
      if (employee) {
        const form = {}; Object.keys(this.data.form).forEach(key => { form[key] = employee[key] })
        this.setData({ form })
      }
      this.syncOptions(); this.setData({ ready: true })
    } catch (error) { if (!this.destroyed) this.setData({ error: error.message }) }
  },
  syncOptions() {
    const departments = [{ id: 0, name: '不指定部门' }, ...this.options.departments]
    const positions = [{ id: 0, name: '不指定岗位' }, ...this.options.positions.filter(v => v.department_id === this.data.form.department_id)]
    const labels = {}
    Object.keys(selections).forEach(key => { labels[key] = (selections[key].find(v => v.value === this.data.form[key]) || {}).name || '请选择' })
    this.setData({ departments, positions, labels, departmentName: formUtil.pick(departments, this.data.form.department_id), positionName: formUtil.pick(positions, this.data.form.position_id) })
  },
  input: formUtil.input,
  choice(e) {
    if (!this.data.ready || this.data.saving) return
    const key = e.currentTarget.dataset.field
    const value = selections[key][Number(e.detail.value)].value
    this.setData({ ['form.' + key]: value })
    if (key === 'employment_status' && value !== 'left') this.setData({ 'form.left_on': '' })
    this.syncOptions(); formUtil.dirty(this)
  },
  select(e) {
    if (!this.data.ready || this.data.saving) return
    const { kind, field } = e.currentTarget.dataset
    const value = this.data[kind][Number(e.detail.value)]
    if (!value) return
    this.setData({ ['form.' + field]: value.id })
    if (field === 'department_id') this.setData({ 'form.position_id': 0 })
    this.syncOptions(); formUtil.dirty(this)
  },
  clearDate(e) {
    if (!this.data.ready || this.data.saving) return
    this.setData({ ['form.' + e.currentTarget.dataset.field]: '' }); formUtil.dirty(this)
  },
  create(e) {
    const kind = e.currentTarget.dataset.kind
    if (kind === 'positions' && !this.data.form.department_id) { api.errorModal(new Error('请先选择岗位所属部门')); return }
    formUtil.createOption(this, kind, kind === 'departments' ? '新建部门' : '新建岗位', { department_id: this.data.form.department_id })
  },
  async save() {
    if (!this.data.ready || this.data.saving) return
    const form = { ...this.data.form, name: this.data.form.name.trim(), employee_no: this.data.form.employee_no.trim() }
    let message = ''
    if (!form.name || !form.employee_no) message = '请填写姓名和员工编号'
    else if (!formUtil.moneyValid(form.current_salary, false, 10)) message = '基本工资不能为负数，最多2位小数'
    else if (form.employment_status === 'left' && !form.left_on) message = '离职员工请填写离职日期'
    else if (form.joined_on && ((form.regularized_on && form.regularized_on < form.joined_on) || (form.left_on && form.left_on < form.joined_on))) message = '转正或离职日期不能早于入职日期'
    if (message) { api.errorModal(new Error(message)); return }
    this.setData({ saving: true })
    try {
      await api.request('/employees' + (this.data.id ? '?id=' + this.data.id : ''), { method: this.data.id ? 'PUT' : 'POST', data: form })
      formUtil.finished(this, '/pages/employees/index')
    } catch (error) { api.errorModal(error) }
    finally { this.setData({ saving: false }) }
  },
  back() { formUtil.back('/pages/employees/index') }
})
