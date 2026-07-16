import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import test from 'node:test'

const deploySource = readFileSync(new URL('../scripts/deploy.sh', import.meta.url), 'utf8')

test('deployment validates the refactored admin console instead of the legacy title', () => {
  assert.match(deploySource, /"评价助手"/)
  assert.match(deploySource, /"运营总览"/)
  assert.match(deploySource, /"商家与门店"/)
  assert.doesNotMatch(deploySource, /"商家运营控制台"/)
})
