import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { authApi, settingsApi } from '@/api/modules'

export const useUserStore = defineStore('user', () => {
  const token = ref<string>(localStorage.getItem('access_token') || '')
  const userInfo = ref<any>(null)

  // 接单开关(持久化到 localStorage,初始乐观值;登录后从 settings API 拉真实值)
  const acceptOrders = ref<boolean>(localStorage.getItem('accept_orders') !== 'false')

  const isLoggedIn = computed(() => !!token.value)

  async function login(username: string, password: string) {
    const resp: any = await authApi.login({ username, password })
    token.value = resp.data.access_token
    userInfo.value = resp.data.user
    localStorage.setItem('access_token', resp.data.access_token)
    return resp.data
  }

  async function fetchMe() {
    if (!token.value) return null
    try {
      const resp: any = await authApi.me()
      userInfo.value = resp.data
      return userInfo.value
    } catch { logout(); return null }
  }

  // 拉接单开关(从后端权威值覆盖本地)
  async function fetchSettings() {
    if (!token.value) return
    try {
      const resp: any = await settingsApi.get()
      acceptOrders.value = resp.data.accept_orders
      localStorage.setItem('accept_orders', String(resp.data.accept_orders))
    } catch {}
  }

  // 切换接单开关(同时落库 + Hub 缓存)
  async function setAcceptOrders(val: boolean) {
    acceptOrders.value = val
    localStorage.setItem('accept_orders', String(val))
    try {
      await settingsApi.update({ accept_orders: val })
    } catch (err) {
      // 回滚
      acceptOrders.value = !val
      localStorage.setItem('accept_orders', String(!val))
      throw err
    }
  }

  function logout() {
    token.value = ''
    userInfo.value = null
    localStorage.removeItem('access_token')
  }

  return { token, userInfo, isLoggedIn, acceptOrders, login, fetchMe, fetchSettings, setAcceptOrders, logout }
})
