import axios, { type AxiosAdapter } from 'axios'

const TOKEN_KEY = 'ppk-token'
const ROLE_KEY = 'ppk-role'
// 部署根路径（Vite base，如 /ppk/ 或 /）。API 与登录跳转都基于它，兼容子路径反代。
const APP_BASE = import.meta.env.BASE_URL || '/'
const apiBaseURL = import.meta.env.VITE_API_BASE_URL || APP_BASE.replace(/\/+$/, '') + '/api'
const useMock = import.meta.env.VITE_USE_MOCK === 'true'
const mockConfig: { adapter?: AxiosAdapter } = useMock
  ? {
      adapter: async (config) => {
        const { mockAdapter } = await import('./mock')
        return mockAdapter(config)
      }
    }
  : {}

function appRoute(path: string) {
  return `${APP_BASE.replace(/\/+$/, '')}/${path.replace(/^\/+/, '')}`
}

function routeStartsWith(currentPath: string, path: string) {
  const prefix = appRoute(path)
  return currentPath === prefix || currentPath.startsWith(`${prefix}/`)
}

function loginPathForCurrentEntry(currentPath: string) {
  if (routeStartsWith(currentPath, 'admin')) return appRoute('admin/login')
  if (routeStartsWith(currentPath, 'merchant')) return appRoute('merchant/login')
  return ''
}

function loginURLForCurrentEntry() {
  const loginPath = loginPathForCurrentEntry(window.location.pathname)
  if (!loginPath) return ''
  const normalizedBase = APP_BASE.replace(/\/+$/, '')
  const routePath = normalizedBase && window.location.pathname.startsWith(normalizedBase)
    ? window.location.pathname.slice(normalizedBase.length) || '/'
    : window.location.pathname
  const params = new URLSearchParams({
    reason: 'session_expired',
    redirect: `${routePath}${window.location.search}${window.location.hash}`
  })
  return `${loginPath}?${params.toString()}`
}

const http = axios.create({
  baseURL: apiBaseURL,
  timeout: 10000,
  // Mock 模式：用自定义 adapter 拦截全部请求，脱离后端独立调试
  ...mockConfig
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
      localStorage.removeItem(TOKEN_KEY)
      localStorage.removeItem(ROLE_KEY)

      const currentPath = window.location.pathname
      const loginURL = loginURLForCurrentEntry()
      if (loginURL && !currentPath.endsWith('/login')) {
        window.location.assign(loginURL)
      }
    }
    return Promise.reject(error)
  }
)

export default http
