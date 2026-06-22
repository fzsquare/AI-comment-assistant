import http from './http'

export const merchantApi = {
  login(payload: { account: string; password: string }) {
    return http.post('/merchant/auth/login', payload)
  },
  getStoreDetail() {
    return http.get('/merchant/store/detail')
  },
  updateStoreDetail(payload: Record<string, unknown>) {
    return http.put('/merchant/store/detail', payload)
  },
  listKeywords() {
    return http.get('/merchant/store/keywords')
  },
  createKeyword(payload: { keyword: string; sortNo: number }) {
    return http.post('/merchant/store/keywords', payload)
  },
  deleteKeyword(id: number) {
    return http.delete(`/merchant/store/keywords/${id}`)
  },
  listImages() {
    return http.get('/merchant/store/images')
  },
  createImage(payload: { imageUrl: string; thumbnailUrl: string; sortNo: number }) {
    return http.post('/merchant/store/images/upload', payload)
  },
  deleteImage(id: number) {
    return http.delete(`/merchant/store/images/${id}`)
  },
  listPlatformLinks() {
    return http.get('/merchant/store/platform-links')
  },
  createPlatformLink(payload: Record<string, unknown>) {
    return http.post('/merchant/store/platform-links', payload)
  },
  updatePlatformLinkStatus(id: number, status: number) {
    return http.put(`/merchant/store/platform-links/${id}/status`, { status })
  },
  deletePlatformLink(id: number) {
    return http.delete(`/merchant/store/platform-links/${id}`)
  },
  listReviews() {
    return http.get('/merchant/reviews')
  },
  createReview(payload: { content: string; status: string; platformCode: string }) {
    return http.post('/merchant/reviews', payload)
  },
  deleteReview(id: number) {
    return http.delete(`/merchant/reviews/${id}`)
  },
  generateReviews(platformCode: string, targetCount = 10) {
    return http.post('/merchant/reviews/generate', { targetCount, platformCode }, { timeout: 180000 })
  },
  listTasks() {
    return http.get('/merchant/review-generation-tasks')
  }
}
