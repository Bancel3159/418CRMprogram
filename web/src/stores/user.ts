import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type { User, Role } from '@/api/modules'
import { authApi } from '@/api/modules'

export const useUserStore = defineStore('user', () => {
  const token = ref<string>(localStorage.getItem('access_token') || '')
  const refreshToken = ref<string>(localStorage.getItem('refresh_token') || '')
  const userInfo = ref<User | null>(null)

  const isLoggedIn = computed(() => !!token.value)
  const isAdmin = computed(() => userInfo.value?.role === 'admin')
  const isManager = computed(() =>
    ['admin', 'a_manager', 'b_manager'].includes(userInfo.value?.role || '')
  )
  const isLead = computed(() =>
    ['admin', 'a_manager', 'a_lead', 'b_manager', 'b_lead'].includes(userInfo.value?.role || '')
  )
  const roleLabel = computed(() => {
    const labels: Record<Role, string> = {
      admin: '超级管理员',
      a_manager: '前端主管',
      a_lead: '前端组长',
      a_staff: '前端员工',
      b_manager: '后端主管',
      b_lead: '后端组长',
      b_staff: '后端员工'
    }
    return userInfo.value ? labels[userInfo.value.role] || '-' : '-'
  })

  async function login(username: string, password: string, turnstileToken = '') {
    const resp: any = await authApi.login({ username, password, turnstile_token: turnstileToken })
    const data = resp.data
    token.value = data.access_token
    refreshToken.value = data.refresh_token
    localStorage.setItem('access_token', data.access_token)
    localStorage.setItem('refresh_token', data.refresh_token)
    userInfo.value = data.user
    return data
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
    refreshToken.value = ''
    userInfo.value = null
    localStorage.removeItem('access_token')
    localStorage.removeItem('refresh_token')
  }

  return {
    token,
    refreshToken,
    userInfo,
    isLoggedIn,
    isAdmin,
    isManager,
    isLead,
    roleLabel,
    login,
    fetchMe,
    logout
  }
})