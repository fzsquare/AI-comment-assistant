import axios from 'axios'

const TOKEN_KEY = 'ppk-token'
const ROLE_KEY = 'ppk-role'
const apiBaseURL = import.meta.env.VITE_API_BASE_URL || '/api'

const http = axios.create({
  baseURL: apiBaseURL,
  timeout: 10000
})

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
