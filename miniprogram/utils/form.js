const api = require('./api')

function today() {
  const d = new Date()
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`
}
function moneyValid(value, positive = true, digits = 12) {
  return new RegExp(`^(0|[1-9][0-9]{0,${digits - 1}})(\\.[0-9]{1,2})?$`).test(value) && (!positive || /[1-9]/.test(value))
}
function dirty(page) {
  if (!page.dirty) {
    page.dirty = true
    if (wx.enableAlertBeforeUnload) wx.enableAlertBeforeUnload({ message: '还有未保存的修改，确定离开吗？' })
  }
}
function input(e) {
  if (!this.data.ready || this.data.saving) return
  this.setData({ ['form.' + e.currentTarget.dataset.field]: e.detail.value })
  dirty(this)
}
function finished(page, tab, id) {
  page.dirty = false
  if (wx.disableAlertBeforeUnload) wx.disableAlertBeforeUnload()
  wx.showToast({ title: '保存成功', icon: 'success' })
  if (getCurrentPages().length > 1) wx.navigateBack()
  else wx.switchTab({ url: tab })
}
function back(tab) {
  if (getCurrentPages().length > 1) wx.navigateBack()
  else wx.switchTab({ url: tab })
}
function pick(options, id, empty = '未选择') {
  const item = options.find(v => v.id === id)
  return item ? item.name : (id ? '原选项已停用，请重新选择' : empty)
}
function createOption(page, kind, title, extra = {}) {
  if (!page.data.ready || page.data.saving || page.creatingOption) return
  page.creatingOption = true
  wx.showModal({
    title, editable: true, placeholderText: '填写名称，最多64字',
    success: async result => {
      if (!result.confirm) return
      const name = (result.content || '').trim()
      if (!name) { wx.showToast({ title: '名称不能为空', icon: 'none' }); return }
      try {
        const created = await api.request('/options', { method: 'POST', data: { kind, name, ...extra } })
        const options = await api.request('/options')
        page.options = options
        const field = { accounts: 'account_id', categories: 'category_id', departments: 'department_id', positions: 'position_id' }[kind]
        const form = { ...page.data.form, [field]: created.id }
        if (kind === 'departments' && 'position_id' in form) form.position_id = 0
        page.setData({ form })
        page.syncOptions()
        dirty(page)
      } catch (error) { api.errorModal(error) }
      finally { page.creatingOption = false }
    },
    fail: () => { page.creatingOption = false },
    complete: result => { if (!result.confirm) page.creatingOption = false }
  })
}
module.exports = { today, moneyValid, dirty, input, finished, back, pick, createOption }
