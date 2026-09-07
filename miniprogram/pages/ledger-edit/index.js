const api = require('../../utils/api')
const formUtil = require('../../utils/form')
const fields = ['occurred_on', 'account_id', 'category_id', 'department_id', 'direction', 'amount', 'summary', 'counterparty', 'voucher_no', 'remark']
Page({
  data: { id: 0, ready: false, saving: false, error: '', locked: false, form: { occurred_on: '', account_id: 0, category_id: 0, department_id: 0, direction: 'expense', amount: '', summary: '', counterparty: '', voucher_no: '', remark: '' }, accounts: [], categories: [], departments: [] },
  onLoad(query) {
    if (!api.guard()) return
    this.setData({ id: Number(query.id) || 0, 'form.occurred_on': formUtil.today() })
    wx.setNavigationBarTitle({ title: this.data.id ? '编辑收支' : '记一笔' })
    this.load()
  },
  onUnload() { this.destroyed = true },
  async load() {
    this.setData({ error: '', ready: false })
    try {
      const results = await Promise.all([api.request('/options'), this.data.id ? api.request('/ledger?id=' + this.data.id) : Promise.resolve(null)])
      if (this.destroyed) return
      this.options = results[0]
      if (results[1]) {
        const form = {}; fields.forEach(key => { form[key] = results[1][key] })
        this.setData({ form, locked: !!results[1].payroll_batch_id })
        if (this.data.locked) wx.setNavigationBarTitle({ title: '工资关联流水' })
      } else if (!this.data.form.account_id && this.options.accounts.length) this.setData({ 'form.account_id': this.options.accounts[0].id })
      this.syncOptions()
      this.setData({ ready: true })
    } catch (error) { if (!this.destroyed) this.setData({ error: error.message }) }
  },
  syncOptions() {
    const accounts = this.options.accounts
    const categories = [{ id: 0, name: '未分类' }, ...this.options.categories.filter(v => v.direction === this.data.form.direction)]
    const departments = [{ id: 0, name: '不指定部门' }, ...this.options.departments]
    this.setData({ accounts, categories, departments, accountName: formUtil.pick(accounts, this.data.form.account_id, '请选择账户'), categoryName: formUtil.pick(categories, this.data.form.category_id), departmentName: formUtil.pick(departments, this.data.form.department_id) })
  },
  input(e) { if (!this.data.locked) formUtil.input.call(this, e) },
  direction(e) {
    if (!this.data.ready || this.data.saving || this.data.locked) return
    const value = e.currentTarget.dataset.value
    if (value === this.data.form.direction) return
    this.setData({ 'form.direction': value, 'form.category_id': 0 }); this.syncOptions(); formUtil.dirty(this)
  },
  select(e) {
    if (!this.data.ready || this.data.saving || this.data.locked) return
    const { kind, field } = e.currentTarget.dataset
    const selected = this.data[kind][Number(e.detail.value)]
    if (selected) { this.setData({ ['form.' + field]: selected.id }); this.syncOptions(); formUtil.dirty(this) }
  },
  create(e) {
    if (this.data.locked) return
    const kind = e.currentTarget.dataset.kind
    formUtil.createOption(this, kind, { accounts: '新建资金账户', categories: '新建收支分类', departments: '新建部门' }[kind], { direction: this.data.form.direction })
  },
  async save() {
    if (!this.data.ready || this.data.saving || this.data.locked) return
    const form = { ...this.data.form, summary: this.data.form.summary.trim(), amount: this.data.form.amount.trim() }
    let message = ''
    if (!formUtil.moneyValid(form.amount)) message = '请填写大于0的金额，最多2位小数'
    else if (!form.summary) message = '请填写这笔收支的摘要'
    else if (!this.options.accounts.some(v => v.id === form.account_id)) message = '请选择有效账户；暂无账户时请先新建'
    if (message) { api.errorModal(new Error(message)); return }
    this.setData({ saving: true })
    try {
      await api.request('/ledger' + (this.data.id ? '?id=' + this.data.id : ''), { method: this.data.id ? 'PUT' : 'POST', data: form })
      formUtil.finished(this, '/pages/ledger/index')
    } catch (error) { api.errorModal(error) }
    finally { this.setData({ saving: false }) }
  },
  back() { formUtil.back('/pages/ledger/index') }
})
