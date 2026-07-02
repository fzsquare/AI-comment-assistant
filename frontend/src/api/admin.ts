import http from './http'
import type { DeviceStats } from './merchant'

export type AdminStats = {
  merchantCount: number
  enabledMerchantCount: number
  disabledMerchantCount: number
  currentWeekNewMerchants: number
  currentMonthNewMerchants: number
  storeCount: number
  enabledStoreCount: number
  disabledStoreCount: number
  tagCount: number
  taskCount: number
  crawlEnabledStoreCount: number
  crawlFailedStoreCount: number
  crawlDataAccumulatingCount: number
  totalCustomerVisits: number
  currentWeekCustomerVisits: number
  currentMonthCustomerVisits: number
  totalPublishClicks: number
  currentWeekPublishClicks: number
  currentMonthPublishClicks: number
  deviceStats: DeviceStats
  updatedAt: string
}

export type AdminStoreAnalytics = {
  totalCustomerVisits: number
  currentWeekCustomerVisits: number
  currentMonthCustomerVisits: number
  totalPublishClicks: number
  currentWeekPublishClicks: number
  currentMonthPublishClicks: number
  activePlatformLinkCount: number
  deviceStats: DeviceStats
}

export type AdminStoreReviewCrawl = {
  platformCode: string
  externalShopId: string
  enabled: boolean
  baselineCompletedAt?: string
  lastCrawledAt?: string
  nextCrawlAt?: string
  lastStatus: string
  lastErrorMessage?: string
}

export type AdminStore = {
  id: number
  merchantUserId: number
  uuid: string
  typeId?: number
  storeName: string
  industryType?: string
  storeIntro?: string
  address?: string
  primaryPlatformStyle: string
  brandTone?: string
  status: number
  createdAt?: string
  updatedAt?: string
  merchantAccount?: string
  merchantName?: string
  contactName?: string
  platformUrl?: string
  landingUrl?: string
  analytics?: AdminStoreAnalytics
  reviewCrawl?: AdminStoreReviewCrawl
}

export type AdminStorePayload = {
  account: string
  password?: string
  merchantName?: string
  contactName?: string
  typeId: number
  storeName: string
  storeIntro?: string
  address?: string
  primaryPlatformStyle?: string
  brandTone?: string
  platformUrl?: string
  reviewCrawlPlatformCode?: string
  reviewCrawlExternalShopId?: string
  reviewCrawlEnabled?: boolean
}

export type ReviewCrawlBatch = {
  id: number
  configId: number
  storeId: number
  platformCode: string
  externalShopIdSnapshot: string
  triggerType: string
  attemptNo: number
  isBaseline: boolean
  windowDays: number
  startedAt?: string
  finishedAt?: string
  status: string
  rawRowCount: number
  insertedRowCount: number
  matchedReviewCount: number
  errorMessage?: string
}

export type ExternalStoreReviewMatch = {
  id: number
  batchId: number
  storeId: number
  platformCode: string
  sourceReviewRef?: string
  userName?: string
  ratingRaw?: string
  reviewTime?: string
  content?: string
  matchedFeedbackId?: number
  matchedReviewItemId?: number
  matchScore: number
  matchReason: string
  matchSource: string
}

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
  deleteMerchant(id: number) {
    return http.delete(`/admin/merchants/${id}`)
  },
  listStoreTypes() {
    return http.get('/admin/store-types')
  },
  createStoreType(payload: { name: string; industryCode: string }) {
    return http.post('/admin/store-types', payload)
  },
  listStores() {
    return http.get<{ code: number; message: string; data: AdminStore[] }>('/admin/stores')
  },
  createStore(payload: AdminStorePayload & { password: string }) {
    return http.post('/admin/stores', payload)
  },
  updateStore(id: number, payload: AdminStorePayload) {
    return http.put(`/admin/stores/${id}`, payload)
  },
  updateStoreStatus(id: number, status: number) {
    return http.put(`/admin/stores/${id}/status`, { status })
  },
  deleteStore(id: number) {
    return http.delete(`/admin/stores/${id}`)
  },
  runStoreReviewCrawl(id: number) {
    return http.post(`/admin/stores/${id}/review-crawl/run`)
  },
  listStoreReviewCrawlBatches(id: number) {
    return http.get<{ code: number; message: string; data: ReviewCrawlBatch[] }>(`/admin/stores/${id}/review-crawl/batches`)
  },
  listStoreReviewCrawlMatches(id: number) {
    return http.get<{ code: number; message: string; data: ExternalStoreReviewMatch[] }>(`/admin/stores/${id}/review-crawl/matches`)
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
    return http.get<{ code: number; message: string; data: AdminStats }>('/admin/stats')
  }
}
