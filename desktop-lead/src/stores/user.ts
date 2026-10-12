import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { authApi } from '@/api/modules'

export const useUserStore = defineStore('user', () => {
  const token = ref<string>(localStorage.getItem('access_token') || '')
  const userInfo = ref<any>(null)

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
    } catch {
      logout()
      return null
    }
  }

  function logout() {
    token.value = ''
    userInfo.value = null
    localStorage.removeItem('access_token')
  }

  return { token, userInfo, isLoggedIn, login, fetchMe, logout }
})