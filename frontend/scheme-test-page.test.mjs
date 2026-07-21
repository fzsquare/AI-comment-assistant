import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import test from 'node:test'

const routerSource = readFileSync(new URL('./src/router/index.ts', import.meta.url), 'utf8')
const pageSource = readFileSync(new URL('./src/views/SchemeTestPage.vue', import.meta.url), 'utf8')
const deploySource = readFileSync(new URL('../scripts/deploy.sh', import.meta.url), 'utf8')

test('scheme test page is publicly reachable at /scheme-test', () => {
  assert.match(routerSource, /path: '\/scheme-test', component: SchemeTestPage/)
  assert.match(deploySource, /SMOKE_SPA_ROUTES=.*\/scheme-test/)
})

test('scheme test page provides platform schemes and foreground detection', () => {
  assert.match(pageSource, /imeituan:\/\//)
  assert.match(pageSource, /dianping:\/\//)
  assert.match(pageSource, /snssdk1128:\/\//)
  assert.match(pageSource, /xhsdiscover:\/\//)
  assert.match(pageSource, /visibilitychange/)
  assert.match(pageSource, /pagehide/)
})
