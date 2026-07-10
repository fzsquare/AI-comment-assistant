import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import test from 'node:test'

const landingPageSource = readFileSync(new URL('./src/views/landing/LandingPage.vue', import.meta.url), 'utf8')
const mockSource = readFileSync(new URL('./src/api/mock.ts', import.meta.url), 'utf8')
const deeplinkSource = readFileSync(new URL('./src/utils/deeplink.ts', import.meta.url), 'utf8')

test('landing page prefers backend-resolved openUrl when opening platform links', () => {
  assert.match(landingPageSource, /openPlatform\(link\.platformCode, link\.openUrl \|\| link\.targetUrl, link\.backupUrl \|\| link\.targetUrl\)/)
})

test('mock landing payload returns the configured platform URL', () => {
  assert.match(mockSource, /openMode: 'official_link'/)
  assert.match(mockSource, /openUrl: link\.targetUrl \|\| link\.backupUrl \|\| ''/)
  assert.doesNotMatch(mockSource, /platformHomeAppUrl/)
})

test('app links do not automatically fall back to a web page', () => {
  assert.doesNotMatch(deeplinkSource, /window\.setTimeout/)
})
