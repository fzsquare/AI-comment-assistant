import { publicApi } from '../../api/public'

export type LandingSession = {
  token: string
  sessionId: string
  selectedPlatformCode: string
  pageViewTracked: boolean
  createdAt: number
}

const landingSessionRefreshAfterMs = 23 * 60 * 60 * 1000

function sessionKey(token: string) {
  return `ppk-landing-session:${token}`
}

function writeLandingSession(session: LandingSession) {
  sessionStorage.setItem(sessionKey(session.token), JSON.stringify(session))
  return session
}

export function discardLandingSession(token: string, sessionId = '') {
  const key = sessionKey(token)
  if (!sessionId) {
    sessionStorage.removeItem(key)
    return
  }
  const raw = sessionStorage.getItem(key)
  if (!raw) return
  try {
    const parsed = JSON.parse(raw) as Partial<LandingSession>
    if (parsed.sessionId === sessionId) sessionStorage.removeItem(key)
  } catch {
    sessionStorage.removeItem(key)
  }
}

export function isLandingSessionError(error: unknown) {
  const message = String((error as any)?.response?.data?.message || '')
  return message.includes('会话已失效')
}

export function readLandingSession(token: string): LandingSession | null {
  const key = sessionKey(token)
  const raw = sessionStorage.getItem(key)
  if (!raw) return null
  try {
    const parsed = JSON.parse(raw) as Partial<LandingSession>
    const createdAt = typeof parsed.createdAt === 'number' ? parsed.createdAt : null
    const age = createdAt === null ? Number.POSITIVE_INFINITY : Date.now() - createdAt
    if (parsed.token !== token || typeof parsed.sessionId !== 'string' || !parsed.sessionId.trim() || createdAt === null || age < 0 || age >= landingSessionRefreshAfterMs) {
      sessionStorage.removeItem(key)
      return null
    }
    return {
      token,
      sessionId: parsed.sessionId,
      selectedPlatformCode: typeof parsed.selectedPlatformCode === 'string' ? parsed.selectedPlatformCode : '',
      pageViewTracked: parsed.pageViewTracked === true,
      createdAt
    }
  } catch {
    sessionStorage.removeItem(key)
    return null
  }
}

export function ensureLandingSession(token: string, sessionId: string) {
  const existing = readLandingSession(token)
  if (existing) return existing
  return writeLandingSession({ token, sessionId, selectedPlatformCode: '', pageViewTracked: false, createdAt: Date.now() })
}

export function selectLandingPlatform(token: string, platformCode: string) {
  const session = readLandingSession(token)
  if (!session) return null
  return writeLandingSession({ ...session, selectedPlatformCode: platformCode })
}

export function markLandingPageViewed(token: string) {
  const session = readLandingSession(token)
  if (!session) return null
  return writeLandingSession({ ...session, pageViewTracked: true })
}

export async function trackLandingEvent(
  token: string,
  sessionId: string,
  event: Record<string, unknown>
) {
  try {
    await publicApi.createEvent(token, {
      ...event,
      sessionId,
      clientUserAgent: navigator.userAgent || ''
    })
    return true
  } catch (error) {
    if (isLandingSessionError(error)) discardLandingSession(token, sessionId)
    console.warn('event tracking failed', error)
    return false
  }
}
