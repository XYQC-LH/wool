// lib/api/client.ts
// Axios 实例配置和拦截器

import axios, { AxiosInstance } from 'axios'
import Cookies from 'js-cookie'

export const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080'
const AUTH_STORAGE_KEY = 'auth-storage'

// 创建 axios 实例
export const api: AxiosInstance = axios.create({
  baseURL: API_BASE_URL,
  timeout: 30000,
  headers: {
    'Content-Type': 'application/json',
  },
})

// 请求拦截器 - 自动添加 JWT Token
api.interceptors.request.use(
  (config) => {
    const token = Cookies.get('token')
    if (token) {
      config.headers.Authorization = `Bearer ${token}`
    }
    return config
  },
  (error) => Promise.reject(error)
)

// 响应拦截器 - 统一处理 401 错误
api.interceptors.response.use(
  (response) => response.data,
  (error) => {
    if (error.response?.status === 401) {
      // Token 过期，清除登录状态
      Cookies.remove('token')
      try {
        if (typeof window !== 'undefined') {
          window.localStorage.removeItem(AUTH_STORAGE_KEY)
          window.dispatchEvent(new Event('auth:logout'))
        }
      } catch {
        // 忽略错误
      }
    }
    return Promise.reject(error)
  }
)
