import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import test from 'node:test'

const deploySource = readFileSync(new URL('../scripts/deploy.sh', import.meta.url), 'utf8')

test('upgrade mode owns the safe existing-database configuration', () => {
  assert.match(deploySource, /upgrade\)\n/)
  assert.match(deploySource, /INIT_DB="false"/)
  assert.match(deploySource, /LOAD_SEED="false"/)
  assert.match(deploySource, /MIGRATE_DB="true"/)
})

test('upgrade mode backs up the database before applying migrations', () => {
  const backupDefinition = deploySource.indexOf('backup_database()')
  const backupCall = deploySource.indexOf('backup_database', backupDefinition + 1)
  const migrationCall = deploySource.indexOf('apply_migrations', backupCall)

  assert.notEqual(backupDefinition, -1)
  assert.notEqual(backupCall, -1)
  assert.notEqual(migrationCall, -1)
  assert.ok(backupCall < migrationCall)
})

test('upgrade mode keeps the historical DATETIME audit gate', () => {
  assert.match(deploySource, /require_historical_datetime_timezone_audit/)
  assert.match(deploySource, /HISTORICAL_DATETIME_TIMEZONE_AUDITED/)
})
