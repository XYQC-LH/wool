// lib/api/gateway-client.ts
// Gateway API 专用客户端（使用 API Key 认证）

import axios, { AxiosInstance, InternalAxiosRequestConfig } from 'axios'
import { API_BASE_URL } from './client'

const GATEWAY_API_KEY_STORAGE_KEY = 'nexus_gateway_api_key'

// 获取 Gateway API Key
export function getGatewayApiKey(): string | null {
  if (typeof window === 'undefined') return null
  return window.localStorage.getItem(GATEWAY_API_KEY_STORAGE_KEY)
}

// 设置 Gateway API Key
export function setGatewayApiKey(apiKey: string | null): void {
  if (typeof window === 'undefined') return
  if (!apiKey) {
    window.localStorage.removeItem(GATEWAY_API_KEY_STORAGE_KEY)
    return
  }
  window.localStorage.setItem(GATEWAY_API_KEY_STORAGE_KEY, apiKey)
}

// 获取 Gateway 公开基础 URL
const GATEWAY_PUBLIC_BASE_URL_STORAGE_KEY = 'nexus_gateway_public_base_url'

export function getGatewayPublicBaseUrl(): string {
  const env = process.env.NEXT_PUBLIC_GATEWAY_PUBLIC_URL
  const fallback = `${API_BASE_URL}/v1`

  const normalize = (value: string) => value.trim().replace(/\/+$/, '')

  if (typeof window === 'undefined') return normalize(env || fallback)

  const stored = window.localStorage.getItem(GATEWAY_PUBLIC_BASE_URL_STORAGE_KEY)
  return normalize(stored || env || fallback)
}

// 设置 Gateway 公开基础 URL
export function setGatewayPublicBaseUrl(baseUrl: string | null): void {
  if (typeof window === 'undefined') return
  if (!baseUrl) {
    window.localStorage.removeItem(GATEWAY_PUBLIC_BASE_URL_STORAGE_KEY)
    return
  }
  const normalized = baseUrl.trim().replace(/\/+$/, '')
  window.localStorage.setItem(GATEWAY_PUBLIC_BASE_URL_STORAGE_KEY, normalized)
}

// 获取 Swagger 文档 URL
export function getSwaggerUrl(): string {
  const gatewayBaseUrl = getGatewayPublicBaseUrl()
  const base = gatewayBaseUrl.replace(/\/v1\/?$/, '')
  return `${base}/swagger/index.html`
}

// 创建 Gateway API 客户端
export const gatewayApi: AxiosInstance = axios.create({
  baseURL: API_BASE_URL,
  timeout: 30000,
  headers: {
    'Content-Type': 'application/json',
  },
})

// Gateway API 请求拦截器
// 注意：与其他拦截器分开定义，避免类型推断问题
gatewayApi.interceptors.request.use(
  (config: InternalAxiosRequestConfig) => {
    const apiKey = getGatewayApiKey()
    if (apiKey) {
      // 设置 Authorization header
      config.headers.set('Authorization', `Bearer ${apiKey}`)
    }
    return config
  },
  (error) => Promise.reject(error)
)

// Gateway API 响应拦截器
gatewayApi.interceptors.response.use(
  (response) => response.data,
  (error) => Promise.reject(error)
)
