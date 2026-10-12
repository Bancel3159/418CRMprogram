import axios, { type AxiosInstance } from 'axios'

const instance: AxiosInstance = axios.create({
  baseURL: '/api/v1',
  timeout: 15000,
  headers: { 'Content-Type': 'application/json' }
})

instance.interceptors.request.use((config) => {
  const token = localStorage.getItem('access_token')
  if (token && config.headers) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

instance.interceptors.response.use(
  (response) => {
    const data = response.data
    if (data && typeof data === 'object' && 'code' in data) {
      if (data.code !== 0) {
        return Promise.reject(new Error(data.message))
      }
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

// 游戏/字典/线索（拉新端用到的）
export const gameApi = {
  list: () => instance.get('/games', { params: { page: 1, page_size: 100 } })
}
export const dictApi = {
  list: (params: any) => instance.get('/dicts', { params })
}
export const leadApi = {
  create: (data: any) => instance.post('/leads', data)
}
export const uploadApi = {
  upload: (file: File, subDir = 'screenshots') => {
    const form = new FormData()
    form.append('files', file)
    form.append('sub_dir', subDir)
    return instance.post('/upload', form, { headers: { 'Content-Type': 'multipart/form-data' } })
  }
}

// 提交线索的扩展字段(由 Web 后台配置,A/B 员工拉新时按需展示)
// 后端在返回 select 类型且设置了 dict_category 的字段时,会把字典选项灌到 options 里
// (格式: [{label, value}, ...]),前端无需另查字典
export interface ExtensionField {
  id: number
  code: string
  label: string
  type: 'text' | 'textarea' | 'number' | 'select' | 'date'
  required: boolean
  enabled: boolean
  sort: number
  options: any
  dict_category: string
  placeholder: string
  remark: string
}
export const extensionFieldApi = {
  list: () => instance.get('/lead-extension-fields')
}

// A 员工拉新数据看板(支持 scope:self / all / specific)
// mine: 仅自己(任何角色都行)
// team: 主管/组长/admin 专用,可选全员 / 某个员工
// teamMembers: A 部成员 + 各自拉新数(做下拉筛选用)
export const dashboardApi = {
  mine: (days = 30) => instance.get('/stats/mine', { params: { days } }),
  team: (params: { days?: number; scope?: 'all'; creator_id?: number }) =>
    instance.get('/stats/team', { params }),
  teamMembers: (days = 30) =>
    instance.get('/stats/team/members', { params: { days } })
}