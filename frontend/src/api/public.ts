import http from './http'

export const publicApi = {
  initLanding(token: string) {
    return http.get(`/public/landing/${token}/init`)
  },
  switchReview(token: string, payload: { tag?: string; sessionId?: string }) {
    return http.post(`/public/landing/${token}/switch-review`, payload)
  },
  createEvent(token: string, payload: Record<string, unknown>) {
    return http.post(`/public/landing/${token}/events`, payload)
  }
}
