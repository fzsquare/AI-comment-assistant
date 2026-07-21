import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import test from 'node:test'

const routerSource = readFileSync(new URL('./src/router/index.ts', import.meta.url), 'utf8')
const adminConsoleSource = readFileSync(new URL('./src/views/admin/AdminConsole.vue', import.meta.url), 'utf8')

test('admin console uses section routes instead of a single anchor page', () => {
  assert.match(routerSource, /path:\s*['"]\/admin\/console['"][\s\S]*redirect:\s*['"]\/admin\/console\/overview['"]/)
  assert.match(routerSource, /path:\s*['"]\/admin\/console\/:section['"]/)
  assert.match(routerSource, /name:\s*['"]admin-console-section['"]/)
})

test('admin sidebar navigates with router links rather than hash anchors', () => {
  assert.match(adminConsoleSource, /<RouterLink/)
  assert.match(adminConsoleSource, /activeAdminSection/)
  assert.doesNotMatch(adminConsoleSource, /href:\s*['"]#/)
})
