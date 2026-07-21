import type { Role } from '../stores/auth'

function decodeBase64Url(value: string) {
  const normalized = value.replace(/-/g, '+').replace(/_/g, '/')
  const padded = normalized.padEnd(Math.ceil(normalized.length / 4) * 4, '=')
  return atob(padded)
}

export function tokenExpiresAt(token: string): number | null {
  const parts = token.split('.')
  if (parts.length !== 3) return null
  try {
    const payload = JSON.parse(decodeBase64Url(parts[1])) as { exp?: unknown }
    return typeof payload.exp === 'number' && Number.isFinite(payload.exp) ? payload.exp * 1000 : null
  } catch {
    return null
  }
}

export function isTokenExpired(token: string, now = Date.now()) {
  const expiresAt = tokenExpiresAt(token)
  return expiresAt !== null && expiresAt <= now
}

export function hasValidRoleSession(token: string, role: Role, expectedRole: Exclude<Role, ''>) {
  return Boolean(token) && role === expectedRole && !isTokenExpired(token)
}

export function safeAdminRedirect(value: unknown, fallback = '/admin/console/overview') {
  const target = typeof value === 'string' ? value.trim() : ''
  if (!target.startsWith('/admin/console') || target.startsWith('//')) return fallback
  return target
}
