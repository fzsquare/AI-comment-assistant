import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import test from 'node:test'

const landingPageSource = readFileSync(new URL('./src/views/landing/LandingPage.vue', import.meta.url), 'utf8')

test('landing page prefers backend-resolved openUrl when opening platform links', () => {
  assert.match(landingPageSource, /openPlatform\(link\.platformCode, link\.openUrl \|\| link\.targetUrl, link\.backupUrl\)/)
})
