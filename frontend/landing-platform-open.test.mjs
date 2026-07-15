import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import test from 'node:test'

const landingPageSource = readFileSync(new URL('./src/views/landing/LandingReviewPage.vue', import.meta.url), 'utf8')
const publicApiSource = readFileSync(new URL('./src/api/public.ts', import.meta.url), 'utf8')
const routerSource = readFileSync(new URL('./src/router/index.ts', import.meta.url), 'utf8')
const mockSource = readFileSync(new URL('./src/api/mock.ts', import.meta.url), 'utf8')
const deeplinkSource = readFileSync(new URL('./src/utils/deeplink.ts', import.meta.url), 'utf8')

test('landing page prefers backend-resolved openUrl when opening platform links', () => {
  assert.match(landingPageSource, /openPlatform\(link\.platformCode, link\.openUrl \|\| link\.targetUrl, link\.backupUrl \|\| link\.targetUrl\)/)
})

test('customer flow uses independent platform and review routes', () => {
  assert.match(routerSource, /path: '\/landing\/:token'.*LandingPlatformPage/)
  assert.match(routerSource, /path: '\/landing\/:token\/review\/:platformCode'.*LandingReviewPage/)
})

test('landing analytics use a non-blocking request that survives page navigation', () => {
  assert.match(publicApiSource, /adapter:\s*'fetch'/)
  assert.match(publicApiSource, /keepalive:\s*true/)
})

test('mock landing payload prefers verified app schemes and preserves official URLs', () => {
  assert.match(mockSource, /openUrl: platformHomeAppUrl\(link\.platformCode\) \|\| link\.targetUrl \|\| link\.backupUrl \|\| ''/)
  assert.match(mockSource, /openMode: platformHomeAppUrl\(link\.platformCode\) \? 'app_link' : 'official_link'/)
  assert.match(mockSource, /function platformHomeAppUrl/)
})

test('app links fall back to the configured official URL when the page stays visible', () => {
  assert.match(deeplinkSource, /window\.setTimeout/)
  assert.match(deeplinkSource, /visibilitychange/)
  assert.match(deeplinkSource, /pagehide/)
  assert.match(deeplinkSource, /fallbackUrl/)
})
