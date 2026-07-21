import { describe, expect, it } from 'vitest'
import { hasValidRoleSession, isTokenExpired, safeAdminRedirect, tokenExpiresAt } from './authSession'

function jwt(payload: Record<string, unknown>) {
  const encode = (value: object) => btoa(JSON.stringify(value)).replace(/=/g, '').replace(/\+/g, '-').replace(/\//g, '_')
  return `${encode({ alg: 'none' })}.${encode(payload)}.signature`
}

describe('admin auth session helpers', () => {
  it('detects an expired JWT while allowing opaque mock tokens', () => {
    const expired = jwt({ exp: 100 })
    expect(tokenExpiresAt(expired)).toBe(100_000)
    expect(isTokenExpired(expired, 100_001)).toBe(true)
    expect(isTokenExpired('mock-admin-token', 100_001)).toBe(false)
  })

  it('requires the expected role and a non-expired token', () => {
    const active = jwt({ exp: Math.floor(Date.now() / 1000) + 60 })
    expect(hasValidRoleSession(active, 'admin', 'admin')).toBe(true)
    expect(hasValidRoleSession(active, 'merchant', 'admin')).toBe(false)
    expect(hasValidRoleSession(jwt({ exp: 50 }), 'admin', 'admin')).toBe(false)
  })

  it('only restores internal admin console routes after login', () => {
    expect(safeAdminRedirect('/admin/console/stores?status=inactive')).toBe('/admin/console/stores?status=inactive')
    expect(safeAdminRedirect('https://attacker.example')).toBe('/admin/console/overview')
    expect(safeAdminRedirect('//attacker.example')).toBe('/admin/console/overview')
  })
})
