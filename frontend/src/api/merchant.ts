import http from './http'

export type PublishTrendPoint = {
  weekStart?: string
  weekEnd?: string
  month?: string
  count: number
}

export type PublishStats = {
  totalPublishClicks: number
  currentWeekPublishClicks: number
  currentMonthPublishClicks: number
  updatedAt: string
  timezone: string
  currentWeekStart: string
  currentWeekEnd: string
  currentMonthStart: string
  currentMonthEnd: string
  platformLinksConfigured: boolean
  activePlatformLinkCount: number
  weeklySeries: PublishTrendPoint[]
  monthlySeries: PublishTrendPoint[]
  partialErrors: string[]
}

export type GenerationPreferences = {
  configured?: boolean
  focusKeywords: string[]
  styleCodes: string[]
  diversityDimensions: string[]
  referenceReviews: string[]
  lengthVariance: string
  updatedAt?: string
}

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
  getPublishStats() {
    return http.get<{ code: number; message: string; data: PublishStats }>('/merchant/dashboard/publish-stats')
  },
  listKeywords() {
    return http.get('/merchant/store/keywords')
  },
  getKeywordSuggestions() {
    return http.get('/merchant/store/keyword-suggestions')
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
  uploadImageFile(file: File) {
    const form = new FormData()
    form.append('file', file)
    return http.post('/merchant/store/images/upload-file', form)
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
  updatePlatformLink(id: number, payload: Record<string, unknown>) {
    return http.put(`/merchant/store/platform-links/${id}`, payload)
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
  getGenerationPreferences() {
    return http.get<{ code: number; message: string; data: GenerationPreferences }>('/merchant/review-generation-preferences')
  },
  saveGenerationPreferences(payload: GenerationPreferences) {
    return http.put<{ code: number; message: string; data: GenerationPreferences }>('/merchant/review-generation-preferences', payload)
  },
  generateReviews(platformCode: string, targetCount = 10) {
    return http.post('/merchant/reviews/generate', { targetCount, platformCode }, { timeout: 180000 })
  },
  listTasks() {
    return http.get('/merchant/review-generation-tasks')
  }
}
