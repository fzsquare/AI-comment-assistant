import type { ReviewGenerationAuditLog, ReviewGenerationTask } from '../api/merchant'

export function latestGenerationLog(task: ReviewGenerationTask): ReviewGenerationAuditLog | null {
  return task.auditLogs?.[0] || null
}

export function generationLogPreview(task: ReviewGenerationTask, limit = 3): ReviewGenerationAuditLog[] {
  return (task.auditLogs || []).slice(0, limit)
}

export function generationFailureReason(task: ReviewGenerationTask): string {
  const errorMessage = String(task.errorMessage || '').trim()
  if (errorMessage) return errorMessage
  if (task.status === 'partial_failed' && task.duplicateFilteredCount > 0) {
    return `重复过滤 ${task.duplicateFilteredCount} 条，剩余评价已入池`
  }
  const latest = latestGenerationLog(task)
  if (latest?.level === 'error' && latest.message) return latest.message
  return '-'
}

export function generationStageSummary(task: ReviewGenerationTask): string {
  const latest = latestGenerationLog(task)
  if (!latest) return '-'
  const duration = latest.durationMs > 0 ? ` · ${latest.durationMs}ms` : ''
  const status = latest.httpStatus > 0 ? ` · HTTP ${latest.httpStatus}` : ''
  return `${latest.stage}：${latest.message}${status}${duration}`
}
