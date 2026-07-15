import http from './http'

export type PublishTrendPoint = {
  weekStart?: string
  weekEnd?: string
  month?: string
  count: number
}

export type PublishStatsRange = '7d' | '30d'

export type PublishFunnelStage = {
  code: 'page_view' | 'platform_select' | 'review_copy' | 'platform_link_click'
  label: string
  count: number
  conversionRate: number
  conversionAvailable: boolean
  conversionLabel?: string
}

export type PublishDailyPoint = {
  date: string
  pageViews: number
  platformSelections: number
  reviewCopies: number
  platformLinkClicks: number
}

export type PublishStatsRecommendation = {
  code: string
  title: string
  message: string
  actionLabel: string
  actionTarget: string
}

export type DeviceBreakdownItem = {
  code: string
  label: string
  count: number
  percent: number
}

export type DeviceStats = {
  totalCount: number
  items: DeviceBreakdownItem[]
}

export type PublishStats = {
  range: PublishStatsRange
  rangeStart: string
  rangeEnd: string
  dataState: 'empty' | 'accumulating' | 'ready'
  uniqueSessions: number
  funnel: PublishFunnelStage[]
  dailySeries: PublishDailyPoint[]
  recommendation: PublishStatsRecommendation
  platformCode: string
  platformName: string
  totalPublishClicks: number
  currentWeekPublishClicks: number
  currentMonthPublishClicks: number
  previousWeekPublishClicks: number
  previousMonthPublishClicks: number
  publishWeekGrowthPercent: number
  publishMonthGrowthPercent: number
  totalCustomerVisits: number
  currentWeekCustomerVisits: number
  currentMonthCustomerVisits: number
  previousWeekCustomerVisits: number
  previousMonthCustomerVisits: number
  visitWeekGrowthPercent: number
  visitMonthGrowthPercent: number
  updatedAt: string
  dataSource: string
  dataSourceLabel: string
  timezone: string
  currentWeekStart: string
  currentWeekEnd: string
  currentMonthStart: string
  currentMonthEnd: string
  platformLinksConfigured: boolean
  activePlatformLinkCount: number
  crawlDataReady: boolean
  crawlDataMessage: string
  weeklyGuidedShareReady: boolean
  monthlyGuidedShareReady: boolean
  weeklyGuidedSharePercent: number
  monthlyGuidedSharePercent: number
  deviceStats: DeviceStats
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

export type ReviewGenerationAuditLog = {
  id: number
  taskId: number
  storeId: number
  platformStyle: string
  triggerType: string
  stage: string
  level: string
  status: string
  message: string
  detail?: string
  agentEndpoint?: string
  httpStatus: number
  durationMs: number
  targetCount: number
  generatedRawCount: number
  insertedRowCount: number
  duplicateFilteredCount: number
  createdAt?: string
}

export type ReviewGenerationTask = {
  id: number
  storeId: number
  platformStyle: string
  triggerType: string
  targetCount: number
  generatedRawCount: number
  insertedRowCount: number
  duplicateFilteredCount: number
  duplicateCheckVersion?: string
  successCount: number
  failedCount: number
  status: string
  errorMessage?: string
  createdAt?: string
  updatedAt?: string
  auditLogs?: ReviewGenerationAuditLog[]
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
  getPublishStats(platformCode = '', range: PublishStatsRange = '7d') {
    const params = { range, ...(platformCode ? { platformCode } : {}) }
    return http.get<{ code: number; message: string; data: PublishStats }>('/merchant/dashboard/publish-stats', { params })
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
    return http.get<{ code: number; message: string; data: ReviewGenerationTask[] }>('/merchant/review-generation-tasks')
  }
}
