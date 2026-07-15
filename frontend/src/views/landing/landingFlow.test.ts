import { beforeEach, describe, expect, it } from 'vitest'
import {
  ensureLandingSession,
  markLandingPageViewed,
  readLandingSession,
  selectLandingPlatform
} from './landingFlow'

describe('landing session', () => {
  beforeEach(() => {
    sessionStorage.clear()
  })

  it('reuses the existing session for the same landing token', () => {
    const first = ensureLandingSession('store-a', 'session-1')
    const second = ensureLandingSession('store-a', 'session-2')

    expect(first.sessionId).toBe('session-1')
    expect(second.sessionId).toBe('session-1')
  })

  it('keeps platform selection and page-view state bound to one token', () => {
    ensureLandingSession('store-a', 'session-1')
    selectLandingPlatform('store-a', 'meituan')
    markLandingPageViewed('store-a')

    expect(readLandingSession('store-a')).toMatchObject({
      token: 'store-a',
      sessionId: 'session-1',
      selectedPlatformCode: 'meituan',
      pageViewTracked: true
    })
    expect(readLandingSession('store-b')).toBeNull()
  })

  it('recovers from malformed session storage', () => {
    sessionStorage.setItem('ppk-landing-session:store-a', '{bad-json')

    expect(readLandingSession('store-a')).toBeNull()
    expect(sessionStorage.getItem('ppk-landing-session:store-a')).toBeNull()
  })
})
