import type { AxiosRequestConfig } from 'axios'
import http from './http'

export type LandingKeyword = { id: number; keyword: string }

export type LandingImage = {
  id: number
  imageUrl?: string
  url?: string
  thumbnailUrl?: string
}

export type LandingPlatformLink = {
  id: number
  platformCode: string
  platformName: string
  buttonText: string
  targetUrl: string
  backupUrl?: string
  openUrl?: string
  openMode?: string
}

export type LandingReview = {
  id: number
  content: string
  platformStyle?: string
}

export type LandingData = {
  sessionId: string
  storeName: string
  primaryPlatformStyle: string
  review?: LandingReview | null
  keywords: LandingKeyword[]
  images: LandingImage[]
  platformLinks: LandingPlatformLink[]
  remainingDispatchableCount: number
}

type ApiEnvelope<T> = { code: number; message: string; data: T }

const landingEventRequestConfig: AxiosRequestConfig | undefined = import.meta.env.VITE_USE_MOCK === 'true'
  ? undefined
  : { adapter: 'fetch', fetchOptions: { keepalive: true } }

export const publicApi = {
  initLanding(token: string) {
    return http.get<ApiEnvelope<LandingData>>(`/public/landing/${token}/init`)
  },
  switchReview(token: string, payload: { platformCode: string; tag?: string; sessionId?: string }) {
    return http.post<ApiEnvelope<{ review: LandingReview; remainingDispatchableCount: number }>>(`/public/landing/${token}/switch-review`, payload)
  },
  createEvent(token: string, payload: Record<string, unknown>) {
    return http.post<ApiEnvelope<{ saved: boolean }>>(`/public/landing/${token}/events`, payload, landingEventRequestConfig)
  }
}
