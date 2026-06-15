import { defineStore } from 'pinia'

type Role = 'merchant' | 'admin' | ''

export const useAuthStore = defineStore('auth', {
  state: () => ({
    token: localStorage.getItem('ppk-token') || '',
    role: (localStorage.getItem('ppk-role') as Role) || ''
  }),
  actions: {
    setAuth(token: string, role: Role) {
      this.token = token
      this.role = role
      localStorage.setItem('ppk-token', token)
      localStorage.setItem('ppk-role', role)
    },
    clear() {
      this.token = ''
      this.role = ''
      localStorage.removeItem('ppk-token')
      localStorage.removeItem('ppk-role')
    }
  }
})
