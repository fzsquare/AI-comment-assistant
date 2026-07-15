import { publicApi } from '../../api/public'

export type LandingSession = {
  token: string
  sessionId: string
  selectedPlatformCode: string
  pageViewTracked: boolean
}

function sessionKey(token: string) {
  return `ppk-landing-session:${token}`
}

function writeLandingSession(session: LandingSession) {
  sessionStorage.setItem(sessionKey(session.token), JSON.stringify(session))
  return session
}

export function readLandingSession(token: string): LandingSession | null {
  const key = sessionKey(token)
  const raw = sessionStorage.getItem(key)
  if (!raw) return null
  try {
    const parsed = JSON.parse(raw) as Partial<LandingSession>
    if (parsed.token !== token || typeof parsed.sessionId !== 'string' || !parsed.sessionId.trim()) {
      sessionStorage.removeItem(key)
      return null
    }
    return {
      token,
      sessionId: parsed.sessionId,
      selectedPlatformCode: typeof parsed.selectedPlatformCode === 'string' ? parsed.selectedPlatformCode : '',
      pageViewTracked: parsed.pageViewTracked === true
    }
  } catch {
    sessionStorage.removeItem(key)
    return null
  }
}

export function ensureLandingSession(token: string, sessionId: string) {
  const existing = readLandingSession(token)
  if (existing) return existing
  return writeLandingSession({ token, sessionId, selectedPlatformCode: '', pageViewTracked: false })
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
    console.warn('event tracking failed', error)
    return false
  }
}
