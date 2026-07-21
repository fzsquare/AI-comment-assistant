import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import test from 'node:test'

const deploySource = readFileSync(new URL('../scripts/deploy.sh', import.meta.url), 'utf8')
const envExample = readFileSync(new URL('../.env.deploy.example', import.meta.url), 'utf8')
const deployGuide = readFileSync(new URL('../README.md', import.meta.url), 'utf8')

test('existing review logs require an explicit historical DATETIME timezone audit', () => {
  assert.match(deploySource, /require_historical_datetime_timezone_audit/)
  assert.match(deploySource, /SELECT EXISTS\(SELECT 1 FROM review_display_logs LIMIT 1\)/)
  assert.match(deploySource, /HISTORICAL_DATETIME_TIMEZONE_AUDITED/)
  assert.match(envExample, /HISTORICAL_DATETIME_TIMEZONE_AUDITED=false/)
  assert.match(deployGuide, /HISTORICAL_DATETIME_TIMEZONE_AUDITED=true/)
  assert.match(deployGuide, /loc=Asia%2FShanghai/)
})
