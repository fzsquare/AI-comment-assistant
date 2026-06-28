import axios from 'axios'
import { mockAdapter } from './mock'

const TOKEN_KEY = 'ppk-token'
const ROLE_KEY = 'ppk-role'
const apiBaseURL = import.meta.env.VITE_API_BASE_URL || '/api'
const useMock = import.meta.env.VITE_USE_MOCK === 'true'

const http = axios.create({
  baseURL: apiBaseURL,
  timeout: 10000,
  // Mock 模式：用自定义 adapter 拦截全部请求，脱离后端独立调试
  ...(useMock ? { adapter: mockAdapter } : {})
})

if (useMock) {
  // 调试可见的提示
  console.info('%c[MOCK] 前端运行在 Mock 模式，未连接真实后端', 'color:#fff;background:#f59e0b;padding:2px 6px;border-radius:4px')
}

http.interceptors.request.use((config) => {
  const token = localStorage.getItem(TOKEN_KEY)
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

http.interceptors.response.use(
  (response) => response,
  (error) => {
    const status = error?.response?.status
    if (status === 401 || status === 403) {
      const role = localStorage.getItem(ROLE_KEY)
      localStorage.removeItem(TOKEN_KEY)
      localStorage.removeItem(ROLE_KEY)

      const currentPath = window.location.pathname
      const loginPath = role === 'admin' || currentPath.startsWith('/admin') ? '/admin/login' : '/merchant/login'
      if (!currentPath.endsWith('/login')) {
        window.location.assign(loginPath)
      }
    }
    return Promise.reject(error)
  }
)

export default http
