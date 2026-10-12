import axios, { type AxiosInstance } from 'axios'

const instance: AxiosInstance = axios.create({
  baseURL: '/api/v1',
  timeout: 15000,
  headers: { 'Content-Type': 'application/json' }
})

instance.interceptors.request.use((config) => {
  const token = localStorage.getItem('access_token')
  if (token && config.headers) config.headers.Authorization = `Bearer ${token}`
  return config
})

instance.interceptors.response.use(
  (response) => {
    const data = response.data
    if (data && typeof data === 'object' && 'code' in data) {
      if (data.code !== 0) return Promise.reject(new Error(data.message))
      return data
    }
    return response.data
  },
  (error) => {
    if (error.response?.status === 401) {
      localStorage.removeItem('access_token')
      window.location.href = '/login'
    }
    return Promise.reject(error)
  }
)

export default instance

// 登录
export const authApi = {
  login: (data: any) => instance.post('/auth/login', data),
  me: () => instance.get('/auth/me')
}

// 线索
export const leadApi = {
  pool: (params: any) => instance.get('/leads/pool', { params }),
  list: (params: any) => instance.get('/leads', { params }),
  get: (id: number) => instance.get(`/leads/${id}`),
  claim: (id: number) => instance.post(`/leads/${id}/claim`),
  release: (id: number) => instance.post(`/leads/${id}/release`),
  archive: (id: number) => instance.post(`/leads/${id}/archive`),
  // 付费标记（admin / AB 主管 / 组长 可调）
  markPaid: (id: number) => instance.post(`/leads/${id}/mark-paid`),
  unmarkPaid: (id: number) => instance.post(`/leads/${id}/unmark-paid`)
}

// 个人设置（follow 端接单开关）
export const settingsApi = {
  get: () => instance.get('/users/me/settings'),
  update: (data: any) => instance.put('/users/me/settings', data)
}

// 玩家列表（只读，follow 端用于查询页）
export const playerApi = {
  list: (params: any) => instance.get('/players', { params })
}

// 订单列表（只读，follow 端用于查询页）
export const orderApi = {
  list: (params: any) => instance.get('/orders', { params })
}

// 统计（follow 端看板）
export const statsApi = {
  followMine: (params: any) => instance.get('/stats/follow/mine', { params })
}

// P11 跟进提醒(今日待办)
export const remindersApi = {
  today: (params: any) => instance.get('/leads/reminders/today', { params })
}

// P11 手动标记(主管 / admin)
export const followApi = {
  markLost: (id: number) => instance.post(`/leads/${id}/mark-lost`),
  markActive: (id: number) => instance.post(`/leads/${id}/mark-active`)
}

// P11 写跟进(任何登录用户可写自己的可见 lead)
export const followRecordApi = {
  create: (id: number, data: any) => instance.post(`/leads/${id}/follow`, data),
  // P9 兼容:历史记录 GET
  list: (id: number) => instance.get(`/leads/${id}/follow-records`)
}