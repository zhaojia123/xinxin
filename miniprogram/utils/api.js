const config = require('../config')
const TOKEN_KEY = 'xinxin_token'
const HOST_KEY = 'xinxin_develop_host'
let redirecting = false

function environment() {
  try { return wx.getAccountInfoSync().miniProgram.envVersion || 'release' } catch (_) { return 'release' }
}
function baseURL() {
  const env = environment()
  return (env === 'develop' ? (wx.getStorageSync(HOST_KEY) || config.develop) : config[env] || '').replace(/\/+$/, '')
}
function setDevelopHost(value) {
  if (environment() !== 'develop') throw new Error('仅开发版本可修改连接地址')
  const host = value.trim().replace(/\/+$/, '')
  if (!/^https?:\/\/[a-zA-Z0-9.-]+(?::\d+)?$/.test(host)) throw new Error('请填写服务器地址，例如 http://192.168.1.8:8999，不包含路径')
  wx.setStorageSync(HOST_KEY, host)
  wx.removeStorageSync(TOKEN_KEY)
}
function token() { return wx.getStorageSync(TOKEN_KEY) || '' }
function saveToken(value) { wx.setStorageSync(TOKEN_KEY, value) }
function clearToken() { wx.removeStorageSync(TOKEN_KEY) }
function toLogin() {
  clearToken()
  if (redirecting) return
  redirecting = true
  wx.reLaunch({ url: '/pages/login/index', complete: () => { redirecting = false } })
}
function guard() { if (token()) return true; toLogin(); return false }
function request(path, { method = 'GET', data, auth = true } = {}) {
  const host = baseURL()
  if (!host) return Promise.reject(new Error('尚未配置此版本的服务器地址，请先设置 config.js'))
  if (auth && !token()) { toLogin(); return Promise.reject(new Error('请先登录')) }
  return new Promise((resolve, reject) => {
    wx.request({
      url: host + '/api/mini' + path, method, data, timeout: 15000,
      header: { 'Content-Type': 'application/json', ...(auth ? { Authorization: 'Bearer ' + token() } : {}) },
      success(res) {
        if (res.statusCode >= 200 && res.statusCode < 300) { resolve(res.data); return }
        const message = res.data && typeof res.data.error === 'string' ? res.data.error : '请求失败，请稍后重试'
        // 后端错误保留日志定位信息，界面只显示业务描述。
        const error = new Error(message.replace(/^(?:[^：]*\.go:\d+：)+/, ''))
        error.status = res.statusCode
        if (auth && (res.statusCode === 401 || res.statusCode === 403)) toLogin()
        reject(error)
      },
      fail() { reject(new Error('连接失败，请检查服务是否启动、连接地址及开发者工具的域名校验设置')) }
    })
  })
}
function login() {
  return new Promise((resolve, reject) => wx.login({
    timeout: 10000,
    success: res => res.code ? resolve(res.code) : reject(new Error('微信未返回登录凭证，请重试')),
    fail: () => reject(new Error('微信登录失败，请确认开发者工具已登录且 AppID 正确'))
  })).then(code => request('/login', { method: 'POST', auth: false, data: { code } }))
    .then(result => { if (!result.token) throw new Error('服务端未返回登录令牌'); saveToken(result.token); return result })
}
function query(values) {
  return Object.keys(values).filter(k => values[k] !== '' && values[k] != null)
    .map(k => encodeURIComponent(k) + '=' + encodeURIComponent(values[k])).join('&')
}
function errorModal(error) { wx.showModal({ title: '未能完成', content: error.message || '请稍后重试', showCancel: false }) }
module.exports = { environment, baseURL, setDevelopHost, token, saveToken, clearToken, toLogin, guard, request, login, query, errorModal }
