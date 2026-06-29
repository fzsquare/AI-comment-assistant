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
  listStoreTypes() {
    return http.get('/admin/store-types')
  },
  createStoreType(payload: { name: string; industryCode: string }) {
    return http.post('/admin/store-types', payload)
  },
  listStores() {
    return http.get('/admin/stores')
  },
  createStore(payload: {
    account: string
    password: string
    merchantName?: string
    contactName?: string
    typeId: number
    storeName: string
    storeIntro?: string
    address?: string
    primaryPlatformStyle?: string
    brandTone?: string
  }) {
    return http.post('/admin/stores', payload)
  },
  updateStoreStatus(id: number, status: number) {
    return http.put(`/admin/stores/${id}/status`, { status })
  },
  listTags(storeId?: number) {
    return http.get('/admin/nfc-tags', storeId ? { params: { storeId } } : undefined)
  },
  createTag(payload: { tagCode?: string; remark?: string; storeId?: number }) {
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
