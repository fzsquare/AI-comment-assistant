import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import {
  discardLandingSession,
  ensureLandingSession,
  markLandingPageViewed,
  readLandingSession,
  selectLandingPlatform
} from './landingFlow'

describe('landing session', () => {
  beforeEach(() => {
    sessionStorage.clear()
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2026-07-15T12:00:00+08:00'))
  })

  afterEach(() => vi.useRealTimers())

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

  it('replaces a session before the backend 24-hour expiry', () => {
    ensureLandingSession('store-a', 'session-1')
    markLandingPageViewed('store-a')
    const stored = JSON.parse(sessionStorage.getItem('ppk-landing-session:store-a') || '{}')
    sessionStorage.setItem('ppk-landing-session:store-a', JSON.stringify({ ...stored, createdAt: Date.now() - (23 * 60 * 60 * 1000 + 1000) }))
    expect(Date.now() - (readLandingSession('store-a')?.createdAt || 0)).toBeGreaterThan(23 * 60 * 60 * 1000)

    const refreshed = ensureLandingSession('store-a', 'session-2')

    expect(refreshed).toMatchObject({ sessionId: 'session-2', pageViewTracked: false, selectedPlatformCode: '' })
  })

  it('only discards the rejected session and preserves a newer replacement', () => {
    ensureLandingSession('store-a', 'session-1')
    discardLandingSession('store-a', 'another-session')
    expect(readLandingSession('store-a')?.sessionId).toBe('session-1')

    discardLandingSession('store-a', 'session-1')
    expect(readLandingSession('store-a')).toBeNull()
  })
})
