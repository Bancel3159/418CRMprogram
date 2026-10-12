import axios, { type AxiosInstance, type AxiosRequestConfig } from 'axios'
import { createDiscreteApi } from 'naive-ui'

// 独立于组件的消息实例(可在 axios 拦截器等模块作用域中使用)
const { message } = createDiscreteApi(['message'])

const instance: AxiosInstance = axios.create({
  baseURL: '/api/v1',
  timeout: 15000,
  headers: {
    'Content-Type': 'application/json'
  }
})

// 请求拦截器：附加 JWT token
instance.interceptors.request.use(
  (config) => {
    const token = localStorage.getItem('access_token')
    if (token && config.headers) {
      config.headers.Authorization = `Bearer ${token}`
    }
    return config
  },
  (error) => Promise.reject(error)
)

// 响应拦截器：统一处理错误 + token 过期
instance.interceptors.response.use(
  (response) => {
    const data = response.data
    if (data && typeof data === 'object' && 'code' in data) {
      if (data.code !== 0) {
        message.error(data.message || '请求失败')
        return Promise.reject(new Error(data.message))
      }
      return data
    }
    return response.data
  },
  (error) => {
    if (error.response?.status === 401) {
      message.error('登录已过期，请重新登录')
      localStorage.removeItem('access_token')
      localStorage.removeItem('refresh_token')
      window.location.href = '/login'
    } else if (error.response?.status === 403) {
      message.error('权限不足')
    } else if (error.response?.status === 404) {
      message.error('资源不存在')
    } else if (error.response?.status >= 500) {
      message.error('服务器错误')
    } else {
      message.error(error.message || '网络错误')
    }
    return Promise.reject(error)
  }
)

export interface ApiResponse<T = any> {
  code: number
  message: string
  data: T
}

export interface PageData<T> {
  list: T[]
  total: number
  page: number
  page_size: number
}

export default instance
export type { AxiosRequestConfig }