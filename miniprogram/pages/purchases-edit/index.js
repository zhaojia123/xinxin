const api = require('../../utils/api')
const formUtil = require('../../utils/form')
const categories = ['蔬菜', '调料', '肉类', '水产', '水果', '粮油', '其他'].map(name => ({ name }))
Page({
  data: { id: 0, ready: false, saving: false, error: '', categories, total: '0.00', form: { date: '', supplier_name: '默认供货商', category: '蔬菜', product_name: '', quantity: '1', unit: '斤', unit_price: '0.00', remark: '' } },
  onLoad(query) {
    if (!api.guard()) return
    const id = Number(query.id) || 0
    this.setData({ id, 'form.date': formUtil.today() })
    if (!api.hasPermission('purchases', id ? 'edit' : 'create')) {
      wx.showModal({ title: '无操作权限', content: '当前账号只有查看权限，不能编辑或新增采购。', showCancel: false, success: () => wx.navigateBack() })
      return
    }
    wx.setNavigationBarTitle({ title: id ? '编辑采购' : '新增采购' })
    this.load()
  },
  onUnload() { this.destroyed = true },
  async load() {
    this.setData({ ready: false, error: '' })
    try {
      if (this.data.id) {
        const item = await api.request('/purchases?id=' + this.data.id)
        if (this.destroyed) return
        const form = { date: item.date, supplier_name: item.supplier_name, category: item.category, product_name: item.product_name, quantity: item.quantity, unit: item.unit, unit_price: item.unit_price, remark: item.remark }
        this.setData({ form })
      }
      this.calculate()
      this.setData({ ready: true })
    } catch (error) { if (!this.destroyed) this.setData({ error: error.message }) }
  },
  input(e) { formUtil.input.call(this, e); const form = { ...this.data.form, [e.currentTarget.dataset.field]: e.detail.value }; this.calculate(form) },
  categoryChanged(e) { this.setData({ 'form.category': this.data.categories[Number(e.detail.value)].name }); formUtil.dirty(this) },
  calculate(form = this.data.form) {
    const quantity = Number(form.quantity) || 0
    const unitPrice = Number(form.unit_price) || 0
    this.setData({ total: (quantity * unitPrice).toFixed(2) })
  },
  async save() {
    if (!this.data.ready || this.data.saving) return
    const form = { ...this.data.form, supplier_name: this.data.form.supplier_name.trim(), product_name: this.data.form.product_name.trim(), quantity: this.data.form.quantity.trim(), unit_price: this.data.form.unit_price.trim() }
    let message = ''
    if (!form.date || !form.supplier_name || !form.category || !form.product_name) message = '请填写日期、供货商、品类和菜品名称'
    else if (!form.quantity || Number(form.quantity) <= 0) message = '采购数量必须大于0'
    else if (!formUtil.moneyValid(form.unit_price, false)) message = '请输入正确的单价，最多2位小数'
    else if (!form.unit) message = '请填写采购单位'
    if (message) { api.errorModal(new Error(message)); return }
    this.setData({ saving: true })
    try {
      await api.request('/purchases' + (this.data.id ? '?id=' + this.data.id : ''), { method: this.data.id ? 'PUT' : 'POST', data: form })
      formUtil.finished(this, '/pages/purchases/index')
    } catch (error) { api.errorModal(error) }
    finally { this.setData({ saving: false }) }
  },
  back() { formUtil.back('/pages/purchases/index') }
})
