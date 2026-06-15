import http from './http'

export const adminApi = {
  login(payload: { account: string; password: string }) {
    return http.post('/admin/auth/login', payload)
  },
  listMerchants() {
    return http.get('/admin/merchants')
  },
  updateMerchantStatus(id: number, status: number) {
    return http.put(`/admin/merchants/${id}/status`, { status })
  },
  listStores() {
    return http.get('/admin/stores')
  },
  updateStoreStatus(id: number, status: number) {
    return http.put(`/admin/stores/${id}/status`, { status })
  },
  listTags() {
    return http.get('/admin/nfc-tags')
  },
  createTag(payload: { tagCode?: string; remark?: string }) {
    return http.post('/admin/nfc-tags', payload)
  },
  bindTag(id: number, storeId: number) {
    return http.put(`/admin/nfc-tags/${id}/bind`, { storeId })
  },
  updateTagStatus(id: number, status: string) {
    return http.put(`/admin/nfc-tags/${id}/status`, { status })
  },
  listTasks() {
    return http.get('/admin/review-generation-tasks')
  },
  getStats() {
    return http.get('/admin/stats')
  }
}
