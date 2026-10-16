const api = require('../../utils/api')
const formUtil = require('../../utils/form')
const basicFields = [
  { key: 'name', label: '姓名', placeholder: '填写员工姓名', required: true, max: 64 },
  { key: 'employee_no', label: '员工编号', placeholder: '留空由系统自动生成', required: false, max: 32 },
  { key: 'mobile', label: '手机号', placeholder: '选填，联系电话', max: 32 },
  { key: 'id_card', label: '身份证号', placeholder: '选填，完整身份证号', max: 32 }
]
const selections = {
  gender: [{ value: 'unknown', name: '未填写' }, { value: 'female', name: '女' }, { value: 'male', name: '男' }],
  employment_status: [{ value: 'probation', name: '试用期' }, { value: 'active', name: '在职' }, { value: 'left', name: '已离职' }],
  employment_type: [{ value: 'full_time', name: '正式员工' }, { value: 'part_time', name: '兼职' }, { value: 'intern', name: '实习' }],
  pay_basis: [{ value: 'monthly', name: '月薪' }, { value: 'daily', name: '日薪' }, { value: 'hourly', name: '时薪' }, { value: 'half_daily', name: '半天工资' }]
}
const temporaryPayBases = [{ value: 'hourly', name: '按小时' }, { value: 'daily', name: '按天' }, { value: 'half_daily', name: '按半天' }]
const documentKinds = [
  { kind: 'id_card_front', label: '身份证人像面' },
  { kind: 'id_card_back', label: '身份证国徽面' },
  { kind: 'health_certificate_front', label: '健康证正面' },
  { kind: 'health_certificate_back', label: '健康证反面' }
]
Page({
  data: {
    id: 0, temporary: false, ready: false, saving: false, uploading: false, error: '', basicFields, visibleBasicFields: basicFields, selections, temporaryPayBases, documentKinds, pendingDocuments: [], selectedDocuments: {},
    choiceFields: [{ key: 'gender', label: '性别' }, { key: 'employment_status', label: '在职状态' }, { key: 'employment_type', label: '用工类型' }, { key: 'pay_basis', label: '计薪方式' }], visibleChoiceFields: [{ key: 'gender', label: '性别' }, { key: 'employment_status', label: '在职状态' }, { key: 'employment_type', label: '用工类型' }, { key: 'pay_basis', label: '计薪方式' }],
    dates: [{ key: 'joined_on', label: '入职日期' }, { key: 'regularized_on', label: '转正日期' }, { key: 'left_on', label: '离职日期（离职时必填）' }], visibleDates: [{ key: 'joined_on', label: '入职日期' }, { key: 'regularized_on', label: '转正日期' }, { key: 'left_on', label: '离职日期（离职时必填）' }],
    form: { employee_no: '', name: '', gender: 'unknown', id_card: '', mobile: '', department_id: 0, position_id: 0, employment_status: 'probation', employment_type: 'full_time', pay_basis: 'monthly', joined_on: '', regularized_on: '', left_on: '', entry_salary: '', current_salary: '', education: '', hometown: '', remark: '' },
    departments: [], positions: [], labels: {}
  },
  onLoad(query) {
    if (!api.guard()) return
    const id = Number(query.id) || 0
    const temporary = query.temporary === '1'
    this.setTemporaryView(temporary)
    this.setData({ id, temporary, 'form.joined_on': formUtil.today(), ...(temporary ? { 'form.gender': 'unknown', 'form.employment_status': 'active', 'form.employment_type': 'part_time', 'form.pay_basis': 'hourly' } : {}) })
    if (!api.hasPermission('employees', id ? 'edit' : 'create')) {
      wx.showModal({ title: '无操作权限', content: '当前账号只有查看权限，不能编辑或新增员工。', showCancel: false, success: () => wx.navigateBack() })
      return
    }
    wx.setNavigationBarTitle({ title: this.data.id ? '编辑员工' : (temporary ? '新增临时工' : '新增正式工') })
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
        const temporary = employee.employment_type === 'part_time'
        this.setTemporaryView(temporary)
        this.setData({ form, temporary })
      }
      this.syncOptions(); this.setData({ ready: true })
    } catch (error) { if (!this.destroyed) this.setData({ error: error.message }) }
  },
  setTemporaryView(temporary) {
    const visibleBasicFields = temporary ? basicFields.filter(item => item.key !== 'employee_no') : basicFields
    const visibleChoiceFields = temporary ? [{ key: 'pay_basis', label: '记薪方式' }] : this.data.choiceFields
    const visibleDates = temporary ? [{ key: 'joined_on', label: '入职日期' }, { key: 'left_on', label: '离职日期（选填）' }] : this.data.dates
    this.setData({ visibleBasicFields, visibleChoiceFields, visibleDates })
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
    const choices = this.data.temporary && key === 'pay_basis' ? temporaryPayBases : selections[key]
    const value = choices[Number(e.detail.value)].value
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
  chooseDocument(e) {
    const kind = e.currentTarget.dataset.kind
    const options = {
      count: 1,
      sourceType: ['album', 'camera'],
      success: result => {
        const item = (result.tempFiles || result.tempFilePaths || [])[0]
        const path = typeof item === 'string' ? item : item && item.tempFilePath
        if (!path) return
        const pendingDocuments = this.data.pendingDocuments.filter(document => document.kind !== kind).concat({ kind, path })
        this.setData({ pendingDocuments, ['selectedDocuments.' + kind]: true })
      },
      fail: error => { if (!error.errMsg || !error.errMsg.includes('cancel')) this.setData({ error: '打开相册失败，请重试' }) }
    }
    if (wx.chooseMedia) wx.chooseMedia({ ...options, mediaType: ['image'] })
    else wx.chooseImage(options)
  },
  async save() {
    if (!this.data.ready || this.data.saving) return
    const form = { ...this.data.form, name: this.data.form.name.trim(), employee_no: this.data.form.employee_no.trim() }
    if (this.data.temporary) {
      form.gender = 'unknown'; form.employment_status = form.left_on ? 'left' : 'active'; form.employment_type = 'part_time'; form.department_id = 0; form.position_id = 0; form.monthly_rest_days = 0
    }
    let message = ''
    if (!form.name) message = '请填写姓名'
    else if ((form.entry_salary && !formUtil.moneyValid(form.entry_salary, false, 10)) || (form.current_salary && !formUtil.moneyValid(form.current_salary, false, 10))) message = '工资不能为负数，最多2位小数'
    else if (form.employment_status === 'left' && !form.left_on) message = '离职员工请填写离职日期'
    else if (form.joined_on && ((form.regularized_on && form.regularized_on < form.joined_on) || (form.left_on && form.left_on < form.joined_on))) message = '转正或离职日期不能早于入职日期'
    if (message) { api.errorModal(new Error(message)); return }
    this.setData({ saving: true })
    try {
      const saved = await api.request('/employees' + (this.data.id ? '?id=' + this.data.id : ''), { method: this.data.id ? 'PUT' : 'POST', data: form })
      const employeeID = saved.id || this.data.id
      if (this.data.pendingDocuments.length) {
        this.setData({ uploading: true })
        try {
          await Promise.all(this.data.pendingDocuments.map(document => api.uploadFile('/employee-document', document.path, { employee_id: String(employeeID), attachment_type: document.kind }, 'document')))
        } catch (error) {
          wx.showModal({ title: '员工已保存', content: '证件图片上传失败，请进入详情页重新上传。', showCancel: false, success: () => wx.redirectTo({ url: '/pages/employee-detail/index?id=' + employeeID }) })
          return
        }
      }
      formUtil.finished(this, '/pages/employees/index')
    } catch (error) { api.errorModal(error) }
    finally { this.setData({ saving: false, uploading: false }) }
  },
  back() { formUtil.back('/pages/employees/index') }
})
